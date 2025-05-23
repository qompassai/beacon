package smtpserver

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qompassai/beacon/dns"
	"github.com/qompassai/beacon/mlog"
	"github.com/qompassai/beacon/beacon-"
	"github.com/qompassai/beacon/queue"
	"github.com/qompassai/beacon/store"
)

// Fuzz the server. For each fuzz string, we set up servers in various connection states, and write the string as command.
func FuzzServer(f *testing.F) {
	f.Add("HELO remote")
	f.Add("EHLO remote")
	f.Add("AUTH PLAIN")
	f.Add("MAIL FROM:<remote@remote>")
	f.Add("RCPT TO:<local@beacon.example>")
	f.Add("DATA")
	f.Add(".")
	f.Add("RSET")
	f.Add("VRFY x")
	f.Add("EXPN x")
	f.Add("HELP")
	f.Add("NOOP")
	f.Add("QUIT")

	log := mlog.New("smtpserver", nil)
	beacon.Context = ctxbg
	beacon.ConfigStaticPath = filepath.FromSlash("../testdata/smtpserverfuzz/beacon.conf")
	beacon.MustLoadConfig(true, false)
	dataDir := beacon.ConfigDirPath(beacon.Conf.Static.DataDir)
	os.RemoveAll(dataDir)
	acc, err := store.OpenAccount(log, "mjl")
	if err != nil {
		f.Fatalf("open account: %v", err)
	}
	defer acc.Close()
	err = acc.SetPassword(log, "testtest")
	if err != nil {
		f.Fatalf("set password: %v", err)
	}
	defer store.Switchboard()()
	err = queue.Init()
	if err != nil {
		f.Fatalf("queue init: %v", err)
	}
	defer queue.Shutdown()

	comm := store.RegisterComm(acc)
	defer comm.Unregister()

	var cid int64 = 1

	var fl *os.File
	if false {
		fl, err = os.Create("fuzz.log")
		if err != nil {
			f.Fatalf("fuzz log")
		}
		defer fl.Close()
	}
	flog := func(err error, msg string) {
		if fl != nil && err != nil {
			fmt.Fprintf(fl, "%s: %v\n", msg, err)
		}
	}

	f.Fuzz(func(t *testing.T, s string) {
		run := func(cmds []string) {
			limitersInit() // Reset rate limiters.
			serverConn, clientConn := net.Pipe()
			defer serverConn.Close()
			defer clientConn.Close()

			go func() {
				err := clientConn.SetDeadline(time.Now().Add(time.Second))
				flog(err, "set client deadline")
				_, err = clientConn.Read(make([]byte, 1024))
				flog(err, "read ehlo")
				for _, cmd := range cmds {
					_, err = clientConn.Write([]byte(cmd + "\r\n"))
					flog(err, "write command")
					_, err = clientConn.Read(make([]byte, 1024))
					flog(err, "read response")
				}
				_, err = clientConn.Write([]byte(s + "\r\n"))
				flog(err, "write test command")
				_, err = clientConn.Read(make([]byte, 1024))
				flog(err, "read test response")
				clientConn.Close()
				serverConn.Close()
			}()

			resolver := dns.MockResolver{}
			const submission = false
			err := serverConn.SetDeadline(time.Now().Add(time.Second))
			flog(err, "set server deadline")
			serve("test", cid, dns.Domain{ASCII: "beacon.example"}, nil, serverConn, resolver, submission, false, 100<<10, false, false, false, nil, 0)
			cid++
		}

		run([]string{})
		run([]string{"EHLO remote"})
		run([]string{"EHLO remote", "MAIL FROM:<remote@example.org>"})
		run([]string{"EHLO remote", "MAIL FROM:<remote@example.org>", "RCPT TO:<mjl@beacon.example>"})
		// todo: submission with login
	})
}
