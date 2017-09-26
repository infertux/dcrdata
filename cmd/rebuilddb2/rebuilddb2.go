package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime/pprof"
	"strconv"

	"github.com/btcsuite/btclog"
	"github.com/dcrdata/dcrdata/db/dbtypes"
	"github.com/dcrdata/dcrdata/db/dcrpg"
	"github.com/dcrdata/dcrdata/rpcutils"
	"github.com/dcrdata/dcrdata/txhelpers"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/txscript"
	"github.com/decred/dcrd/wire"
	"github.com/decred/dcrrpcclient"
)

var (
	backendLog      *btclog.Backend
	rpcclientLogger btclog.Logger
	sqliteLogger    btclog.Logger
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

	blockHash, _, err := client.GetBestBlock()
	if err != nil {
		log.Error("GetBestBlock failed: ", err)
		return err
	}
	msgBlock, err := client.GetBlock(blockHash)
	if err != nil {
		log.Error("GetBlock failed: ", err)
		return err
	}
	//block := dcrutil.NewBlock(msgBlock)
	//gbvr, _ := client.GetBlockVerbose(blockHash)

	blockHeader := msgBlock.Header

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

	// regular transactions
	dbTransactions, dbTxVouts := processTransactions(msgBlock.Transactions, blockHash)

	dbBlock.TxDbIDs = make([]uint64, len(dbTransactions))
	for it, dbtx := range dbTransactions {
		vouts := dbTxVouts[it]
		dbtx.VoutDbIds = make([]uint64, len(vouts))
		for iv, dbvout := range vouts {
			dbtx.VoutDbIds[iv], err = dcrpg.InsertVout(db, dbvout)
			if err != nil {
				fmt.Println(err)
			}
		}

		dbBlock.TxDbIDs[it], err = dcrpg.InsertTx(db, dbtx)
		if err != nil {
			fmt.Println(err)
		}
	}

	// stake transactions, txTree := wire.TxTreeStake
	dbSTransactions, dbSTxVouts := processTransactions(msgBlock.STransactions, blockHash)

	dbBlock.STxDbIDs = make([]uint64, len(dbSTransactions))
	for it, dbtx := range dbSTransactions {
		vouts := dbSTxVouts[it]
		dbtx.VoutDbIds = make([]uint64, len(vouts))
		for iv, dbvout := range vouts {
			dbtx.VoutDbIds[iv], err = dcrpg.InsertVout(db, dbvout)
			if err != nil {
				fmt.Println(err)
			}
		}

		dbBlock.STxDbIDs[it], err = dcrpg.InsertTx(db, dbtx)
		if err != nil {
			fmt.Println(err)
		}
	}

	id, err := dcrpg.InsertBlock(db, &dbBlock)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println("New record ID is:", id)

	return nil
}

func main() {
	if err := mainCore(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func processTransactions(txs []*wire.MsgTx, blockHash *chainhash.Hash) ([]*dbtypes.Tx, [][]*dbtypes.Vout) {
	dbTransactions := make([]*dbtypes.Tx, 0, len(txs))
	dbTxVouts := make([][]*dbtypes.Vout, len(txs))

	for txIndex, tx := range txs {
		dbTx := &dbtypes.Tx{
			BlockHash:  blockHash.String(),
			BlockIndex: uint32(txIndex),
			TxID:       tx.TxHash().String(),
			Version:    tx.Version,
			Locktime:   tx.LockTime,
			Expiry:     tx.Expiry,
			NumVin:     uint32(len(tx.TxIn)),
			NumVout:    uint32(len(tx.TxOut)),
		}

		dbTx.Vin = make([]dbtypes.VinTxProperty, 0, dbTx.NumVin)
		for _, txin := range tx.TxIn {
			dbTx.Vin = append(dbTx.Vin, dbtypes.VinTxProperty{
				PrevTxHash:  txin.PreviousOutPoint.Hash.String(),
				PrevTxIndex: txin.PreviousOutPoint.Index,
				PrevTxTree:  uint16(txin.PreviousOutPoint.Tree),
				Sequence:    txin.Sequence,
				ValueIn:     uint64(txin.ValueIn),
				BlockHeight: txin.BlockHeight,
				BlockIndex:  txin.BlockIndex,
				ScriptHex:   txin.SignatureScript,
			})
		}

		// Vouts and their db IDs
		dbTxVouts[txIndex] = make([]*dbtypes.Vout, 0, len(tx.TxOut))
		for io, txout := range tx.TxOut {
			outpoint := dbTx.TxID + ":" + strconv.Itoa(io)
			vout := dbtypes.Vout{
				Outpoint:     outpoint,
				Value:        uint64(txout.Value),
				Ind:          uint32(io),
				Version:      txout.Version,
				ScriptPubKey: txout.PkScript,
			}
			scriptClass, scriptAddrs, reqSigs, err := txscript.ExtractPkScriptAddrs(
				vout.Version, vout.ScriptPubKey, activeChain)
			if err != nil {
				fmt.Println(err)
			}
			addys := make([]string, 0, len(scriptAddrs))
			for ia := range scriptAddrs {
				addys = append(addys, scriptAddrs[ia].String())
			}
			vout.ScriptPubKeyData.ReqSigs = uint32(reqSigs)
			vout.ScriptPubKeyData.Type = scriptClass.String()
			vout.ScriptPubKeyData.Addresses = addys
			dbTxVouts[txIndex] = append(dbTxVouts[txIndex], &vout)
		}

		dbTx.VoutDbIds = make([]uint64, len(dbTxVouts[txIndex]))

		dbTransactions = append(dbTransactions, dbTx)
	}

	return dbTransactions, dbTxVouts
}
