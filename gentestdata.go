package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mjl-/bstore"
	"github.com/mjl-/sconf"

	"github.com/qompassai/beacon/config"
	"github.com/qompassai/beacon/dmarcdb"
	"github.com/qompassai/beacon/dmarcrpt"
	"github.com/qompassai/beacon/dns"
	"github.com/qompassai/beacon/mlog"
	"github.com/qompassai/beacon/beacon-"
	"github.com/qompassai/beacon/beaconvar"
	"github.com/qompassai/beacon/mtasts"
	"github.com/qompassai/beacon/mtastsdb"
	"github.com/qompassai/beacon/queue"
	"github.com/qompassai/beacon/smtp"
	"github.com/qompassai/beacon/store"
	"github.com/qompassai/beacon/tlsrpt"
	"github.com/qompassai/beacon/tlsrptdb"
)

func cmdGentestdata(c *cmd) {
	c.unlisted = true
	c.params = "dest-dir"
	c.help = `Generate a data directory populated, for testing upgrades.`
	args := c.Parse()
	if len(args) != 1 {
		c.Usage()
	}

	destDataDir, err := filepath.Abs(args[0])
	xcheckf(err, "making destination directory an absolute path")

	if _, err := os.Stat(destDataDir); err == nil {
		log.Fatalf("destination directory already exists, refusing to generate test data")
	}
	err = os.MkdirAll(destDataDir, 0770)
	xcheckf(err, "creating destination data directory")
	err = os.MkdirAll(filepath.Join(destDataDir, "tmp"), 0770)
	xcheckf(err, "creating tmp directory")

	tempfile := func() *os.File {
		f, err := os.CreateTemp(filepath.Join(destDataDir, "tmp"), "temp")
		xcheckf(err, "creating temp file")
		return f
	}

	ctxbg := context.Background()
	beacon.Conf.Log[""] = mlog.LevelInfo
	mlog.SetConfig(beacon.Conf.Log)

	const domainsConf = `
Domains:
	beacon.example: nil
	☺.example: nil
Accounts:
	test0:
		Domain: beacon.example
		Destinations:
			test0@beacon.example: nil
	test1:
		Domain: beacon.example
		Destinations:
			test1@beacon.example: nil
	test2:
		Domain: ☺.example
		Destinations:
			☹@☺.example: nil
		JunkFilter:
			Threshold: 0.95
			Params:
				Twograms: true
				MaxPower: 0.1
				TopWords: 10
				IgnoreWords: 0.1
`

	beacon.ConfigStaticPath = filepath.FromSlash("/tmp/beacon-bogus/beacon.conf")
	beacon.ConfigDynamicPath = filepath.FromSlash("/tmp/beacon-bogus/domains.conf")
	beacon.Conf.DynamicLastCheck = time.Now() // Should prevent warning.
	beacon.Conf.Static = config.Static{
		DataDir: destDataDir,
	}
	err = sconf.Parse(strings.NewReader(domainsConf), &beacon.Conf.Dynamic)
	xcheckf(err, "parsing domains config")

	const dmarcReport = `<?xml version="1.0" encoding="UTF-8" ?>
<feedback>
  <report_metadata>
    <org_name>google.com</org_name>
    <email>noreply-dmarc-support@google.com</email>
    <extra_contact_info>https://support.google.com/a/answer/2466580</extra_contact_info>
    <report_id>10051505501689795560</report_id>
    <date_range>
      <begin>1596412800</begin>
      <end>1596499199</end>
    </date_range>
  </report_metadata>
  <policy_published>
    <domain>beacon.example</domain>
    <adkim>r</adkim>
    <aspf>r</aspf>
    <p>reject</p>
    <sp>reject</sp>
    <pct>100</pct>
  </policy_published>
  <record>
    <row>
      <source_ip>127.0.0.1</source_ip>
      <count>1</count>
      <policy_evaluated>
        <disposition>none</disposition>
        <dkim>pass</dkim>
        <spf>pass</spf>
      </policy_evaluated>
    </row>
    <identifiers>
      <header_from>example.org</header_from>
    </identifiers>
    <auth_results>
      <dkim>
        <domain>example.org</domain>
        <result>pass</result>
        <selector>example</selector>
      </dkim>
      <spf>
        <domain>example.org</domain>
        <result>pass</result>
      </spf>
    </auth_results>
  </record>
</feedback>
`

	const tlsReport = `{
     "organization-name": "Company-X",
     "date-range": {
       "start-datetime": "2016-04-01T00:00:00Z",
       "end-datetime": "2016-04-01T23:59:59Z"
     },
     "contact-info": "sts-reporting@company-x.example",
     "report-id": "5065427c-23d3-47ca-b6e0-946ea0e8c4be",
     "policies": [{
       "policy": {
         "policy-type": "sts",
         "policy-string": ["version: STSv1","mode: testing",
               "mx: *.mail.company-y.example","max_age: 86400"],
         "policy-domain": "beacon.example",
         "mx-host": ["*.mail.company-y.example"]
       },
       "summary": {
         "total-successful-session-count": 5326,
         "total-failure-session-count": 303
       },
       "failure-details": [{
         "result-type": "certificate-expired",
         "sending-mta-ip": "2001:db8:abcd:0012::1",
         "receiving-mx-hostname": "mx1.mail.company-y.example",
         "failed-session-count": 100
       }, {
         "result-type": "starttls-not-supported",
         "sending-mta-ip": "2001:db8:abcd:0013::1",
         "receiving-mx-hostname": "mx2.mail.company-y.example",
         "receiving-ip": "203.0.113.56",
         "failed-session-count": 200,
         "additional-information": "https://reports.company-x.example/report_info ? id = 5065427 c - 23 d3# StarttlsNotSupported "
       }, {
         "result-type": "validation-failure",
         "sending-mta-ip": "198.51.100.62",
         "receiving-ip": "203.0.113.58",
         "receiving-mx-hostname": "mx-backup.mail.company-y.example",
         "failed-session-count": 3,
         "failure-reason-code": "X509_V_ERR_PROXY_PATH_LENGTH_EXCEEDED"
       }]
     }]
   }`

	err = os.WriteFile(filepath.Join(destDataDir, "beaconversion"), []byte(beaconvar.Version), 0660)
	xcheckf(err, "writing beaconversion")

	// Populate dmarc.db.
	err = dmarcdb.Init()
	xcheckf(err, "dmarcdb init")
	report, err := dmarcrpt.ParseReport(strings.NewReader(dmarcReport))
	xcheckf(err, "parsing dmarc aggregate report")
	err = dmarcdb.AddReport(ctxbg, report, dns.Domain{ASCII: "beacon.example"})
	xcheckf(err, "adding dmarc aggregate report")

	// Populate mtasts.db.
	err = mtastsdb.Init(false)
	xcheckf(err, "mtastsdb init")
	mtastsPolicy := mtasts.Policy{
		Version: "STSv1",
		Mode:    mtasts.ModeTesting,
		MX: []mtasts.STSMX{
			{Domain: dns.Domain{ASCII: "mx1.example.com"}},
			{Domain: dns.Domain{ASCII: "mx2.example.com"}},
			{Domain: dns.Domain{ASCII: "backup-example.com"}, Wildcard: true},
		},
		MaxAgeSeconds: 1296000,
	}
	err = mtastsdb.Upsert(ctxbg, dns.Domain{ASCII: "beacon.example"}, "123", &mtastsPolicy, mtastsPolicy.String())
	xcheckf(err, "adding mtastsdb report")

	// Populate tlsrpt.db.
	err = tlsrptdb.Init()
	xcheckf(err, "tlsrptdb init")
	tlsreportJSON, err := tlsrpt.Parse(strings.NewReader(tlsReport))
	xcheckf(err, "parsing tls report")
	tlsr := tlsreportJSON.Convert()
	err = tlsrptdb.AddReport(ctxbg, c.log, dns.Domain{ASCII: "beacon.example"}, "tlsrpt@beacon.example", false, &tlsr)
	xcheckf(err, "adding tls report")

	// Populate queue, with a message.
	err = queue.Init()
	xcheckf(err, "queue init")
	mailfrom := smtp.Path{Localpart: "other", IPDomain: dns.IPDomain{Domain: dns.Domain{ASCII: "other.example"}}}
	rcptto := smtp.Path{Localpart: "test0", IPDomain: dns.IPDomain{Domain: dns.Domain{ASCII: "beacon.example"}}}
	prefix := []byte{}
	mf := tempfile()
	xcheckf(err, "temp file for queue message")
	defer os.Remove(mf.Name())
	defer mf.Close()
	const qmsg = "From: <test0@beacon.example>\r\nTo: <other@remote.example>\r\nSubject: test\r\n\r\nthe message...\r\n"
	_, err = fmt.Fprint(mf, qmsg)
	xcheckf(err, "writing message")
	qm := queue.MakeMsg("test0", mailfrom, rcptto, false, false, int64(len(qmsg)), "<test@localhost>", prefix, nil)
	err = queue.Add(ctxbg, c.log, &qm, mf)
	xcheckf(err, "enqueue message")

	// Create three accounts.
	// First account without messages.
	accTest0, err := store.OpenAccount(c.log, "test0")
	xcheckf(err, "open account test0")
	err = accTest0.ThreadingWait(c.log)
	xcheckf(err, "wait for threading to finish")
	err = accTest0.Close()
	xcheckf(err, "close account")

	// Second account with one message.
	accTest1, err := store.OpenAccount(c.log, "test1")
	xcheckf(err, "open account test1")
	err = accTest1.ThreadingWait(c.log)
	xcheckf(err, "wait for threading to finish")
	err = accTest1.DB.Write(ctxbg, func(tx *bstore.Tx) error {
		inbox, err := bstore.QueryTx[store.Mailbox](tx).FilterNonzero(store.Mailbox{Name: "Inbox"}).Get()
		xcheckf(err, "looking up inbox")
		const msg = "From: <other@remote.example>\r\nTo: <test1@beacon.example>\r\nSubject: test\r\n\r\nthe message...\r\n"
		m := store.Message{
			MailboxID:          inbox.ID,
			MailboxOrigID:      inbox.ID,
			MailboxDestinedID:  inbox.ID,
			RemoteIP:           "1.2.3.4",
			RemoteIPMasked1:    "1.2.3.4",
			RemoteIPMasked2:    "1.2.3.0",
			RemoteIPMasked3:    "1.2.0.0",
			EHLODomain:         "other.example",
			MailFrom:           "other@remote.example",
			MailFromLocalpart:  smtp.Localpart("other"),
			MailFromDomain:     "remote.example",
			RcptToLocalpart:    "test1",
			RcptToDomain:       "beacon.example",
			MsgFromLocalpart:   "other",
			MsgFromDomain:      "remote.example",
			MsgFromOrgDomain:   "remote.example",
			EHLOValidated:      true,
			MailFromValidated:  true,
			MsgFromValidated:   true,
			EHLOValidation:     store.ValidationStrict,
			MailFromValidation: store.ValidationPass,
			MsgFromValidation:  store.ValidationStrict,
			DKIMDomains:        []string{"other.example"},
			Size:               int64(len(msg)),
		}
		mf := tempfile()
		xcheckf(err, "creating temp file for delivery")
		_, err = fmt.Fprint(mf, msg)
		xcheckf(err, "writing deliver message to file")
		err = accTest1.DeliverMessage(c.log, tx, &m, mf, false, true, false, true)

		mfname := mf.Name()
		xcheckf(err, "add message to account test1")
		err = mf.Close()
		xcheckf(err, "closing file")
		err = os.Remove(mfname)
		xcheckf(err, "removing temp message file")

		err = tx.Get(&inbox)
		xcheckf(err, "get inbox")
		inbox.Add(m.MailboxCounts())
		err = tx.Update(&inbox)
		xcheckf(err, "update inbox")

		return nil
	})
	xcheckf(err, "write transaction with new message")
	err = accTest1.Close()
	xcheckf(err, "close account")

	// Third account with two messages and junkfilter.
	accTest2, err := store.OpenAccount(c.log, "test2")
	xcheckf(err, "open account test2")
	err = accTest2.ThreadingWait(c.log)
	xcheckf(err, "wait for threading to finish")
	err = accTest2.DB.Write(ctxbg, func(tx *bstore.Tx) error {
		inbox, err := bstore.QueryTx[store.Mailbox](tx).FilterNonzero(store.Mailbox{Name: "Inbox"}).Get()
		xcheckf(err, "looking up inbox")
		const msg0 = "From: <other@remote.example>\r\nTo: <☹@xn--74h.example>\r\nSubject: test\r\n\r\nthe message...\r\n"
		m0 := store.Message{
			MailboxID:          inbox.ID,
			MailboxOrigID:      inbox.ID,
			MailboxDestinedID:  inbox.ID,
			RemoteIP:           "::1",
			RemoteIPMasked1:    "::",
			RemoteIPMasked2:    "::",
			RemoteIPMasked3:    "::",
			EHLODomain:         "other.example",
			MailFrom:           "other@remote.example",
			MailFromLocalpart:  smtp.Localpart("other"),
			MailFromDomain:     "remote.example",
			RcptToLocalpart:    "☹",
			RcptToDomain:       "☺.example",
			MsgFromLocalpart:   "other",
			MsgFromDomain:      "remote.example",
			MsgFromOrgDomain:   "remote.example",
			EHLOValidated:      true,
			MailFromValidated:  true,
			MsgFromValidated:   true,
			EHLOValidation:     store.ValidationStrict,
			MailFromValidation: store.ValidationPass,
			MsgFromValidation:  store.ValidationStrict,
			DKIMDomains:        []string{"other.example"},
			Size:               int64(len(msg0)),
		}
		mf0 := tempfile()
		xcheckf(err, "creating temp file for delivery")
		_, err = fmt.Fprint(mf0, msg0)
		xcheckf(err, "writing deliver message to file")
		err = accTest2.DeliverMessage(c.log, tx, &m0, mf0, false, false, false, true)
		xcheckf(err, "add message to account test2")

		mf0name := mf0.Name()
		err = mf0.Close()
		xcheckf(err, "closing file")
		err = os.Remove(mf0name)
		xcheckf(err, "removing temp message file")

		err = tx.Get(&inbox)
		xcheckf(err, "get inbox")
		inbox.Add(m0.MailboxCounts())
		err = tx.Update(&inbox)
		xcheckf(err, "update inbox")

		sent, err := bstore.QueryTx[store.Mailbox](tx).FilterNonzero(store.Mailbox{Name: "Sent"}).Get()
		xcheckf(err, "looking up inbox")
		const prefix1 = "Extra: test\r\n"
		const msg1 = "From: <other@remote.example>\r\nTo: <☹@xn--74h.example>\r\nSubject: test\r\n\r\nthe message...\r\n"
		m1 := store.Message{
			MailboxID:         sent.ID,
			MailboxOrigID:     sent.ID,
			MailboxDestinedID: sent.ID,
			Flags:             store.Flags{Seen: true, Junk: true},
			Size:              int64(len(prefix1) + len(msg1)),
			MsgPrefix:         []byte(prefix1),
		}
		mf1 := tempfile()
		xcheckf(err, "creating temp file for delivery")
		_, err = fmt.Fprint(mf1, msg1)
		xcheckf(err, "writing deliver message to file")
		err = accTest2.DeliverMessage(c.log, tx, &m1, mf1, false, false, false, true)
		xcheckf(err, "add message to account test2")

		mf1name := mf1.Name()
		err = mf1.Close()
		xcheckf(err, "closing file")
		err = os.Remove(mf1name)
		xcheckf(err, "removing temp message file")

		err = tx.Get(&sent)
		xcheckf(err, "get sent")
		sent.Add(m1.MailboxCounts())
		err = tx.Update(&sent)
		xcheckf(err, "update sent")

		return nil
	})
	xcheckf(err, "write transaction with new message")
	err = accTest2.Close()
	xcheckf(err, "close account")
}
