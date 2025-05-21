package main

import (
	"fmt"
	"os"
	"time"

	"github.com/qompassai/beacon/dmarcdb"
	"github.com/qompassai/beacon/dns"
	"github.com/qompassai/beacon/http"
	"github.com/qompassai/beacon/imapserver"
	"github.com/qompassai/beacon/mlog"
	"github.com/qompassai/beacon/beacon-"
	"github.com/qompassai/beacon/mtastsdb"
	"github.com/qompassai/beacon/queue"
	"github.com/qompassai/beacon/smtpserver"
	"github.com/qompassai/beacon/store"
	"github.com/qompassai/beacon/tlsrptdb"
	"github.com/qompassai/beacon/tlsrptsend"
)

func shutdown(log mlog.Log) {
	// We indicate we are shutting down. Causes new connections and new SMTP commands
	// to be rejected. Should stop active connections pretty quickly.
	beacon.ShutdownCancel()

	// Now we are going to wait for all connections to be gone, up to a timeout.
	done := beacon.Connections.Done()
	second := time.Tick(time.Second)
	select {
	case <-done:
		log.Print("connections shutdown, waiting until 1 second passed")
		<-second

	case <-time.Tick(3 * time.Second):
		// We now cancel all pending operations, and set an immediate deadline on sockets.
		// Should get us a clean shutdown relatively quickly.
		beacon.ContextCancel()
		beacon.Connections.Shutdown()

		second := time.Tick(time.Second)
		select {
		case <-done:
			log.Print("no more connections, shutdown is clean, waiting until 1 second passed")
			<-second // Still wait for second, giving processes like imports a chance to clean up.
		case <-second:
			log.Print("shutting down with pending sockets")
		}
	}
	err := os.Remove(beacon.DataDirPath("ctl"))
	log.Check(err, "removing ctl unix domain socket during shutdown")
}

// start initializes all packages, starts all listeners and the switchboard
// goroutine, then returns.
func start(mtastsdbRefresher, sendDMARCReports, sendTLSReports, skipForkExec bool) error {
	smtpserver.Listen()
	imapserver.Listen()
	http.Listen()

	if !skipForkExec {
		// If we were just launched as root, fork and exec as unprivileged user, handing
		// over the bound sockets to the new process. We'll get to this same code path
		// again, skipping this if block, continuing below with the actual serving.
		if os.Getuid() == 0 {
			beacon.ForkExecUnprivileged()
			panic("cannot happen")
		} else {
			beacon.CleanupPassedFiles()
		}
	}

	if err := mtastsdb.Init(mtastsdbRefresher); err != nil {
		return fmt.Errorf("mtasts init: %s", err)
	}

	if err := tlsrptdb.Init(); err != nil {
		return fmt.Errorf("tlsrpt init: %s", err)
	}

	done := make(chan struct{}, 1)
	if err := queue.Start(dns.StrictResolver{Pkg: "queue"}, done); err != nil {
		return fmt.Errorf("queue start: %s", err)
	}

	// dmarcdb starts after queue because it may start sending reports through the queue.
	if err := dmarcdb.Init(); err != nil {
		return fmt.Errorf("dmarc init: %s", err)
	}
	if sendDMARCReports {
		dmarcdb.Start(dns.StrictResolver{Pkg: "dmarcdb"})
	}

	if sendTLSReports {
		tlsrptsend.Start(dns.StrictResolver{Pkg: "tlsrptsend"})
	}

	store.StartAuthCache()
	smtpserver.Serve()
	imapserver.Serve()
	http.Serve()

	go func() {
		store.Switchboard()
		<-make(chan struct{})
	}()
	return nil
}
