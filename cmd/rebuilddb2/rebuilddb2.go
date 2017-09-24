package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime/pprof"

	"github.com/btcsuite/btclog"
	"github.com/dcrdata/dcrdata/db/dcrpg"
	"github.com/dcrdata/dcrdata/rpcutils"
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

func mainCore() int {
	// Parse the configuration file, and setup logger.
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Failed to load dcrdata config: %s\n", err.Error())
		return 1
	}

	if cfg.CPUProfile != "" {
		f, err := os.Create(cfg.CPUProfile)
		if err != nil {
			log.Fatal(err)
			return -1
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Connect to node RPC server
	client, _, err := rpcutils.ConnectNodeRPC(cfg.DcrdServ, cfg.DcrdUser,
		cfg.DcrdPass, cfg.DcrdCert, cfg.DisableDaemonTLS)
	if err != nil {
		log.Fatalf("Unable to connect to RPC server: %v", err)
		return 1
	}

	infoResult, err := client.GetInfo()
	if err != nil {
		log.Errorf("GetInfo failed: %v", err)
		return 1
	}
	log.Info("Node connection count: ", infoResult.Connections)

	_, _, err = client.GetBestBlock()
	if err != nil {
		log.Error("GetBestBlock failed: ", err)
		return 2
	}

	host, port, err := net.SplitHostPort(cfg.DBHostPort)
	if err != nil {
		log.Errorf("SplitHostPort failed: %v", err)
		return 3
	}

	db, err := dcrpg.Connect(host, port, cfg.DBUser, cfg.DBPass, cfg.DBName)
	if db != nil {
		defer db.Close()
	}
	if err != nil {
		fmt.Println(err)
		return 4
	}

	if cfg.DropDBTables {
		dcrpg.DropTables(db)
		return 0
	}

	if err = dcrpg.CreateTables(db); err != nil {
		fmt.Println(err)
		return 5
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

	// Resync db
	// var waitSync sync.WaitGroup
	// waitSync.Add(1)
	// //go sqliteDB.SyncDB(&waitSync, quit)
	// err = db.SyncDBWithPoolValue(&waitSync, quit)
	// if err != nil {
	// 	log.Error(err)
	// }

	// waitSync.Wait()

	log.Print("Done!")

	return 0
}

func main() {
	os.Exit(mainCore())
}
