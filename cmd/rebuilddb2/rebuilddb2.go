package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime/pprof"
	"time"

	"github.com/btcsuite/btclog"
	"github.com/dcrdata/dcrdata/db/dbtypes"
	"github.com/dcrdata/dcrdata/db/dcrpg"
	"github.com/dcrdata/dcrdata/rpcutils"
	"github.com/dcrdata/dcrdata/txhelpers"
	"github.com/decred/dcrd/wire"
	"github.com/decred/dcrrpcclient"
)

var (
	backendLog      *btclog.Backend
	rpcclientLogger btclog.Logger
	sqliteLogger    btclog.Logger
)

const (
	rescanLogBlockChunk = 250
)

func init() {
	err := InitLogger()
	if err != nil {
		fmt.Printf("Unable to start logger: %v", err)
		os.Exit(1)
	}
	backendLog = btclog.NewBackend(log.Writer())
	rpcclientLogger = backendLog.Logger("RPC")
	dcrrpcclient.UseLogger(rpcclientLogger)
	sqliteLogger = backendLog.Logger("DSQL")
	dcrpg.UseLogger(rpcclientLogger)
}

func mainCore() error {
	// Parse the configuration file, and setup logger.
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Failed to load dcrdata config: %s\n", err.Error())
		return err
	}

	if cfg.CPUProfile != "" {
		var f *os.File
		f, err = os.Create(cfg.CPUProfile)
		if err != nil {
			log.Fatal(err)
			return err
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Connect to node RPC server
	client, _, err := rpcutils.ConnectNodeRPC(cfg.DcrdServ, cfg.DcrdUser,
		cfg.DcrdPass, cfg.DcrdCert, cfg.DisableDaemonTLS)
	if err != nil {
		log.Fatalf("Unable to connect to RPC server: %v", err)
		return err
	}

	infoResult, err := client.GetInfo()
	if err != nil {
		log.Errorf("GetInfo failed: %v", err)
		return err
	}
	log.Info("Node connection count: ", infoResult.Connections)

	host, port, err := net.SplitHostPort(cfg.DBHostPort)
	if err != nil {
		log.Errorf("SplitHostPort failed: %v", err)
		return err
	}

	db, err := dcrpg.Connect(host, port, cfg.DBUser, cfg.DBPass, cfg.DBName)
	if db != nil {
		defer db.Close()
	}
	if err != nil {
		return err
	}

	if cfg.DropDBTables {
		dcrpg.DropTables(db)
		return nil
	}

	if err = dcrpg.CreateTables(db); err != nil {
		return err
	}

	vers := dcrpg.TableVersions(db)
	for tab, ver := range vers {
		fmt.Printf("Table %s: v%d\n", tab, ver)
	}

	// Ctrl-C to shut down.
	// Nothing should be sent the quit channel.  It should only be closed.
	quit := make(chan struct{})
	// Only accept a single CTRL+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Start waiting for the interrupt signal
	go func() {
		<-c
		signal.Stop(c)
		// Close the channel so multiple goroutines can get the message
		log.Infof("CTRL+C hit.  Closing goroutines. Please wait.")
		close(quit)
	}()

	// Get chain servers's best block
	_, height, err := client.GetBestBlock()
	if err != nil {
		return fmt.Errorf("GetBestBlock failed: %v", err)
	}

	// genesisHash, err := client.GetBlockHash(0)
	// if err != nil {
	// 	log.Error("GetBlockHash failed: ", err)
	// 	return err
	// }
	//prev_hash := genesisHash.String()

	var totalTxs, totalRTxs, totalSTxs, totalVouts int64
	var lastTxs, lastVouts int64
	tickTime := 2 * time.Second
	ticker := time.NewTicker(tickTime)
	startTime := time.Now()

	defer func() {
		ticker.Stop()
		totalElapsed := time.Since(startTime).Seconds()
		totalVoutPerSec := totalVouts / int64(totalElapsed)
		totalTxPerSec := totalTxs / int64(totalElapsed)
		log.Infof("Avg. speed: %d tx/s, %d vout/s", totalTxPerSec, totalVoutPerSec)
	}()

	lastBlockDbID := int64(-1)

	startHeight := int64(0)
	for ib := startHeight; ib <= height; ib++ {
		// check for quit signal
		select {
		case <-quit:
			log.Infof("Rescan cancelled at height %d.", ib)
			return nil
		default:
		}

		if (ib-1)%rescanLogBlockChunk == 0 || ib == startHeight {
			if ib == 0 {
				log.Infof("Scanning genesis block.")
			} else {
				endRangeBlock := rescanLogBlockChunk * (1 + (ib-1)/rescanLogBlockChunk)
				if endRangeBlock > height {
					endRangeBlock = height
				}
				log.Infof("Scanning blocks %d to %d...", ib, endRangeBlock)
			}
		}
		select {
		case <-ticker.C:
			txPerSec := float64(totalTxs-lastTxs) / tickTime.Seconds()
			voutPerSec := float64(totalVouts-lastVouts) / tickTime.Seconds()
			log.Infof("(%5d tx/s,%5d vout/s)", int64(txPerSec), int64(voutPerSec))
			lastTxs, lastVouts = totalTxs, totalVouts
		default:
		}

		block, blockHash, err := rpcutils.GetBlock(ib, client)
		if err != nil {
			return fmt.Errorf("GetBlock failed (%s): %v", blockHash, err)
		}
		msgBlock := block.MsgBlock()

		// Create the dbtypes.Block structure
		blockHeader := msgBlock.Header

		// convert each transaction hash to a hex string
		var txHashStrs []string
		txHashes := msgBlock.TxHashes()
		for i := range txHashes {
			txHashStrs = append(txHashStrs, txHashes[i].String())
		}

		var stxHashStrs []string
		stxHashes := msgBlock.STxHashes()
		for i := range stxHashes {
			stxHashStrs = append(stxHashStrs, stxHashes[i].String())
		}

		// Assemble the block
		dbBlock := dbtypes.Block{
			Hash:       blockHash.String(),
			Size:       uint32(msgBlock.SerializeSize()),
			Height:     blockHeader.Height,
			Version:    uint32(blockHeader.Version),
			MerkleRoot: blockHeader.MerkleRoot.String(),
			StakeRoot:  blockHeader.StakeRoot.String(),
			NumTx:      uint32(len(msgBlock.Transactions) + len(msgBlock.STransactions)),
			// nil []int64 for TxDbIDs
			NumRegTx:     uint32(len(msgBlock.Transactions)),
			Tx:           txHashStrs,
			NumStakeTx:   uint32(len(msgBlock.STransactions)),
			STx:          stxHashStrs,
			Time:         uint64(blockHeader.Timestamp.Unix()),
			Nonce:        uint64(blockHeader.Nonce),
			VoteBits:     blockHeader.VoteBits,
			FinalState:   blockHeader.FinalState[:],
			Voters:       blockHeader.Voters,
			FreshStake:   blockHeader.FreshStake,
			Revocations:  blockHeader.Revocations,
			PoolSize:     blockHeader.PoolSize,
			Bits:         blockHeader.Bits,
			SBits:        uint64(blockHeader.SBits),
			Difficulty:   txhelpers.GetDifficultyRatio(blockHeader.Bits, activeChain),
			ExtraData:    blockHeader.ExtraData[:],
			StakeVersion: blockHeader.StakeVersion,
			PreviousHash: blockHeader.PrevBlock.String(),
		}

		// Extract transactions and their vouts. Insert vouts into their pg table,
		// returning their PK IDs, which are stored in the corresponding transaction
		// data struct. Insert each transaction once they are updated with their
		// vouts' IDs, returning the transaction PK ID, which are stored in the
		// containing block data struct.

		// regular transactions
		dbTransactions, dbTxVouts := dbtypes.ExtractBlockTransactions(msgBlock,
			wire.TxTreeRegular, activeChain)

		dbBlock.TxDbIDs = make([]uint64, len(dbTransactions))
		for it, dbtx := range dbTransactions {
			// vouts := dbTxVouts[it]
			// dbtx.VoutDbIds = make([]uint64, len(vouts))
			// for iv, dbvout := range vouts {
			// 	// Store the vout PK ID in the transaction
			// 	dbtx.VoutDbIds[iv], err = dcrpg.InsertVout(db, dbvout)
			// 	if err != nil {
			// 		fmt.Println(err)
			// 	}
			// }

			dbtx.VoutDbIds, err = dcrpg.InsertVouts(db, dbTxVouts[it])
			if err != nil {
				log.Error(err)
			}
			totalVouts += int64(len(dbtx.VoutDbIds))

			// Store the tx PK ID in the block
			dbBlock.TxDbIDs[it], err = dcrpg.InsertTx(db, dbtx)
			if err != nil {
				log.Error(err)
			}
		}
		totalTxs += int64(len(dbTransactions))
		totalRTxs += int64(len(dbTransactions))

		// stake transactions
		dbSTransactions, dbSTxVouts := dbtypes.ExtractBlockTransactions(msgBlock,
			wire.TxTreeStake, activeChain)

		dbBlock.STxDbIDs = make([]uint64, len(dbSTransactions))
		for it, dbtx := range dbSTransactions {
			// vouts := dbSTxVouts[it]
			// dbtx.VoutDbIds = make([]uint64, len(vouts))
			// for iv, dbvout := range vouts {
			// 	// Store the vout PK ID in the transaction
			// 	dbtx.VoutDbIds[iv], err = dcrpg.InsertVout(db, dbvout)
			// 	if err != nil {
			// 		fmt.Println(err)
			// 	}
			// }

			dbtx.VoutDbIds, err = dcrpg.InsertVouts(db, dbSTxVouts[it])
			if err != nil {
				log.Error(err)
			}
			totalVouts += int64(len(dbtx.VoutDbIds))

			// Store the tx PK ID in the block
			dbBlock.STxDbIDs[it], err = dcrpg.InsertTx(db, dbtx)
			if err != nil {
				log.Error(err)
			}
		}
		totalTxs += int64(len(dbSTransactions))
		totalSTxs += int64(len(dbSTransactions))

		// Store the block now that it has all it's transaction PK IDs
		blockDbID, err := dcrpg.InsertBlock(db, &dbBlock)
		if err != nil {
			log.Error(err)
			return err
		}

		err = dcrpg.InsertBlockPrevNext(db, blockDbID, dbBlock.Hash,
			dbBlock.PreviousHash, "")
		if err != nil {
			log.Error(err)
			return err
		}

		// Update last block in db with this block's hash as it's next
		if lastBlockDbID > 0 {
			err = dcrpg.UpdateBlockNext(db, uint64(lastBlockDbID), dbBlock.Hash)
			if err != nil {
				fmt.Println(err)
				return err
			}
		}
		lastBlockDbID = int64(blockDbID)

		// update height, the end condition for the loop
		if _, height, err = client.GetBestBlock(); err != nil {
			return fmt.Errorf("GetBestBlock failed: %v", err)
		}
	}

	return nil
}

func main() {
	if err := mainCore(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
	os.Exit(0)
}
