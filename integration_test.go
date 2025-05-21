//go:build integration

// todo: set up a test for dane, mta-sts, etc.

package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/slog"

	"github.com/qompassai/beacon/dns"
	"github.com/qompassai/beacon/imapclient"
	"github.com/qompassai/beacon/mlog"
	"github.com/qompassai/beacon/beacon-"
	"github.com/qompassai/beacon/sasl"
	"github.com/qompassai/beacon/smtpclient"
)

func tcheck(t *testing.T, err error, errmsg string) {
	if err != nil {
		t.Helper()
		t.Fatalf("%s: %s", errmsg, err)
	}
}

func TestDeliver(t *testing.T) {
	log := mlog.New("integration", nil)
	mlog.Logfmt = true

	hostname, err := os.Hostname()
	tcheck(t, err, "hostname")
	ourHostname, err := dns.ParseDomain(hostname)
	tcheck(t, err, "parse hostname")

	// Single update from IMAP IDLE.
	type idleResponse struct {
		untagged imapclient.Untagged
		err      error
	}

	// Deliver submits a message over submissions, and checks with imap idle if the
	// message is received by the destination mail server.
	deliver := func(checkTime bool, dialtls bool, imaphost, imapuser, imappassword string, send func()) {
		t.Helper()

		// Connect to IMAP, execute IDLE command, which will return on deliver message.
		// TLS certificates work because the container has the CA certificates configured.
		var imapconn net.Conn
		var err error
		if dialtls {
			imapconn, err = tls.Dial("tcp", imaphost, nil)
		} else {
			imapconn, err = net.Dial("tcp", imaphost)
		}
		tcheck(t, err, "dial imap")
		defer imapconn.Close()

		imapc, err := imapclient.New(imapconn, false)
		tcheck(t, err, "new imapclient")

		_, _, err = imapc.Login(imapuser, imappassword)
		tcheck(t, err, "imap login")

		_, _, err = imapc.Select("Inbox")
		tcheck(t, err, "imap select inbox")

		err = imapc.Commandf("", "idle")
		tcheck(t, err, "write imap idle command")

		_, _, _, err = imapc.ReadContinuation()
		tcheck(t, err, "read imap continuation")

		idle := make(chan idleResponse)
		go func() {
			for {
				untagged, err := imapc.ReadUntagged()
				idle <- idleResponse{untagged, err}
				if err != nil {
					return
				}
			}
		}()
		defer func() {
			err := imapc.Writelinef("done")
			tcheck(t, err, "aborting idle")
		}()

		t0 := time.Now()
		send()

		// Wait for notification of delivery.
		select {
		case resp := <-idle:
			tcheck(t, resp.err, "idle notification")
			_, ok := resp.untagged.(imapclient.UntaggedExists)
			if !ok {
				t.Fatalf("got idle %#v, expected untagged exists", resp.untagged)
			}
			if d := time.Since(t0); checkTime && d < 1*time.Second {
				t.Fatalf("delivery took %v, but should have taken at least 1 second, the first-time sender delay", d)
			}
		case <-time.After(30 * time.Second):
			t.Fatalf("timeout after 5s waiting for IMAP IDLE notification of new message, should take about 1 second")
		}
	}

	submit := func(dialtls bool, mailfrom, password, desthost, rcptto string) {
		var conn net.Conn
		var err error
		if dialtls {
			conn, err = tls.Dial("tcp", desthost, nil)
		} else {
			conn, err = net.Dial("tcp", desthost)
		}
		tcheck(t, err, "dial submission")
		defer conn.Close()

		msg := fmt.Sprintf(`From: <%s>
To: <%s>
Subject: test message

This is the message.
`, mailfrom, rcptto)
		msg = strings.ReplaceAll(msg, "\n", "\r\n")
		auth := func(mechanisms []string, cs *tls.ConnectionState) (sasl.Client, error) {
			return sasl.NewClientPlain(mailfrom, password), nil
		}
		c, err := smtpclient.New(beacon.Context, log.Logger, conn, smtpclient.TLSSkip, false, ourHostname, dns.Domain{ASCII: desthost}, smtpclient.Opts{Auth: auth})
		tcheck(t, err, "smtp hello")
		err = c.Deliver(beacon.Context, mailfrom, rcptto, int64(len(msg)), strings.NewReader(msg), false, false, false)
		tcheck(t, err, "deliver with smtp")
		err = c.Close()
		tcheck(t, err, "close smtpclient")
	}

	// Make sure beaconacmepebble has a TLS certificate.
	conn, err := tls.Dial("tcp", "beaconacmepebble.beacon1.example:465", nil)
	tcheck(t, err, "dial submission")
	defer conn.Close()

	log.Print("submitting email to beaconacmepebble, waiting for imap notification at beaconmail2")
	t0 := time.Now()
	deliver(true, true, "beaconmail2.beacon2.example:993", "beacontest2@beacon2.example", "accountpass4321", func() {
		submit(true, "beacontest1@beacon1.example", "accountpass1234", "beaconacmepebble.beacon1.example:465", "beacontest2@beacon2.example")
	})
	log.Print("success", slog.Duration("duration", time.Since(t0)))

	log.Print("submitting email to beaconmail2, waiting for imap notification at beaconacmepebble")
	t0 = time.Now()
	deliver(true, true, "beaconacmepebble.beacon1.example:993", "beacontest1@beacon1.example", "accountpass1234", func() {
		submit(true, "beacontest2@beacon2.example", "accountpass4321", "beaconmail2.beacon2.example:465", "beacontest1@beacon1.example")
	})
	log.Print("success", slog.Duration("duration", time.Since(t0)))

	log.Print("submitting email to postfix, waiting for imap notification at beaconacmepebble")
	t0 = time.Now()
	deliver(false, true, "beaconacmepebble.beacon1.example:993", "beacontest1@beacon1.example", "accountpass1234", func() {
		submit(true, "beacontest1@beacon1.example", "accountpass1234", "beaconacmepebble.beacon1.example:465", "root@postfix.example")
	})
	log.Print("success", slog.Duration("duration", time.Since(t0)))

	log.Print("submitting email to localserve")
	t0 = time.Now()
	deliver(false, false, "localserve.beacon1.example:1143", "beacon@localhost", "beaconbeaconbeacon", func() {
		submit(false, "beacon@localhost", "beaconbeaconbeacon", "localserve.beacon1.example:1587", "beacontest1@beacon1.example")
	})
	log.Print("success", slog.Duration("duration", time.Since(t0)))

	log.Print("submitting email to localserve")
	t0 = time.Now()
	deliver(false, false, "localserve.beacon1.example:1143", "beacon@localhost", "beaconbeaconbeacon", func() {
		cmd := exec.Command("go", "run", ".", "sendmail", "beacon@localhost")
		const msg = `Subject: test

a message.
`
		cmd.Stdin = strings.NewReader(msg)
		var out strings.Builder
		cmd.Stdout = &out
		err := cmd.Run()
		log.Print("sendmail", slog.String("output", out.String()))
		tcheck(t, err, "sendmail")
	})
	log.Print("success", slog.Any("duration", time.Since(t0)))
}
