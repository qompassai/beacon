// Package webadmin is a web app for the beacon administrator for viewing and changing
// the configuration, like creating/removing accounts, viewing DMARC and TLS
// reports, check DNS records for a domain, change the webserver configuration,
// etc.
package webadmin

import (
	"bufio"
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	_ "embed"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slog"

	"github.com/mjl-/adns"

	"github.com/mjl-/bstore"
	"github.com/mjl-/sherpa"
	"github.com/mjl-/sherpadoc"
	"github.com/mjl-/sherpaprom"

	"github.com/qompassai/beacon/config"
	"github.com/qompassai/beacon/dkim"
	"github.com/qompassai/beacon/dmarc"
	"github.com/qompassai/beacon/dmarcdb"
	"github.com/qompassai/beacon/dmarcrpt"
	"github.com/qompassai/beacon/dns"
	"github.com/qompassai/beacon/dnsbl"
	"github.com/qompassai/beacon/metrics"
	"github.com/qompassai/beacon/mlog"
	beacon "github.com/qompassai/beacon/beacon-"
	"github.com/qompassai/beacon/beaconvar"
	"github.com/qompassai/beacon/mtasts"
	"github.com/qompassai/beacon/mtastsdb"
	"github.com/qompassai/beacon/publicsuffix"
	"github.com/qompassai/beacon/queue"
	"github.com/qompassai/beacon/smtp"
	"github.com/qompassai/beacon/spf"
	"github.com/qompassai/beacon/store"
	"github.com/qompassai/beacon/tlsrpt"
	"github.com/qompassai/beacon/tlsrptdb"
	"github.com/qompassai/beacon/webauth"
)

var pkglog = mlog.New("webadmin", nil)

//go:embed api.json
var adminapiJSON []byte

//go:embed admin.html
var adminHTML []byte

//go:embed admin.js
var adminJS []byte

var webadminFile = &beacon.WebappFile{
	HTML:     adminHTML,
	JS:       adminJS,
	HTMLPath: filepath.FromSlash("webadmin/admin.html"),
	JSPath:   filepath.FromSlash("webadmin/admin.js"),
}

var adminDoc = mustParseAPI("admin", adminapiJSON)

func mustParseAPI(api string, buf []byte) (doc sherpadoc.Section) {
	err := json.Unmarshal(buf, &doc)
	if err != nil {
		pkglog.Fatalx("parsing webadmin api docs", err, slog.String("api", api))
	}
	return doc
}

var sherpaHandlerOpts *sherpa.HandlerOpts

func makeSherpaHandler(cookiePath string, isForwarded bool) (http.Handler, error) {
	return sherpa.NewHandler("/api/", beaconvar.Version, Admin{cookiePath, isForwarded}, &adminDoc, sherpaHandlerOpts)
}

func init() {
	collector, err := sherpaprom.NewCollector("beaconadmin", nil)
	if err != nil {
		pkglog.Fatalx("creating sherpa prometheus collector", err)
	}

	sherpaHandlerOpts = &sherpa.HandlerOpts{Collector: collector, AdjustFunctionNames: "none", NoCORS: true}
	// Just to validate.
	_, err = makeSherpaHandler("", false)
	if err != nil {
		pkglog.Fatalx("sherpa handler", err)
	}
}

// Handler returns a handler for the webadmin endpoints, customized for the
// cookiePath.
func Handler(cookiePath string, isForwarded bool) func(w http.ResponseWriter, r *http.Request) {
	sh, err := makeSherpaHandler(cookiePath, isForwarded)
	return func(w http.ResponseWriter, r *http.Request) {
		if err != nil {
			http.Error(w, "500 - internal server error - cannot handle requests", http.StatusInternalServerError)
			return
		}
		handle(sh, isForwarded, w, r)
	}
}

// Admin exports web API functions for the admin web interface. All its methods are
// exported under api/. Function calls require valid HTTP Authentication
// credentials of a user.
type Admin struct {
	cookiePath  string // From listener, for setting authentication cookies.
	isForwarded bool   // From listener, whether we look at X-Forwarded-* headers.
}

type ctxKey string

var requestInfoCtxKey ctxKey = "requestInfo"

type requestInfo struct {
	SessionToken store.SessionToken
	Response     http.ResponseWriter
	Request      *http.Request // For Proto and TLS connection state during message submit.
}

func handle(apiHandler http.Handler, isForwarded bool, w http.ResponseWriter, r *http.Request) {
	ctx := context.WithValue(r.Context(), mlog.CidKey, beacon.Cid())
	log := pkglog.WithContext(ctx).With(slog.String("adminauth", ""))

	// HTML/JS can be retrieved without authentication.
	if r.URL.Path == "/" {
		switch r.Method {
		case "GET", "HEAD":
			webadminFile.Serve(ctx, log, w, r)
		default:
			http.Error(w, "405 - method not allowed - use get", http.StatusMethodNotAllowed)
		}
		return
	}

	isAPI := strings.HasPrefix(r.URL.Path, "/api/")
	// Only allow POST for calls, they will not work cross-domain without CORS.
	if isAPI && r.URL.Path != "/api/" && r.Method != "POST" {
		http.Error(w, "405 - method not allowed - use post", http.StatusMethodNotAllowed)
		return
	}

	// All other URLs, except the login endpoint require some authentication.
	var sessionToken store.SessionToken
	if r.URL.Path != "/api/LoginPrep" && r.URL.Path != "/api/Login" {
		var ok bool
		_, sessionToken, _, ok = webauth.Check(ctx, log, webauth.Admin, "webadmin", isForwarded, w, r, isAPI, isAPI, false)
		if !ok {
			// Response has been written already.
			return
		}
	}

	if isAPI {
		reqInfo := requestInfo{sessionToken, w, r}
		ctx = context.WithValue(ctx, requestInfoCtxKey, reqInfo)
		apiHandler.ServeHTTP(w, r.WithContext(ctx))
		return
	}

	http.NotFound(w, r)
}

func xcheckf(ctx context.Context, err error, format string, args ...any) {
	if err == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	errmsg := fmt.Sprintf("%s: %s", msg, err)
	pkglog.WithContext(ctx).Errorx(msg, err)
	code := "server:error"
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		code = "user:error"
	}
	panic(&sherpa.Error{Code: code, Message: errmsg})
}

func xcheckuserf(ctx context.Context, err error, format string, args ...any) {
	if err == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	errmsg := fmt.Sprintf("%s: %s", msg, err)
	pkglog.WithContext(ctx).Errorx(msg, err)
	panic(&sherpa.Error{Code: "user:error", Message: errmsg})
}

// LoginPrep returns a login token, and also sets it as cookie. Both must be
// present in the call to Login.
func (w Admin) LoginPrep(ctx context.Context) string {
	log := pkglog.WithContext(ctx)
	reqInfo := ctx.Value(requestInfoCtxKey).(requestInfo)

	var data [8]byte
	_, err := cryptorand.Read(data[:])
	xcheckf(ctx, err, "generate token")
	loginToken := base64.RawURLEncoding.EncodeToString(data[:])

	webauth.LoginPrep(ctx, log, "webadmin", w.cookiePath, w.isForwarded, reqInfo.Response, reqInfo.Request, loginToken)

	return loginToken
}

// Login returns a session token for the credentials, or fails with error code
// "user:badLogin". Call LoginPrep to get a loginToken.
func (w Admin) Login(ctx context.Context, loginToken, password string) store.CSRFToken {
	log := pkglog.WithContext(ctx)
	reqInfo := ctx.Value(requestInfoCtxKey).(requestInfo)

	csrfToken, err := webauth.Login(ctx, log, webauth.Admin, "webadmin", w.cookiePath, w.isForwarded, reqInfo.Response, reqInfo.Request, loginToken, "", password)
	if _, ok := err.(*sherpa.Error); ok {
		panic(err)
	}
	xcheckf(ctx, err, "login")
	return csrfToken
}

// Logout invalidates the session token.
func (w Admin) Logout(ctx context.Context) {
	log := pkglog.WithContext(ctx)
	reqInfo := ctx.Value(requestInfoCtxKey).(requestInfo)

	err := webauth.Logout(ctx, log, webauth.Admin, "webadmin", w.cookiePath, w.isForwarded, reqInfo.Response, reqInfo.Request, "", reqInfo.SessionToken)
	xcheckf(ctx, err, "logout")
}

type Result struct {
	Errors       []string
	Warnings     []string
	Instructions []string
}

type DNSSECResult struct {
	Result
}

type IPRevCheckResult struct {
	Hostname dns.Domain          // This hostname, IPs must resolve back to this.
	IPNames  map[string][]string // IP to names.
	Result
}

type MX struct {
	Host string
	Pref int
	IPs  []string
}

type MXCheckResult struct {
	Records []MX
	Result
}

type TLSCheckResult struct {
	Result
}

type DANECheckResult struct {
	Result
}

type SPFRecord struct {
	spf.Record
}

type SPFCheckResult struct {
	DomainTXT    string
	DomainRecord *SPFRecord
	HostTXT      string
	HostRecord   *SPFRecord
	Result
}

type DKIMCheckResult struct {
	Records []DKIMRecord
	Result
}

type DKIMRecord struct {
	Selector string
	TXT      string
	Record   *dkim.Record
}

type DMARCRecord struct {
	dmarc.Record
}

type DMARCCheckResult struct {
	Domain string
	TXT    string
	Record *DMARCRecord
	Result
}

type TLSRPTRecord struct {
	tlsrpt.Record
}

type TLSRPTCheckResult struct {
	TXT    string
	Record *TLSRPTRecord
	Result
}

type MTASTSRecord struct {
	mtasts.Record
}
type MTASTSCheckResult struct {
	TXT        string
	Record     *MTASTSRecord
	PolicyText string
	Policy     *mtasts.Policy
	Result
}

type SRVConfCheckResult struct {
	SRVs map[string][]net.SRV // Service (e.g. "_imaps") to records.
	Result
}

type AutoconfCheckResult struct {
	ClientSettingsDomainIPs []string
	IPs                     []string
	Result
}

type AutodiscoverSRV struct {
	net.SRV
	IPs []string
}

type AutodiscoverCheckResult struct {
	Records []AutodiscoverSRV
	Result
}

// CheckResult is the analysis of a domain, its actual configuration (DNS, TLS,
// connectivity) and the beacon configuration. It includes configuration instructions
// (e.g. DNS records), and warnings and errors encountered.
type CheckResult struct {
	Domain       string
	DNSSEC       DNSSECResult
	IPRev        IPRevCheckResult
	MX           MXCheckResult
	TLS          TLSCheckResult
	DANE         DANECheckResult
	SPF          SPFCheckResult
	DKIM         DKIMCheckResult
	DMARC        DMARCCheckResult
	HostTLSRPT   TLSRPTCheckResult
	DomainTLSRPT TLSRPTCheckResult
	MTASTS       MTASTSCheckResult
	SRVConf      SRVConfCheckResult
	Autoconf     AutoconfCheckResult
	Autodiscover AutodiscoverCheckResult
}

// logPanic can be called with a defer from a goroutine to prevent the entire program from being shutdown in case of a panic.
func logPanic(ctx context.Context) {
	x := recover()
	if x == nil {
		return
	}
	pkglog.WithContext(ctx).Error("recover from panic", slog.Any("panic", x))
	debug.PrintStack()
	metrics.PanicInc(metrics.Webadmin)
}

// return IPs we may be listening on.
func xlistenIPs(ctx context.Context, receiveOnly bool) []net.IP {
	ips, err := beacon.IPs(ctx, receiveOnly)
	xcheckf(ctx, err, "listing ips")
	return ips
}

// return IPs from which we may be sending.
func xsendingIPs(ctx context.Context) []net.IP {
	ips, err := beacon.IPs(ctx, false)
	xcheckf(ctx, err, "listing ips")
	return ips
}

// CheckDomain checks the configuration for the domain, such as MX, SMTP STARTTLS,
// SPF, DKIM, DMARC, TLSRPT, MTASTS, autoconfig, autodiscover.
func (Admin) CheckDomain(ctx context.Context, domainName string) (r CheckResult) {
	// todo future: should run these checks without a DNS cache so recent changes are picked up.

	resolver := dns.StrictResolver{Pkg: "check", Log: pkglog.WithContext(ctx).Logger}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	nctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return checkDomain(nctx, resolver, dialer, domainName)
}

func unptr[T any](l []*T) []T {
	if l == nil {
		return nil
	}
	r := make([]T, len(l))
	for i, e := range l {
		r[i] = *e
	}
	return r
}

func checkDomain(ctx context.Context, resolver dns.Resolver, dialer *net.Dialer, domainName string) (r CheckResult) {
	log := pkglog.WithContext(ctx)

	domain, err := dns.ParseDomain(domainName)
	xcheckuserf(ctx, err, "parsing domain")

	domConf, ok := beacon.Conf.Domain(domain)
	if !ok {
		panic(&sherpa.Error{Code: "user:notFound", Message: "domain not found"})
	}

	listenIPs := xlistenIPs(ctx, true)
	isListenIP := func(ip net.IP) bool {
		for _, lip := range listenIPs {
			if ip.Equal(lip) {
				return true
			}
		}
		return false
	}

	addf := func(l *[]string, format string, args ...any) {
		*l = append(*l, fmt.Sprintf(format, args...))
	}

	// Host must be an absolute dns name, ending with a dot.
	lookupIPs := func(errors *[]string, host string) (ips []string, ourIPs, notOurIPs []net.IP, rerr error) {
		addrs, _, err := resolver.LookupHost(ctx, host)
		if err != nil {
			addf(errors, "Looking up %q: %s", host, err)
			return nil, nil, nil, err
		}
		for _, addr := range addrs {
			ip := net.ParseIP(addr)
			if ip == nil {
				addf(errors, "Bad IP %q", addr)
				continue
			}
			ips = append(ips, ip.String())
			if isListenIP(ip) {
				ourIPs = append(ourIPs, ip)
			} else {
				notOurIPs = append(notOurIPs, ip)
			}
		}
		return ips, ourIPs, notOurIPs, nil
	}

	checkTLS := func(errors *[]string, host string, ips []string, port string) {
		d := tls.Dialer{
			NetDialer: dialer,
			Config: &tls.Config{
				ServerName: host,
				MinVersion: tls.VersionTLS12, // ../rfc/8996:31 ../rfc/8997:66
				RootCAs:    beacon.Conf.Static.TLS.CertPool,
			},
		}
		for _, ip := range ips {
			conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(ip, port))
			if err != nil {
				addf(errors, "TLS connection to hostname %q, IP %q: %s", host, ip, err)
			} else {
				conn.Close()
			}
		}
	}

	// If at least one listener with SMTP enabled has unspecified NATed IPs, we'll skip
	// some checks related to these IPs.
	var isNAT, isUnspecifiedNAT bool
	for _, l := range beacon.Conf.Static.Listeners {
		if !l.SMTP.Enabled {
			continue
		}
		if l.IPsNATed {
			isUnspecifiedNAT = true
			isNAT = true
		}
		if len(l.NATIPs) > 0 {
			isNAT = true
		}
	}

	var wg sync.WaitGroup

	// DNSSEC
	wg.Add(1)
	go func() {
		defer logPanic(ctx)
		defer wg.Done()

		_, result, err := resolver.LookupNS(ctx, ".")
		if err != nil {
			addf(&r.DNSSEC.Errors, "Looking up NS for DNS root (.) to check support in resolver for DNSSEC-verification: %s", err)
		} else if !result.Authentic {
			addf(&r.DNSSEC.Warnings, `It looks like the DNS resolvers configured on your system do not verify DNSSEC, or aren't trusted (by having loopback IPs or through "options trust-ad" in /etc/resolv.conf).  Without DNSSEC, outbound delivery with SMTP used unprotected MX records, and SMTP STARTTLS connections cannot verify the TLS certificate with DANE (based on a public key in DNS), and will fallback to either MTA-STS for verification, or use "opportunistic TLS" with no certificate verification.`)
		} else {
			_, result, _ := resolver.LookupMX(ctx, domain.ASCII+".")
			if !result.Authentic {
				addf(&r.DNSSEC.Warnings, `DNS records for this domain (zone) are not DNSSEC-signed. Mail servers sending email to your domain, or receiving email from your domain, cannot verify that the MX/SPF/DKIM/DMARC/MTA-STS records they see are authentic.`)
			}
		}

		addf(&r.DNSSEC.Instructions, `Enable DNSSEC-signing of the DNS records of your domain (zone) at your DNS hosting provider.`)

		addf(&r.DNSSEC.Instructions, `If your DNS records are already DNSSEC-signed, you may not have a DNSSEC-verifying recursive resolver in use. Install unbound, and enable support for "extended DNS errors" (EDE), for example:

cat <<EOF >/etc/unbound/unbound.conf.d/ede.conf
server:
    ede: yes
    val-log-level: 2
EOF
`)
	}()

	// IPRev
	wg.Add(1)
	go func() {
		defer logPanic(ctx)
		defer wg.Done()

		// For each beacon.Conf.SpecifiedSMTPListenIPs and all NATIPs, and each IP for
		// beacon.Conf.HostnameDomain, check if they resolve back to the host name.
		hostIPs := map[dns.Domain][]net.IP{}
		ips, _, err := resolver.LookupIP(ctx, "ip", beacon.Conf.Static.HostnameDomain.ASCII+".")
		if err != nil {
			addf(&r.IPRev.Errors, "Looking up IPs for hostname: %s", err)
		}

		gatherMoreIPs := func(publicIPs []net.IP) {
		nextip:
			for _, ip := range publicIPs {
				for _, xip := range ips {
					if ip.Equal(xip) {
						continue nextip
					}
				}
				ips = append(ips, ip)
			}
		}
		if !isNAT {
			gatherMoreIPs(beacon.Conf.Static.SpecifiedSMTPListenIPs)
		}
		for _, l := range beacon.Conf.Static.Listeners {
			if !l.SMTP.Enabled {
				continue
			}
			var natips []net.IP
			for _, ip := range l.NATIPs {
				natips = append(natips, net.ParseIP(ip))
			}
			gatherMoreIPs(natips)
		}
		hostIPs[beacon.Conf.Static.HostnameDomain] = ips

		iplist := func(ips []net.IP) string {
			var ipstrs []string
			for _, ip := range ips {
				ipstrs = append(ipstrs, ip.String())
			}
			return strings.Join(ipstrs, ", ")
		}

		r.IPRev.Hostname = beacon.Conf.Static.HostnameDomain
		r.IPRev.Instructions = []string{
			fmt.Sprintf("Ensure IPs %s have reverse address %s.", iplist(ips), beacon.Conf.Static.HostnameDomain.ASCII),
		}

		// If we have a socks transport, also check its host and IP.
		for tname, t := range beacon.Conf.Static.Transports {
			if t.Socks != nil {
				hostIPs[t.Socks.Hostname] = append(hostIPs[t.Socks.Hostname], t.Socks.IPs...)
				instr := fmt.Sprintf("For SOCKS transport %s, ensure IPs %s have reverse address %s.", tname, iplist(t.Socks.IPs), t.Socks.Hostname)
				r.IPRev.Instructions = append(r.IPRev.Instructions, instr)
			}
		}

		type result struct {
			Host  dns.Domain
			IP    string
			Addrs []string
			Err   error
		}
		results := make(chan result)
		n := 0
		for host, ips := range hostIPs {
			for _, ip := range ips {
				n++
				s := ip.String()
				host := host
				go func() {
					addrs, _, err := resolver.LookupAddr(ctx, s)
					results <- result{host, s, addrs, err}
				}()
			}
		}
		r.IPRev.IPNames = map[string][]string{}
		for i := 0; i < n; i++ {
			lr := <-results
			host, addrs, ip, err := lr.Host, lr.Addrs, lr.IP, lr.Err
			if err != nil {
				addf(&r.IPRev.Errors, "Looking up reverse name for %s of %s: %v", ip, host, err)
				continue
			}
			if len(addrs) != 1 {
				addf(&r.IPRev.Errors, "Expected exactly 1 name for %s of %s, got %d (%v)", ip, host, len(addrs), addrs)
			}
			var match bool
			for i, a := range addrs {
				a = strings.TrimRight(a, ".")
				addrs[i] = a
				ad, err := dns.ParseDomain(a)
				if err != nil {
					addf(&r.IPRev.Errors, "Parsing reverse name %q for %s: %v", a, ip, err)
				}
				if ad == host {
					match = true
				}
			}
			if !match {
				addf(&r.IPRev.Errors, "Reverse name(s) %s for ip %s do not match hostname %s, which will cause other mail servers to reject incoming messages from this IP.", strings.Join(addrs, ","), ip, host)
			}
			r.IPRev.IPNames[ip] = addrs
		}

		// Linux machines are often initially set up with a loopback IP for the hostname in
		// /etc/hosts, presumably because it isn't known if their external IPs are static.
		// For mail servers, they should certainly be static. The quickstart would also
		// have warned about this, but could have been missed/ignored.
		for _, ip := range ips {
			if ip.IsLoopback() {
				addf(&r.IPRev.Errors, "Hostname %s resolves to loopback IP %s, this will likely prevent email delivery to local accounts from working. The loopback IP was probably configured in /etc/hosts at system installation time. Replace the loopback IP with your actual external IPs in /etc/hosts.", beacon.Conf.Static.HostnameDomain, ip.String())
			}
		}
	}()

	// MX
	wg.Add(1)
	go func() {
		defer logPanic(ctx)
		defer wg.Done()

		mxs, _, err := resolver.LookupMX(ctx, domain.ASCII+".")
		if err != nil {
			addf(&r.MX.Errors, "Looking up MX records for %s: %s", domain, err)
		}
		r.MX.Records = make([]MX, len(mxs))
		for i, mx := range mxs {
			r.MX.Records[i] = MX{mx.Host, int(mx.Pref), nil}
		}
		if len(mxs) == 1 && mxs[0].Host == "." {
			addf(&r.MX.Errors, `MX records consists of explicit null mx record (".") indicating that domain does not accept email.`)
			return
		}
		for i, mx := range mxs {
			ips, ourIPs, notOurIPs, err := lookupIPs(&r.MX.Errors, mx.Host)
			if err != nil {
				addf(&r.MX.Errors, "Looking up IPs for mx host %q: %s", mx.Host, err)
			}
			r.MX.Records[i].IPs = ips
			if isUnspecifiedNAT {
				continue
			}
			if len(ourIPs) == 0 {
				addf(&r.MX.Errors, "None of the IPs that mx %q points to is ours: %v", mx.Host, notOurIPs)
			} else if len(notOurIPs) > 0 {
				addf(&r.MX.Errors, "Some of the IPs that mx %q points to are not ours: %v", mx.Host, notOurIPs)
			}

		}
		r.MX.Instructions = []string{
			fmt.Sprintf("Ensure a DNS MX record like the following exists:\n\n\t%s MX 10 %s\n\nWithout the trailing dot, the name would be interpreted as relative to the domain.", domain.ASCII+".", beacon.Conf.Static.HostnameDomain.ASCII+"."),
		}
	}()

	// TLS, mostly checking certificate expiration and CA trust.
	// todo: should add checks about the listeners (which aren't specific to domains) somewhere else, not on the domain page with this checkDomain call. i.e. submissions, imap starttls, imaps.
	wg.Add(1)
	go func() {
		defer logPanic(ctx)
		defer wg.Done()

		// MTA-STS, autoconfig, autodiscover are checked in their sections.

		// Dial a single MX host with given IP and perform STARTTLS handshake.
		dialSMTPSTARTTLS := func(host, ip string) error {
			conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip, "25"))
			if err != nil {
				return err
			}
			defer func() {
				if conn != nil {
					conn.Close()
				}
			}()

			end := time.Now().Add(10 * time.Second)
			cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			err = conn.SetDeadline(end)
			log.WithContext(ctx).Check(err, "setting deadline")

			br := bufio.NewReader(conn)
			_, err = br.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading SMTP banner from remote: %s", err)
			}
			if _, err := fmt.Fprintf(conn, "EHLO beacontest\r\n"); err != nil {
				return fmt.Errorf("writing SMTP EHLO to remote: %s", err)
			}
			for {
				line, err := br.ReadString('\n')
				if err != nil {
					return fmt.Errorf("reading SMTP EHLO response from remote: %s", err)
				}
				if strings.HasPrefix(line, "250-") {
					continue
				}
				if strings.HasPrefix(line, "250 ") {
					break
				}
				return fmt.Errorf("unexpected response to SMTP EHLO from remote: %q", strings.TrimSuffix(line, "\r\n"))
			}
			if _, err := fmt.Fprintf(conn, "STARTTLS\r\n"); err != nil {
				return fmt.Errorf("writing SMTP STARTTLS to remote: %s", err)
			}
			line, err := br.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading response to SMTP STARTTLS from remote: %s", err)
			}
			if !strings.HasPrefix(line, "220 ") {
				return fmt.Errorf("SMTP STARTTLS response from remote not 220 OK: %q", strings.TrimSuffix(line, "\r\n"))
			}
			config := &tls.Config{
				ServerName: host,
				RootCAs:    beacon.Conf.Static.TLS.CertPool,
			}
			tlsconn := tls.Client(conn, config)
			if err := tlsconn.HandshakeContext(cctx); err != nil {
				return fmt.Errorf("TLS handshake after SMTP STARTTLS: %s", err)
			}
			cancel()
			conn.Close()
			conn = nil
			return nil
		}

		checkSMTPSTARTTLS := func() {
			// Initial errors are ignored, will already have been warned about by MX checks.
			mxs, _, err := resolver.LookupMX(ctx, domain.ASCII+".")
			if err != nil {
				return
			}
			if len(mxs) == 1 && mxs[0].Host == "." {
				return
			}
			for _, mx := range mxs {
				ips, _, _, err := lookupIPs(&r.MX.Errors, mx.Host)
				if err != nil {
					continue
				}

				for _, ip := range ips {
					if err := dialSMTPSTARTTLS(mx.Host, ip); err != nil {
						addf(&r.TLS.Errors, "SMTP connection with STARTTLS to MX hostname %q IP %s: %s", mx.Host, ip, err)
					}
				}
			}
		}

		checkSMTPSTARTTLS()

	}()

	// DANE
	wg.Add(1)
	go func() {
		defer logPanic(ctx)
		defer wg.Done()

		daneRecords := func(l config.Listener) map[string]struct{} {
			if l.TLS == nil {
				return nil
			}
			records := map[string]struct{}{}
			addRecord := func(privKey crypto.Signer) {
				spkiBuf, err := x509.MarshalPKIXPublicKey(privKey.Public())
				if err != nil {
					addf(&r.DANE.Errors, "marshal SubjectPublicKeyInfo for DANE record: %v", err)
					return
				}
				sum := sha256.Sum256(spkiBuf)
				r := adns.TLSA{
					Usage:     adns.TLSAUsageDANEEE,
					Selector:  adns.TLSASelectorSPKI,
					MatchType: adns.TLSAMatchTypeSHA256,
					CertAssoc: sum[:],
				}
				records[r.Record()] = struct{}{}
			}
			for _, privKey := range l.TLS.HostPrivateRSA2048Keys {
				addRecord(privKey)
			}
			for _, privKey := range l.TLS.HostPrivateECDSAP256Keys {
				addRecord(privKey)
			}
			return records
		}

		expectedDANERecords := func(host string) map[string]struct{} {
			for _, l := range beacon.Conf.Static.Listeners {
				if l.HostnameDomain.ASCII == host {
					return daneRecords(l)
				}
			}
			public := beacon.Conf.Static.Listeners["public"]
			if beacon.Conf.Static.HostnameDomain.ASCII == host && public.HostnameDomain.ASCII == "" {
				return daneRecords(public)
			}
			return nil
		}

		mxl, result, err := resolver.LookupMX(ctx, domain.ASCII+".")
		if err != nil {
			addf(&r.DANE.Errors, "Looking up MX hosts to check for DANE records: %s", err)
		} else {
			if !result.Authentic {
				addf(&r.DANE.Warnings, "DANE is inactive because MX records are not DNSSEC-signed.")
			}
			for _, mx := range mxl {
				expect := expectedDANERecords(mx.Host)

				tlsal, tlsaResult, err := resolver.LookupTLSA(ctx, 25, "tcp", mx.Host+".")
				if dns.IsNotFound(err) {
					if len(expect) > 0 {
						addf(&r.DANE.Errors, "No DANE records for MX host %s, expected: %s.", mx.Host, strings.Join(maps.Keys(expect), "; "))
					}
					continue
				} else if err != nil {
					addf(&r.DANE.Errors, "Looking up DANE records for MX host %s: %v", mx.Host, err)
					continue
				} else if !tlsaResult.Authentic && len(tlsal) > 0 {
					addf(&r.DANE.Errors, "DANE records exist for MX host %s, but are not DNSSEC-signed.", mx.Host)
				}

				extra := map[string]struct{}{}
				for _, e := range tlsal {
					s := e.Record()
					if _, ok := expect[s]; ok {
						delete(expect, s)
					} else {
						extra[s] = struct{}{}
					}
				}
				if len(expect) > 0 {
					l := maps.Keys(expect)
					sort.Strings(l)
					addf(&r.DANE.Errors, "Missing DANE records of type TLSA for MX host _25._tcp.%s: %s", mx.Host, strings.Join(l, "; "))
				}
				if len(extra) > 0 {
					l := maps.Keys(extra)
					sort.Strings(l)
					addf(&r.DANE.Errors, "Unexpected DANE records of type TLSA for MX host _25._tcp.%s: %s", mx.Host, strings.Join(l, "; "))
				}
			}
		}

		public := beacon.Conf.Static.Listeners["public"]
		pubDom := public.HostnameDomain
		if pubDom.ASCII == "" {
			pubDom = beacon.Conf.Static.HostnameDomain
		}
		records := maps.Keys(daneRecords(public))
		sort.Strings(records)
		if len(records) > 0 {
			instr := "Ensure the DNS records below exist. These records are for the whole machine, not per domain, so create them only once. Make sure DNSSEC is enabled, otherwise the records have no effect. The records indicate that a remote mail server trying to deliver email with SMTP (TCP port 25) must verify the TLS certificate with DANE-EE (3), based on the certificate public key (\"SPKI\", 1) that is SHA2-256-hashed (1) to the hexadecimal hash. DANE-EE verification means only the certificate or public key is verified, not whether the certificate is signed by a (centralized) certificate authority (CA), is expired, or matches the host name.\n\n"
			for _, r := range records {
				instr += fmt.Sprintf("\t_25._tcp.%s. TLSA %s\n", pubDom.ASCII, r)
			}
			addf(&r.DANE.Instructions, instr)
		}
	}()

	// SPF
	// todo: add warnings if we have Transports with submission? admin should ensure their IPs are in the SPF record. it may be an IP(net), or an include. that means we cannot easily check for it. and should we first check the transport can be used from this domain (or an account that has this domain?). also see DKIM.
	wg.Add(1)
	go func() {
		defer logPanic(ctx)
		defer wg.Done()

		// Verify a domain with the configured IPs that do SMTP.
		verifySPF := func(kind string, domain dns.Domain) (string, *SPFRecord, spf.Record) {
			_, txt, record, _, err := spf.Lookup(ctx, log.Logger, resolver, domain)
			if err != nil {
				addf(&r.SPF.Errors, "Looking up %s SPF record: %s", kind, err)
			}
			var xrecord *SPFRecord
			if record != nil {
				xrecord = &SPFRecord{*record}
			}

			spfr := spf.Record{
				Version: "spf1",
			}

			checkSPFIP := func(ip net.IP) {
				mechanism := "ip4"
				if ip.To4() == nil {
					mechanism = "ip6"
				}
				spfr.Directives = append(spfr.Directives, spf.Directive{Mechanism: mechanism, IP: ip})

				if record == nil {
					return
				}

				args := spf.Args{
					RemoteIP:          ip,
					MailFromLocalpart: "postmaster",
					MailFromDomain:    domain,
					HelloDomain:       dns.IPDomain{Domain: domain},
					LocalIP:           net.ParseIP("127.0.0.1"),
					LocalHostname:     dns.Domain{ASCII: "localhost"},
				}
				status, mechanism, expl, _, err := spf.Evaluate(ctx, log.Logger, record, resolver, args)
				if err != nil {
					addf(&r.SPF.Errors, "Evaluating IP %q against %s SPF record: %s", ip, kind, err)
				} else if status != spf.StatusPass {
					addf(&r.SPF.Errors, "IP %q does not pass %s SPF evaluation, status not \"pass\" but %q (mechanism %q, explanation %q)", ip, kind, status, mechanism, expl)
				}
			}

			for _, l := range beacon.Conf.Static.Listeners {
				if !l.SMTP.Enabled || l.IPsNATed {
					continue
				}
				ips := l.IPs
				if len(l.NATIPs) > 0 {
					ips = l.NATIPs
				}
				for _, ipstr := range ips {
					ip := net.ParseIP(ipstr)
					checkSPFIP(ip)
				}
			}
			for _, t := range beacon.Conf.Static.Transports {
				if t.Socks != nil {
					for _, ip := range t.Socks.IPs {
						checkSPFIP(ip)
					}
				}
			}

			spfr.Directives = append(spfr.Directives, spf.Directive{Qualifier: "-", Mechanism: "all"})
			return txt, xrecord, spfr
		}

		// Check SPF record for domain.
		var dspfr spf.Record
		r.SPF.DomainTXT, r.SPF.DomainRecord, dspfr = verifySPF("domain", domain)
		// todo: possibly check all hosts for MX records? assuming they are also sending mail servers.
		r.SPF.HostTXT, r.SPF.HostRecord, _ = verifySPF("host", beacon.Conf.Static.HostnameDomain)

		dtxt, err := dspfr.Record()
		if err != nil {
			addf(&r.SPF.Errors, "Making SPF record for instructions: %s", err)
		}
		domainspf := fmt.Sprintf("%s TXT %s", domain.ASCII+".", beacon.TXTStrings(dtxt))

		// Check SPF record for sending host. ../rfc/7208:2263 ../rfc/7208:2287
		hostspf := fmt.Sprintf(`%s TXT "v=spf1 a -all"`, beacon.Conf.Static.HostnameDomain.ASCII+".")

		addf(&r.SPF.Instructions, "Ensure DNS TXT records like the following exists:\n\n\t%s\n\t%s\n\nIf you have an existing mail setup, with other hosts also sending mail for you domain, you should add those IPs as well. You could replace \"-all\" with \"~all\" to treat mail sent from unlisted IPs as \"softfail\", or with \"?all\" for \"neutral\".", domainspf, hostspf)
	}()

	// DKIM
	// todo: add warnings if we have Transports with submission? admin should ensure DKIM records exist. we cannot easily check if they actually exist though. and should we first check the transport can be used from this domain (or an account that has this domain?). also see SPF.
	wg.Add(1)
	go func() {
		defer logPanic(ctx)
		defer wg.Done()

		var missing []string
		var haveEd25519 bool
		for sel, selc := range domConf.DKIM.Selectors {
			if _, ok := selc.Key.(ed25519.PrivateKey); ok {
				haveEd25519 = true
			}

			_, record, txt, _, err := dkim.Lookup(ctx, log.Logger, resolver, selc.Domain, domain)
			if err != nil {
				missing = append(missing, sel)
				if errors.Is(err, dkim.ErrNoRecord) {
					addf(&r.DKIM.Errors, "No DKIM DNS record for selector %q.", sel)
				} else if errors.Is(err, dkim.ErrSyntax) {
					addf(&r.DKIM.Errors, "Parsing DKIM DNS record for selector %q: %s", sel, err)
				} else {
					addf(&r.DKIM.Errors, "Fetching DKIM record for selector %q: %s", sel, err)
				}
			}
			if txt != "" {
				r.DKIM.Records = append(r.DKIM.Records, DKIMRecord{sel, txt, record})
				pubKey := selc.Key.Public()
				var pk []byte
				switch k := pubKey.(type) {
				case *rsa.PublicKey:
					var err error
					pk, err = x509.MarshalPKIXPublicKey(k)
					if err != nil {
						addf(&r.DKIM.Errors, "Marshal public key for %q to compare against DNS: %s", sel, err)
						continue
					}
				case ed25519.PublicKey:
					pk = []byte(k)
				default:
					addf(&r.DKIM.Errors, "Internal error: unknown public key type %T.", pubKey)
					continue
				}

				if record != nil && !bytes.Equal(record.Pubkey, pk) {
					addf(&r.DKIM.Errors, "For selector %q, the public key in DKIM DNS TXT record does not match with configured private key.", sel)
					missing = append(missing, sel)
				}
			}
		}
		if len(domConf.DKIM.Selectors) == 0 {
			addf(&r.DKIM.Errors, "No DKIM configuration, add a key to the configuration file, and instructions for DNS records will appear here.")
		} else if !haveEd25519 {
			addf(&r.DKIM.Warnings, "Consider adding an ed25519 key: the keys are smaller, the cryptography faster and more modern.")
		}
		instr := ""
		for _, sel := range missing {
			dkimr := dkim.Record{
				Version:   "DKIM1",
				Hashes:    []string{"sha256"},
				PublicKey: domConf.DKIM.Selectors[sel].Key.Public(),
			}
			switch dkimr.PublicKey.(type) {
			case *rsa.PublicKey:
			case ed25519.PublicKey:
				dkimr.Key = "ed25519"
			default:
				addf(&r.DKIM.Errors, "Internal error: unknown public key type %T.", dkimr.PublicKey)
			}
			txt, err := dkimr.Record()
			if err != nil {
				addf(&r.DKIM.Errors, "Making DKIM record for instructions: %s", err)
				continue
			}
			instr += fmt.Sprintf("\n\t%s._domainkey TXT %s\n", sel, beacon.TXTStrings(txt))
		}
		if instr != "" {
			instr = "Ensure the following DNS record(s) exists, so mail servers receiving emails from this domain can verify the signatures in the mail headers:\n" + instr
			addf(&r.DKIM.Instructions, "%s", instr)
		}
	}()

	// DMARC
	wg.Add(1)
	go func() {
		defer logPanic(ctx)
		defer wg.Done()

		_, dmarcDomain, record, txt, _, err := dmarc.Lookup(ctx, log.Logger, resolver, domain)
		if err != nil {
			addf(&r.DMARC.Errors, "Looking up DMARC record: %s", err)
		} else if record == nil {
			addf(&r.DMARC.Errors, "No DMARC record")
		}
		r.DMARC.Domain = dmarcDomain.Name()
		r.DMARC.TXT = txt
		if record != nil {
			r.DMARC.Record = &DMARCRecord{*record}
		}
		if record != nil && record.Policy == "none" {
			addf(&r.DMARC.Warnings, "DMARC policy is in test mode (p=none), do not forget to change to p=reject or p=quarantine after test period has been completed.")
		}
		if record != nil && record.SubdomainPolicy == "none" {
			addf(&r.DMARC.Warnings, "DMARC subdomain policy is in test mode (sp=none), do not forget to change to sp=reject or sp=quarantine after test period has been completed.")
		}
		if record != nil && len(record.AggregateReportAddresses) == 0 {
			addf(&r.DMARC.Warnings, "It is recommended you specify you would like aggregate reports about delivery success in the DMARC record, see instructions.")
		}

		dmarcr := dmarc.DefaultRecord
		dmarcr.Policy = "reject"

		var extInstr string
		if domConf.DMARC != nil {
			// If the domain is in a different Organizational Domain, the receiving domain
			// needs a special DNS record to opt-in to receiving reports. We check for that
			// record.
			// ../rfc/7489:1541
			orgDom := publicsuffix.Lookup(ctx, log.Logger, domain)
			destOrgDom := publicsuffix.Lookup(ctx, log.Logger, domConf.DMARC.DNSDomain)
			if orgDom != destOrgDom {
				accepts, status, _, _, _, err := dmarc.LookupExternalReportsAccepted(ctx, log.Logger, resolver, domain, domConf.DMARC.DNSDomain)
				if status != dmarc.StatusNone {
					addf(&r.DMARC.Errors, "Checking if external destination accepts reports: %s", err)
				} else if !accepts {
					addf(&r.DMARC.Errors, "External destination does not accept reports (%s)", err)
				}
				extInstr = fmt.Sprintf("Ensure a DNS TXT record exists in the domain of the destination address to opt-in to receiving reports from this domain:\n\n\t%s._report._dmarc.%s. TXT \"v=DMARC1;\"\n\n", domain.ASCII, domConf.DMARC.DNSDomain.ASCII)
			}

			uri := url.URL{
				Scheme: "mailto",
				Opaque: smtp.NewAddress(domConf.DMARC.ParsedLocalpart, domConf.DMARC.DNSDomain).Pack(false),
			}
			uristr := uri.String()
			dmarcr.AggregateReportAddresses = []dmarc.URI{
				{Address: uristr, MaxSize: 10, Unit: "m"},
			}

			if record != nil {
				found := false
				for _, addr := range record.AggregateReportAddresses {
					if addr.Address == uristr {
						found = true
						break
					}
				}
				if !found {
					addf(&r.DMARC.Errors, "Configured DMARC reporting address is not present in record.")
				}
			}
		} else {
			addf(&r.DMARC.Instructions, `Configure a DMARC destination in domain in config file.`)
		}
		instr := fmt.Sprintf("Ensure a DNS TXT record like the following exists:\n\n\t_dmarc TXT %s\n\nYou can start with testing mode by replacing p=reject with p=none. You can also request for the policy to be applied to a percentage of emails instead of all, by adding pct=X, with X between 0 and 100. Keep in mind that receiving mail servers will apply some anti-spam assessment regardless of the policy and whether it is applied to the message. The ruf= part requests daily aggregate reports to be sent to the specified address, which is automatically configured and reports automatically analyzed.", beacon.TXTStrings(dmarcr.String()))
		addf(&r.DMARC.Instructions, instr)
		if extInstr != "" {
			addf(&r.DMARC.Instructions, extInstr)
		}
	}()

	checkTLSRPT := func(result *TLSRPTCheckResult, dom dns.Domain, address smtp.Address, isHost bool) {
		defer logPanic(ctx)
		defer wg.Done()

		record, txt, err := tlsrpt.Lookup(ctx, log.Logger, resolver, dom)
		if err != nil {
			addf(&result.Errors, "Looking up TLSRPT record: %s", err)
		}
		result.TXT = txt
		if record != nil {
			result.Record = &TLSRPTRecord{*record}
		}

		instr := `TLSRPT is an opt-in mechanism to request feedback about TLS connectivity from remote SMTP servers when they connect to us. It allows detecting delivery problems and unwanted downgrades to plaintext SMTP connections. With TLSRPT you configure an email address to which reports should be sent. Remote SMTP servers will send a report once a day with the number of successful connections, and the number of failed connections including details that should help debugging/resolving any issues. Both the mail host (e.g. mail.domain.example) and a recipient domain (e.g. domain.example, with an MX record pointing to mail.domain.example) can have a TLSRPT record. The TLSRPT record for the hosts is for reporting about DANE, the TLSRPT record for the domain is for MTA-STS.`
		var zeroaddr smtp.Address
		if address != zeroaddr {
			// TLSRPT does not require validation of reporting addresses outside the domain.
			// ../rfc/8460:1463
			uri := url.URL{
				Scheme: "mailto",
				Opaque: address.Pack(false),
			}
			rua := tlsrpt.RUA(uri.String())
			tlsrptr := &tlsrpt.Record{
				Version: "TLSRPTv1",
				RUAs:    [][]tlsrpt.RUA{{rua}},
			}
			instr += fmt.Sprintf(`

Ensure a DNS TXT record like the following exists:

	_smtp._tls TXT %s
`, beacon.TXTStrings(tlsrptr.String()))

			if err == nil {
				found := false
			RUA:
				for _, l := range record.RUAs {
					for _, e := range l {
						if e == rua {
							found = true
							break RUA
						}
					}
				}
				if !found {
					addf(&result.Errors, `Configured reporting address is not present in TLSRPT record.`)
				}
			}

		} else if isHost {
			addf(&result.Errors, `Configure a host TLSRPT localpart in static beacon.conf config file.`)
		} else {
			addf(&result.Errors, `Configure a domain TLSRPT destination in domains.conf config file.`)
		}
		addf(&result.Instructions, instr)
	}

	// Host TLSRPT
	wg.Add(1)
	var hostTLSRPTAddr smtp.Address
	if beacon.Conf.Static.HostTLSRPT.Localpart != "" {
		hostTLSRPTAddr = smtp.NewAddress(beacon.Conf.Static.HostTLSRPT.ParsedLocalpart, beacon.Conf.Static.HostnameDomain)
	}
	go checkTLSRPT(&r.HostTLSRPT, beacon.Conf.Static.HostnameDomain, hostTLSRPTAddr, true)

	// Domain TLSRPT
	wg.Add(1)
	var domainTLSRPTAddr smtp.Address
	if domConf.TLSRPT != nil {
		domainTLSRPTAddr = smtp.NewAddress(domConf.TLSRPT.ParsedLocalpart, domain)
	}
	go checkTLSRPT(&r.DomainTLSRPT, domain, domainTLSRPTAddr, false)

	// MTA-STS
	wg.Add(1)
	go func() {
		defer logPanic(ctx)
		defer wg.Done()

		record, txt, err := mtasts.LookupRecord(ctx, log.Logger, resolver, domain)
		if err != nil {
			addf(&r.MTASTS.Errors, "Looking up MTA-STS record: %s", err)
		}
		r.MTASTS.TXT = txt
		if record != nil {
			r.MTASTS.Record = &MTASTSRecord{*record}
		}

		policy, text, err := mtasts.FetchPolicy(ctx, log.Logger, domain)
		if err != nil {
			addf(&r.MTASTS.Errors, "Fetching MTA-STS policy: %s", err)
		} else if policy.Mode == mtasts.ModeNone {
			addf(&r.MTASTS.Warnings, "MTA-STS policy is present, but does not require TLS.")
		} else if policy.Mode == mtasts.ModeTesting {
			addf(&r.MTASTS.Warnings, "MTA-STS policy is in testing mode, do not forget to change to mode enforce after testing period.")
		}
		r.MTASTS.PolicyText = text
		r.MTASTS.Policy = policy
		if policy != nil && policy.Mode != mtasts.ModeNone {
			if !policy.Matches(beacon.Conf.Static.HostnameDomain) {
				addf(&r.MTASTS.Warnings, "Configured hostname is missing from policy MX list.")
			}
			if policy.MaxAgeSeconds <= 24*3600 {
				addf(&r.MTASTS.Warnings, "Policy has a MaxAge of less than 1 day. For stable configurations, the recommended period is in weeks.")
			}

			mxl, _, _ := resolver.LookupMX(ctx, domain.ASCII+".")
			// We do not check for errors, the MX check will complain about mx errors, we assume we will get the same error here.
			mxs := map[dns.Domain]struct{}{}
			for _, mx := range mxl {
				d, err := dns.ParseDomain(strings.TrimSuffix(mx.Host, "."))
				if err != nil {
					addf(&r.MTASTS.Warnings, "MX record %q is invalid: %s", mx.Host, err)
					continue
				}
				mxs[d] = struct{}{}
			}
			for mx := range mxs {
				if !policy.Matches(mx) {
					addf(&r.MTASTS.Warnings, "MX record %q does not match MTA-STS policy MX list.", mx)
				}
			}
			for _, mx := range policy.MX {
				if mx.Wildcard {
					continue
				}
				if _, ok := mxs[mx.Domain]; !ok {
					addf(&r.MTASTS.Warnings, "MX %q in MTA-STS policy is not in MX record.", mx)
				}
			}
		}

		intro := `MTA-STS is an opt-in mechanism to signal to remote SMTP servers which MX records are valid and that they must use the STARTTLS command and verify the TLS connection. Email servers should already be using STARTTLS to protect communication, but active attackers can, and have in the past, removed the indication of support for the optional STARTTLS support from SMTP sessions, or added additional MX records in DNS responses. MTA-STS protects against compromised DNS and compromised plaintext SMTP sessions, but not against compromised internet PKI infrastructure. If an attacker controls a certificate authority, and is willing to use it, MTA-STS does not prevent an attack. MTA-STS does not protect against attackers on first contact with a domain. Only on subsequent contacts, with MTA-STS policies in the cache, can attacks can be detected.

After enabling MTA-STS for this domain, remote SMTP servers may still deliver in plain text, without TLS-protection. MTA-STS is an opt-in mechanism, not all servers support it yet.

You can opt-in to MTA-STS by creating a DNS record, _mta-sts.<domain>, and serving a policy at https://mta-sts.<domain>/.well-known/mta-sts.txt. Beacon will serve the policy, you must create the DNS records.

You can start with a policy in "testing" mode. Remote SMTP servers will apply the MTA-STS policy, but not abort delivery in case of failure. Instead, you will receive a report if you have TLSRPT configured. By starting in testing mode for a representative period, verifying all mail can be deliverd, you can safely switch to "enforce" mode. While in enforce mode, plaintext deliveries to beacon are refused.

The _mta-sts DNS TXT record has an "id" field. The id serves as a version of the policy. A policy specifies the mode: none, testing, enforce. For "none", no TLS is required. A policy has a "max age", indicating how long the policy can be cached. Allowing the policy to be cached for a long time provides stronger counter measures to active attackers, but reduces configuration change agility. After enabling "enforce" mode, remote SMTP servers may and will cache your policy for as long as "max age" was configured. Keep this in mind when enabling/disabling MTA-STS. To disable MTA-STS after having it enabled, publish a new record with mode "none" until all past policy expiration times have passed.

When enabling MTA-STS, or updating a policy, always update the policy first (through a configuration change and reload/restart), and the DNS record second.
`
		addf(&r.MTASTS.Instructions, intro)

		addf(&r.MTASTS.Instructions, `Enable a policy through the configuration file. For new deployments, it is best to start with mode "testing" while enabling TLSRPT. Start with a short "max_age", so updates to your policy are picked up quickly. When confidence in the deployment is high enough, switch to "enforce" mode and a longer "max age". A max age in the order of weeks is recommended. If you foresee a change to your setup in the future, requiring different policies or MX records, you may want to dial back the "max age" ahead of time, similar to how you would handle TTL's in DNS record updates.`)

		host := fmt.Sprintf("Ensure DNS CNAME/A/AAAA records exist that resolve mta-sts.%s to this mail server. For example:\n\n\t%s CNAME %s\n\n", domain.ASCII, "mta-sts."+domain.ASCII+".", beacon.Conf.Static.HostnameDomain.ASCII+".")
		addf(&r.MTASTS.Instructions, host)

		mtastsr := mtasts.Record{
			Version: "STSv1",
			ID:      time.Now().Format("20060102T150405"),
		}
		dns := fmt.Sprintf("Ensure a DNS TXT record like the following exists:\n\n\t_mta-sts TXT %s\n\nConfigure the ID in the configuration file, it must be of the form [a-zA-Z0-9]{1,31}. It represents the version of the policy. For each policy change, you must change the ID to a new unique value. You could use a timestamp like 20220621T123000. When this field exists, an SMTP server will fetch a policy at https://mta-sts.%s/.well-known/mta-sts.txt. This policy is served by beacon.", beacon.TXTStrings(mtastsr.String()), domain.Name())
		addf(&r.MTASTS.Instructions, dns)
	}()

	// SRVConf
	wg.Add(1)
	go func() {
		defer logPanic(ctx)
		defer wg.Done()

		type srvReq struct {
			name string
			port uint16
			host string
			srvs []*net.SRV
			err  error
		}

		// We'll assume if any submissions is configured, it is public. Same for imap. And
		var submissions, imaps bool
		for _, l := range beacon.Conf.Static.Listeners {
			if l.TLS != nil && l.Submissions.Enabled {
				submissions = true
			}
			if l.TLS != nil && l.IMAPS.Enabled {
				imaps = true
			}
		}
		srvhost := func(ok bool) string {
			if ok {
				return beacon.Conf.Static.HostnameDomain.ASCII + "."
			}
			return "."
		}
		var reqs = []srvReq{
			{name: "_submissions", port: 465, host: srvhost(submissions)},
			{name: "_submission", port: 587, host: srvhost(!submissions)},
			{name: "_imaps", port: 993, host: srvhost(imaps)},
			{name: "_imap", port: 143, host: srvhost(!imaps)},
			{name: "_pop3", port: 110, host: "."},
			{name: "_pop3s", port: 995, host: "."},
		}
		var srvwg sync.WaitGroup
		srvwg.Add(len(reqs))
		for i := range reqs {
			go func(i int) {
				defer srvwg.Done()
				_, reqs[i].srvs, _, reqs[i].err = resolver.LookupSRV(ctx, reqs[i].name[1:], "tcp", domain.ASCII+".")
			}(i)
		}
		srvwg.Wait()

		instr := "Ensure DNS records like the following exist:\n\n"
		r.SRVConf.SRVs = map[string][]net.SRV{}
		for _, req := range reqs {
			name := req.name + "_.tcp." + domain.ASCII
			instr += fmt.Sprintf("\t%s._tcp.%-*s SRV 0 1 %d %s\n", req.name, len("_submissions")-len(req.name)+len(domain.ASCII+"."), domain.ASCII+".", req.port, req.host)
			r.SRVConf.SRVs[req.name] = unptr(req.srvs)
			if err != nil {
				addf(&r.SRVConf.Errors, "Looking up SRV record %q: %s", name, err)
			} else if len(req.srvs) == 0 {
				addf(&r.SRVConf.Errors, "Missing SRV record %q", name)
			} else if len(req.srvs) != 1 || req.srvs[0].Target != req.host || req.srvs[0].Port != req.port {
				addf(&r.SRVConf.Errors, "Unexpected SRV record(s) for %q", name)
			}
		}
		addf(&r.SRVConf.Instructions, instr)
	}()

	// Autoconf
	wg.Add(1)
	go func() {
		defer logPanic(ctx)
		defer wg.Done()

		if domConf.ClientSettingsDomain != "" {
			addf(&r.Autoconf.Instructions, "Ensure a DNS CNAME record like the following exists:\n\n\t%s CNAME %s\n\nNote: the trailing dot is relevant, it makes the host name absolute instead of relative to the domain name.", domConf.ClientSettingsDNSDomain.ASCII+".", beacon.Conf.Static.HostnameDomain.ASCII+".")

			ips, ourIPs, notOurIPs, err := lookupIPs(&r.Autoconf.Errors, domConf.ClientSettingsDNSDomain.ASCII+".")
			if err != nil {
				addf(&r.Autoconf.Errors, "Looking up client settings DNS CNAME: %s", err)
			}
			r.Autoconf.ClientSettingsDomainIPs = ips
			if !isUnspecifiedNAT {
				if len(ourIPs) == 0 {
					addf(&r.Autoconf.Errors, "Client settings domain does not point to one of our IPs.")
				} else if len(notOurIPs) > 0 {
					addf(&r.Autoconf.Errors, "Client settings domain points to some IPs that are not ours: %v", notOurIPs)
				}
			}
		}

		addf(&r.Autoconf.Instructions, "Ensure a DNS CNAME record like the following exists:\n\n\tautoconfig.%s CNAME %s\n\nNote: the trailing dot is relevant, it makes the host name absolute instead of relative to the domain name.", domain.ASCII+".", beacon.Conf.Static.HostnameDomain.ASCII+".")

		host := "autoconfig." + domain.ASCII + "."
		ips, ourIPs, notOurIPs, err := lookupIPs(&r.Autoconf.Errors, host)
		if err != nil {
			addf(&r.Autoconf.Errors, "Looking up autoconfig host: %s", err)
			return
		}

		r.Autoconf.IPs = ips
		if !isUnspecifiedNAT {
			if len(ourIPs) == 0 {
				addf(&r.Autoconf.Errors, "Autoconfig does not point to one of our IPs.")
			} else if len(notOurIPs) > 0 {
				addf(&r.Autoconf.Errors, "Autoconfig points to some IPs that are not ours: %v", notOurIPs)
			}
		}

		checkTLS(&r.Autoconf.Errors, "autoconfig."+domain.ASCII, ips, "443")
	}()

	// Autodiscover
	wg.Add(1)
	go func() {
		defer logPanic(ctx)
		defer wg.Done()

		addf(&r.Autodiscover.Instructions, "Ensure DNS records like the following exist:\n\n\t_autodiscover._tcp.%s SRV 0 1 443 %s\n\tautoconfig.%s CNAME %s\n\nNote: the trailing dots are relevant, it makes the host names absolute instead of relative to the domain name.", domain.ASCII+".", beacon.Conf.Static.HostnameDomain.ASCII+".", domain.ASCII+".", beacon.Conf.Static.HostnameDomain.ASCII+".")

		_, srvs, _, err := resolver.LookupSRV(ctx, "autodiscover", "tcp", domain.ASCII+".")
		if err != nil {
			addf(&r.Autodiscover.Errors, "Looking up SRV record %q: %s", "autodiscover", err)
			return
		}
		match := false
		for _, srv := range srvs {
			ips, ourIPs, notOurIPs, err := lookupIPs(&r.Autodiscover.Errors, srv.Target)
			if err != nil {
				addf(&r.Autodiscover.Errors, "Looking up target %q from SRV record: %s", srv.Target, err)
				continue
			}
			if srv.Port != 443 {
				continue
			}
			match = true
			r.Autodiscover.Records = append(r.Autodiscover.Records, AutodiscoverSRV{*srv, ips})
			if !isUnspecifiedNAT {
				if len(ourIPs) == 0 {
					addf(&r.Autodiscover.Errors, "SRV target %q does not point to our IPs.", srv.Target)
				} else if len(notOurIPs) > 0 {
					addf(&r.Autodiscover.Errors, "SRV target %q points to some IPs that are not ours: %v", srv.Target, notOurIPs)
				}
			}

			checkTLS(&r.Autodiscover.Errors, strings.TrimSuffix(srv.Target, "."), ips, "443")
		}
		if !match {
			addf(&r.Autodiscover.Errors, "No SRV record for port 443 for https.")
		}
	}()

	wg.Wait()
	return
}

// Domains returns all configured domain names, in UTF-8 for IDNA domains.
func (Admin) Domains(ctx context.Context) []dns.Domain {
	l := []dns.Domain{}
	for _, s := range beacon.Conf.Domains() {
		d, _ := dns.ParseDomain(s)
		l = append(l, d)
	}
	return l
}

// Domain returns the dns domain for a (potentially unicode as IDNA) domain name.
func (Admin) Domain(ctx context.Context, domain string) dns.Domain {
	d, err := dns.ParseDomain(domain)
	xcheckuserf(ctx, err, "parse domain")
	_, ok := beacon.Conf.Domain(d)
	if !ok {
		xcheckuserf(ctx, errors.New("no such domain"), "looking up domain")
	}
	return d
}

// ParseDomain parses a domain, possibly an IDNA domain.
func (Admin) ParseDomain(ctx context.Context, domain string) dns.Domain {
	d, err := dns.ParseDomain(domain)
	xcheckuserf(ctx, err, "parse domain")
	return d
}

// DomainLocalparts returns the encoded localparts and accounts configured in domain.
func (Admin) DomainLocalparts(ctx context.Context, domain string) (localpartAccounts map[string]string) {
	d, err := dns.ParseDomain(domain)
	xcheckuserf(ctx, err, "parsing domain")
	_, ok := beacon.Conf.Domain(d)
	if !ok {
		xcheckuserf(ctx, errors.New("no such domain"), "looking up domain")
	}
	return beacon.Conf.DomainLocalparts(d)
}

// Accounts returns the names of all configured accounts.
func (Admin) Accounts(ctx context.Context) []string {
	l := beacon.Conf.Accounts()
	sort.Slice(l, func(i, j int) bool {
		return l[i] < l[j]
	})
	return l
}

// Account returns the parsed configuration of an account.
func (Admin) Account(ctx context.Context, account string) map[string]any {
	ac, ok := beacon.Conf.Account(account)
	if !ok {
		xcheckuserf(ctx, errors.New("no such account"), "looking up account")
	}

	// todo: should change sherpa to understand config.Account directly, with its anonymous structs.
	buf, err := json.Marshal(ac)
	xcheckf(ctx, err, "marshal to json")
	r := map[string]any{}
	err = json.Unmarshal(buf, &r)
	xcheckf(ctx, err, "unmarshal from json")

	return r
}

// ConfigFiles returns the paths and contents of the static and dynamic configuration files.
func (Admin) ConfigFiles(ctx context.Context) (staticPath, dynamicPath, static, dynamic string) {
	buf0, err := os.ReadFile(beacon.ConfigStaticPath)
	xcheckf(ctx, err, "read static config file")
	buf1, err := os.ReadFile(beacon.ConfigDynamicPath)
	xcheckf(ctx, err, "read dynamic config file")
	return beacon.ConfigStaticPath, beacon.ConfigDynamicPath, string(buf0), string(buf1)
}

// MTASTSPolicies returns all mtasts policies from the cache.
func (Admin) MTASTSPolicies(ctx context.Context) (records []mtastsdb.PolicyRecord) {
	records, err := mtastsdb.PolicyRecords(ctx)
	xcheckf(ctx, err, "fetching mtasts policies from database")
	return records
}

// TLSReports returns TLS reports overlapping with period start/end, for the given
// policy domain (or all domains if empty). The reports are sorted first by period
// end (most recent first), then by policy domain.
func (Admin) TLSReports(ctx context.Context, start, end time.Time, policyDomain string) (reports []tlsrptdb.TLSReportRecord) {
	var polDom dns.Domain
	if policyDomain != "" {
		var err error
		polDom, err = dns.ParseDomain(policyDomain)
		xcheckuserf(ctx, err, "parsing domain %q", policyDomain)
	}

	records, err := tlsrptdb.RecordsPeriodDomain(ctx, start, end, polDom)
	xcheckf(ctx, err, "fetching tlsrpt report records from database")
	sort.Slice(records, func(i, j int) bool {
		iend := records[i].Report.DateRange.End
		jend := records[j].Report.DateRange.End
		if iend == jend {
			return records[i].Domain < records[j].Domain
		}
		return iend.After(jend)
	})
	return records
}

// TLSReportID returns a single TLS report.
func (Admin) TLSReportID(ctx context.Context, domain string, reportID int64) tlsrptdb.TLSReportRecord {
	record, err := tlsrptdb.RecordID(ctx, reportID)
	if err == nil && record.Domain != domain {
		err = bstore.ErrAbsent
	}
	if err == bstore.ErrAbsent {
		xcheckuserf(ctx, err, "fetching tls report from database")
	}
	xcheckf(ctx, err, "fetching tls report from database")
	return record
}

// TLSRPTSummary presents TLS reporting statistics for a single domain
// over a period.
type TLSRPTSummary struct {
	PolicyDomain     dns.Domain
	Success          int64
	Failure          int64
	ResultTypeCounts map[tlsrpt.ResultType]int64
}

// TLSRPTSummaries returns a summary of received TLS reports overlapping with
// period start/end for one or all domains (when domain is empty).
// The returned summaries are ordered by domain name.
func (Admin) TLSRPTSummaries(ctx context.Context, start, end time.Time, policyDomain string) (domainSummaries []TLSRPTSummary) {
	var polDom dns.Domain
	if policyDomain != "" {
		var err error
		polDom, err = dns.ParseDomain(policyDomain)
		xcheckuserf(ctx, err, "parsing policy domain")
	}
	reports, err := tlsrptdb.RecordsPeriodDomain(ctx, start, end, polDom)
	xcheckf(ctx, err, "fetching tlsrpt reports from database")

	summaries := map[dns.Domain]TLSRPTSummary{}
	for _, r := range reports {
		dom, err := dns.ParseDomain(r.Domain)
		xcheckf(ctx, err, "parsing domain %q", r.Domain)

		sum := summaries[dom]
		sum.PolicyDomain = dom
		for _, result := range r.Report.Policies {
			sum.Success += result.Summary.TotalSuccessfulSessionCount
			sum.Failure += result.Summary.TotalFailureSessionCount
			for _, details := range result.FailureDetails {
				if sum.ResultTypeCounts == nil {
					sum.ResultTypeCounts = map[tlsrpt.ResultType]int64{}
				}
				sum.ResultTypeCounts[details.ResultType] += details.FailedSessionCount
			}
		}
		summaries[dom] = sum
	}
	sums := make([]TLSRPTSummary, 0, len(summaries))
	for _, sum := range summaries {
		sums = append(sums, sum)
	}
	sort.Slice(sums, func(i, j int) bool {
		return sums[i].PolicyDomain.Name() < sums[j].PolicyDomain.Name()
	})
	return sums
}

// DMARCReports returns DMARC reports overlapping with period start/end, for the
// given domain (or all domains if empty). The reports are sorted first by period
// end (most recent first), then by domain.
func (Admin) DMARCReports(ctx context.Context, start, end time.Time, domain string) (reports []dmarcdb.DomainFeedback) {
	reports, err := dmarcdb.RecordsPeriodDomain(ctx, start, end, domain)
	xcheckf(ctx, err, "fetching dmarc aggregate reports from database")
	sort.Slice(reports, func(i, j int) bool {
		iend := reports[i].ReportMetadata.DateRange.End
		jend := reports[j].ReportMetadata.DateRange.End
		if iend == jend {
			return reports[i].Domain < reports[j].Domain
		}
		return iend > jend
	})
	return reports
}

// DMARCReportID returns a single DMARC report.
func (Admin) DMARCReportID(ctx context.Context, domain string, reportID int64) (report dmarcdb.DomainFeedback) {
	report, err := dmarcdb.RecordID(ctx, reportID)
	if err == nil && report.Domain != domain {
		err = bstore.ErrAbsent
	}
	if err == bstore.ErrAbsent {
		xcheckuserf(ctx, err, "fetching dmarc aggregate report from database")
	}
	xcheckf(ctx, err, "fetching dmarc aggregate report from database")
	return report
}

// DMARCSummary presents DMARC aggregate reporting statistics for a single domain
// over a period.
type DMARCSummary struct {
	Domain                string
	Total                 int
	DispositionNone       int
	DispositionQuarantine int
	DispositionReject     int
	DKIMFail              int
	SPFFail               int
	PolicyOverrides       map[dmarcrpt.PolicyOverride]int
}

// DMARCSummaries returns a summary of received DMARC reports overlapping with
// period start/end for one or all domains (when domain is empty).
// The returned summaries are ordered by domain name.
func (Admin) DMARCSummaries(ctx context.Context, start, end time.Time, domain string) (domainSummaries []DMARCSummary) {
	reports, err := dmarcdb.RecordsPeriodDomain(ctx, start, end, domain)
	xcheckf(ctx, err, "fetching dmarc aggregate reports from database")
	summaries := map[string]DMARCSummary{}
	for _, r := range reports {
		sum := summaries[r.Domain]
		sum.Domain = r.Domain
		for _, record := range r.Records {
			n := record.Row.Count

			sum.Total += n

			switch record.Row.PolicyEvaluated.Disposition {
			case dmarcrpt.DispositionNone:
				sum.DispositionNone += n
			case dmarcrpt.DispositionQuarantine:
				sum.DispositionQuarantine += n
			case dmarcrpt.DispositionReject:
				sum.DispositionReject += n
			}

			if record.Row.PolicyEvaluated.DKIM == dmarcrpt.DMARCFail {
				sum.DKIMFail += n
			}
			if record.Row.PolicyEvaluated.SPF == dmarcrpt.DMARCFail {
				sum.SPFFail += n
			}

			for _, reason := range record.Row.PolicyEvaluated.Reasons {
				if sum.PolicyOverrides == nil {
					sum.PolicyOverrides = map[dmarcrpt.PolicyOverride]int{}
				}
				sum.PolicyOverrides[reason.Type] += n
			}
		}
		summaries[r.Domain] = sum
	}
	sums := make([]DMARCSummary, 0, len(summaries))
	for _, sum := range summaries {
		sums = append(sums, sum)
	}
	sort.Slice(sums, func(i, j int) bool {
		return sums[i].Domain < sums[j].Domain
	})
	return sums
}

// Reverse is the result of a reverse lookup.
type Reverse struct {
	Hostnames []string

	// In the future, we can add a iprev-validated host name, and possibly the IPs of the host names.
}

// LookupIP does a reverse lookup of ip.
func (Admin) LookupIP(ctx context.Context, ip string) Reverse {
	resolver := dns.StrictResolver{Pkg: "webadmin", Log: pkglog.WithContext(ctx).Logger}
	names, _, err := resolver.LookupAddr(ctx, ip)
	xcheckuserf(ctx, err, "looking up ip")
	return Reverse{names}
}

// DNSBLStatus returns the IPs from which outgoing connections may be made and
// their current status in DNSBLs that are configured. The IPs are typically the
// configured listen IPs, or otherwise IPs on the machines network interfaces, with
// internal/private IPs removed.
//
// The returned value maps IPs to per DNSBL statuses, where "pass" means not listed and
// anything else is an error string, e.g. "fail: ..." or "temperror: ...".
func (Admin) DNSBLStatus(ctx context.Context) map[string]map[string]string {
	log := mlog.New("webadmin", nil).WithContext(ctx)
	resolver := dns.StrictResolver{Pkg: "check", Log: log.Logger}
	return dnsblsStatus(ctx, log, resolver)
}

func dnsblsStatus(ctx context.Context, log mlog.Log, resolver dns.Resolver) map[string]map[string]string {
	// todo: check health before using dnsbl?
	var dnsbls []dns.Domain
	if l, ok := beacon.Conf.Static.Listeners["public"]; ok {
		for _, dnsbl := range l.SMTP.DNSBLs {
			zone, err := dns.ParseDomain(dnsbl)
			xcheckf(ctx, err, "parse dnsbl zone")
			dnsbls = append(dnsbls, zone)
		}
	}

	r := map[string]map[string]string{}
	for _, ip := range xsendingIPs(ctx) {
		if ip.IsLoopback() || ip.IsPrivate() {
			continue
		}
		ipstr := ip.String()
		r[ipstr] = map[string]string{}
		for _, zone := range dnsbls {
			status, expl, err := dnsbl.Lookup(ctx, log.Logger, resolver, zone, ip)
			result := string(status)
			if err != nil {
				result += ": " + err.Error()
			}
			if expl != "" {
				result += ": " + expl
			}
			r[ipstr][zone.LogString()] = result
		}
	}
	return r
}

// DomainRecords returns lines describing DNS records that should exist for the
// configured domain.
func (Admin) DomainRecords(ctx context.Context, domain string) []string {
	log := pkglog.WithContext(ctx)
	return DomainRecords(ctx, log, domain)
}

// DomainRecords is the implementation of API function Admin.DomainRecords, taking
// a logger.
func DomainRecords(ctx context.Context, log mlog.Log, domain string) []string {
	d, err := dns.ParseDomain(domain)
	xcheckuserf(ctx, err, "parsing domain")
	dc, ok := beacon.Conf.Domain(d)
	if !ok {
		xcheckuserf(ctx, errors.New("unknown domain"), "lookup domain")
	}
	resolver := dns.StrictResolver{Pkg: "webadmin", Log: pkglog.WithContext(ctx).Logger}
	_, result, err := resolver.LookupTXT(ctx, domain+".")
	if !dns.IsNotFound(err) {
		xcheckf(ctx, err, "looking up record to determine if dnssec is implemented")
	}

	var certIssuerDomainName, acmeAccountURI string
	public := beacon.Conf.Static.Listeners["public"]
	if public.TLS != nil && public.TLS.ACME != "" {
		acme, ok := beacon.Conf.Static.ACME[public.TLS.ACME]
		if ok && acme.Manager.Manager.Client != nil {
			certIssuerDomainName = acme.IssuerDomainName
			acc, err := acme.Manager.Manager.Client.GetReg(ctx, "")
			log.Check(err, "get public acme account")
			if err == nil {
				acmeAccountURI = acc.URI
			}
		}
	}

	records, err := beacon.DomainRecords(dc, d, result.Authentic, certIssuerDomainName, acmeAccountURI)
	xcheckf(ctx, err, "dns records")
	return records
}

// DomainAdd adds a new domain and reloads the configuration.
func (Admin) DomainAdd(ctx context.Context, domain, accountName, localpart string) {
	d, err := dns.ParseDomain(domain)
	xcheckuserf(ctx, err, "parsing domain")

	err = beacon.DomainAdd(ctx, d, accountName, smtp.Localpart(localpart))
	xcheckf(ctx, err, "adding domain")
}

// DomainRemove removes an existing domain and reloads the configuration.
func (Admin) DomainRemove(ctx context.Context, domain string) {
	d, err := dns.ParseDomain(domain)
	xcheckuserf(ctx, err, "parsing domain")

	err = beacon.DomainRemove(ctx, d)
	xcheckf(ctx, err, "removing domain")
}

// AccountAdd adds existing a new account, with an initial email address, and
// reloads the configuration.
func (Admin) AccountAdd(ctx context.Context, accountName, address string) {
	err := beacon.AccountAdd(ctx, accountName, address)
	xcheckf(ctx, err, "adding account")
}

// AccountRemove removes an existing account and reloads the configuration.
func (Admin) AccountRemove(ctx context.Context, accountName string) {
	err := beacon.AccountRemove(ctx, accountName)
	xcheckf(ctx, err, "removing account")
}

// AddressAdd adds a new address to the account, which must already exist.
func (Admin) AddressAdd(ctx context.Context, address, accountName string) {
	err := beacon.AddressAdd(ctx, address, accountName)
	xcheckf(ctx, err, "adding address")
}

// AddressRemove removes an existing address.
func (Admin) AddressRemove(ctx context.Context, address string) {
	err := beacon.AddressRemove(ctx, address)
	xcheckf(ctx, err, "removing address")
}

// SetPassword saves a new password for an account, invalidating the previous password.
// Sessions are not interrupted, and will keep working. New login attempts must use the new password.
// Password must be at least 8 characters.
func (Admin) SetPassword(ctx context.Context, accountName, password string) {
	log := pkglog.WithContext(ctx)
	if len(password) < 8 {
		panic(&sherpa.Error{Code: "user:error", Message: "password must be at least 8 characters"})
	}
	acc, err := store.OpenAccount(log, accountName)
	xcheckf(ctx, err, "open account")
	defer func() {
		err := acc.Close()
		log.WithContext(ctx).Check(err, "closing account")
	}()
	err = acc.SetPassword(log, password)
	xcheckf(ctx, err, "setting password")
}

// SetAccountLimits set new limits on outgoing messages for an account.
func (Admin) SetAccountLimits(ctx context.Context, accountName string, maxOutgoingMessagesPerDay, maxFirstTimeRecipientsPerDay int, maxMsgSize int64) {
	err := beacon.AccountLimitsSave(ctx, accountName, maxOutgoingMessagesPerDay, maxFirstTimeRecipientsPerDay, maxMsgSize)
	xcheckf(ctx, err, "saving account limits")
}

// ClientConfigsDomain returns configurations for email clients, IMAP and
// Submission (SMTP) for the domain.
func (Admin) ClientConfigsDomain(ctx context.Context, domain string) beacon.ClientConfigs {
	d, err := dns.ParseDomain(domain)
	xcheckuserf(ctx, err, "parsing domain")

	cc, err := beacon.ClientConfigsDomain(d)
	xcheckf(ctx, err, "client config for domain")
	return cc
}

// QueueList returns the messages currently in the outgoing queue.
func (Admin) QueueList(ctx context.Context) []queue.Msg {
	l, err := queue.List(ctx)
	xcheckf(ctx, err, "listing messages in queue")
	return l
}

// QueueSize returns the number of messages currently in the outgoing queue.
func (Admin) QueueSize(ctx context.Context) int {
	n, err := queue.Count(ctx)
	xcheckf(ctx, err, "listing messages in queue")
	return n
}

// QueueKick initiates delivery of a message from the queue and sets the transport
// to use for delivery.
func (Admin) QueueKick(ctx context.Context, id int64, transport string) {
	n, err := queue.Kick(ctx, id, "", "", &transport)
	if err == nil && n == 0 {
		err = errors.New("message not found")
	}
	xcheckf(ctx, err, "kick message in queue")
}

// QueueDrop removes a message from the queue.
func (Admin) QueueDrop(ctx context.Context, id int64) {
	log := pkglog.WithContext(ctx)
	n, err := queue.Drop(ctx, log, id, "", "")
	if err == nil && n == 0 {
		err = errors.New("message not found")
	}
	xcheckf(ctx, err, "drop message from queue")
}

// QueueSaveRequireTLS updates the requiretls field for a message in the queue,
// to be used for the next delivery.
func (Admin) QueueSaveRequireTLS(ctx context.Context, id int64, requireTLS *bool) {
	err := queue.SaveRequireTLS(ctx, id, requireTLS)
	xcheckf(ctx, err, "update requiretls for message in queue")
}

// LogLevels returns the current log levels.
func (Admin) LogLevels(ctx context.Context) map[string]string {
	m := map[string]string{}
	for pkg, level := range beacon.Conf.LogLevels() {
		s, ok := mlog.LevelStrings[level]
		if !ok {
			s = level.String()
		}
		m[pkg] = s
	}
	return m
}

// LogLevelSet sets a log level for a package.
func (Admin) LogLevelSet(ctx context.Context, pkg string, levelStr string) {
	level, ok := mlog.Levels[levelStr]
	if !ok {
		xcheckuserf(ctx, errors.New("unknown"), "lookup level")
	}
	beacon.Conf.LogLevelSet(pkglog.WithContext(ctx), pkg, level)
}

// LogLevelRemove removes a log level for a package, which cannot be the empty string.
func (Admin) LogLevelRemove(ctx context.Context, pkg string) {
	beacon.Conf.LogLevelRemove(pkglog.WithContext(ctx), pkg)
}

// CheckUpdatesEnabled returns whether checking for updates is enabled.
func (Admin) CheckUpdatesEnabled(ctx context.Context) bool {
	return beacon.Conf.Static.CheckUpdates
}

// WebserverConfig is the combination of WebDomainRedirects and WebHandlers
// from the domains.conf configuration file.
type WebserverConfig struct {
	WebDNSDomainRedirects [][2]dns.Domain // From server to frontend.
	WebDomainRedirects    [][2]string     // From frontend to server, it's not convenient to create dns.Domain in the frontend.
	WebHandlers           []config.WebHandler
}

// WebserverConfig returns the current webserver config
func (Admin) WebserverConfig(ctx context.Context) (conf WebserverConfig) {
	conf = webserverConfig()
	conf.WebDomainRedirects = nil
	return conf
}

func webserverConfig() WebserverConfig {
	r, l := beacon.Conf.WebServer()
	x := make([][2]dns.Domain, 0, len(r))
	xs := make([][2]string, 0, len(r))
	for k, v := range r {
		x = append(x, [2]dns.Domain{k, v})
		xs = append(xs, [2]string{k.Name(), v.Name()})
	}
	sort.Slice(x, func(i, j int) bool {
		return x[i][0].ASCII < x[j][0].ASCII
	})
	sort.Slice(xs, func(i, j int) bool {
		return xs[i][0] < xs[j][0]
	})
	return WebserverConfig{x, xs, l}
}

// WebserverConfigSave saves a new webserver config. If oldConf is not equal to
// the current config, an error is returned.
func (Admin) WebserverConfigSave(ctx context.Context, oldConf, newConf WebserverConfig) (savedConf WebserverConfig) {
	current := webserverConfig()
	webhandlersEqual := func() bool {
		if len(current.WebHandlers) != len(oldConf.WebHandlers) {
			return false
		}
		for i, wh := range current.WebHandlers {
			if !wh.Equal(oldConf.WebHandlers[i]) {
				return false
			}
		}
		return true
	}
	if !reflect.DeepEqual(oldConf.WebDNSDomainRedirects, current.WebDNSDomainRedirects) || !webhandlersEqual() {
		xcheckuserf(ctx, errors.New("config has changed"), "comparing old/current config")
	}

	// Convert to map, check that there are no duplicates here. The canonicalized
	// dns.Domain are checked again for uniqueness when parsing the config before
	// storing.
	domainRedirects := map[string]string{}
	for _, x := range newConf.WebDomainRedirects {
		if _, ok := domainRedirects[x[0]]; ok {
			xcheckuserf(ctx, errors.New("already present"), "checking redirect %s", x[0])
		}
		domainRedirects[x[0]] = x[1]
	}

	err := beacon.WebserverConfigSet(ctx, domainRedirects, newConf.WebHandlers)
	xcheckf(ctx, err, "saving webserver config")

	savedConf = webserverConfig()
	savedConf.WebDomainRedirects = nil
	return savedConf
}

// Transports returns the configured transports, for sending email.
func (Admin) Transports(ctx context.Context) map[string]config.Transport {
	return beacon.Conf.Static.Transports
}

// DMARCEvaluationStats returns a map of all domains with evaluations to a count of
// the evaluations and whether those evaluations will cause a report to be sent.
func (Admin) DMARCEvaluationStats(ctx context.Context) map[string]dmarcdb.EvaluationStat {
	stats, err := dmarcdb.EvaluationStats(ctx)
	xcheckf(ctx, err, "get evaluation stats")
	return stats
}

// DMARCEvaluationsDomain returns all evaluations for aggregate reports for the
// domain, sorted from oldest to most recent.
func (Admin) DMARCEvaluationsDomain(ctx context.Context, domain string) (dns.Domain, []dmarcdb.Evaluation) {
	dom, err := dns.ParseDomain(domain)
	xcheckf(ctx, err, "parsing domain")

	evals, err := dmarcdb.EvaluationsDomain(ctx, dom)
	xcheckf(ctx, err, "get evaluations for domain")
	return dom, evals
}

// DMARCRemoveEvaluations removes evaluations for a domain.
func (Admin) DMARCRemoveEvaluations(ctx context.Context, domain string) {
	dom, err := dns.ParseDomain(domain)
	xcheckf(ctx, err, "parsing domain")

	err = dmarcdb.RemoveEvaluationsDomain(ctx, dom)
	xcheckf(ctx, err, "removing evaluations for domain")
}

// DMARCSuppressAdd adds a reporting address to the suppress list. Outgoing
// reports will be suppressed for a period.
func (Admin) DMARCSuppressAdd(ctx context.Context, reportingAddress string, until time.Time, comment string) {
	addr, err := smtp.ParseAddress(reportingAddress)
	xcheckuserf(ctx, err, "parsing reporting address")

	ba := dmarcdb.SuppressAddress{ReportingAddress: addr.String(), Until: until, Comment: comment}
	err = dmarcdb.SuppressAdd(ctx, &ba)
	xcheckf(ctx, err, "adding address to suppresslist")
}

// DMARCSuppressList returns all reporting addresses on the suppress list.
func (Admin) DMARCSuppressList(ctx context.Context) []dmarcdb.SuppressAddress {
	l, err := dmarcdb.SuppressList(ctx)
	xcheckf(ctx, err, "listing reporting addresses in suppresslist")
	return l
}

// DMARCSuppressRemove removes a reporting address record from the suppress list.
func (Admin) DMARCSuppressRemove(ctx context.Context, id int64) {
	err := dmarcdb.SuppressRemove(ctx, id)
	xcheckf(ctx, err, "removing reporting address from suppresslist")
}

// DMARCSuppressExtend updates the until field of a suppressed reporting address record.
func (Admin) DMARCSuppressExtend(ctx context.Context, id int64, until time.Time) {
	err := dmarcdb.SuppressUpdate(ctx, id, until)
	xcheckf(ctx, err, "updating reporting address in suppresslist")
}

// TLSRPTResults returns all TLSRPT results in the database.
func (Admin) TLSRPTResults(ctx context.Context) []tlsrptdb.TLSResult {
	results, err := tlsrptdb.Results(ctx)
	xcheckf(ctx, err, "get results")
	return results
}

// TLSRPTResultsPolicyDomain returns the TLS results for a domain.
func (Admin) TLSRPTResultsDomain(ctx context.Context, isRcptDom bool, policyDomain string) (dns.Domain, []tlsrptdb.TLSResult) {
	dom, err := dns.ParseDomain(policyDomain)
	xcheckf(ctx, err, "parsing domain")

	if isRcptDom {
		results, err := tlsrptdb.ResultsRecipientDomain(ctx, dom)
		xcheckf(ctx, err, "get result for recipient domain")
		return dom, results
	}
	results, err := tlsrptdb.ResultsPolicyDomain(ctx, dom)
	xcheckf(ctx, err, "get result for policy domain")
	return dom, results
}

// LookupTLSRPTRecord looks up a TLSRPT record and returns the parsed form, original txt
// form from DNS, and error with the TLSRPT record as a string.
func (Admin) LookupTLSRPTRecord(ctx context.Context, domain string) (record *TLSRPTRecord, txt string, errstr string) {
	log := pkglog.WithContext(ctx)
	dom, err := dns.ParseDomain(domain)
	xcheckf(ctx, err, "parsing domain")

	resolver := dns.StrictResolver{Pkg: "webadmin", Log: log.Logger}
	r, txt, err := tlsrpt.Lookup(ctx, log.Logger, resolver, dom)
	if err != nil && (errors.Is(err, tlsrpt.ErrNoRecord) || errors.Is(err, tlsrpt.ErrMultipleRecords) || errors.Is(err, tlsrpt.ErrRecordSyntax) || errors.Is(err, tlsrpt.ErrDNS)) {
		errstr = err.Error()
		err = nil
	}
	xcheckf(ctx, err, "fetching tlsrpt record")

	if r != nil {
		record = &TLSRPTRecord{Record: *r}
	}

	return record, txt, errstr
}

// TLSRPTRemoveResults removes the TLS results for a domain for the given day. If
// day is empty, all results are removed.
func (Admin) TLSRPTRemoveResults(ctx context.Context, isRcptDom bool, domain string, day string) {
	dom, err := dns.ParseDomain(domain)
	xcheckf(ctx, err, "parsing domain")

	if isRcptDom {
		err = tlsrptdb.RemoveResultsRecipientDomain(ctx, dom, day)
		xcheckf(ctx, err, "removing tls results")
	} else {
		err = tlsrptdb.RemoveResultsPolicyDomain(ctx, dom, day)
		xcheckf(ctx, err, "removing tls results")
	}
}

// TLSRPTSuppressAdd adds a reporting address to the suppress list. Outgoing
// reports will be suppressed for a period.
func (Admin) TLSRPTSuppressAdd(ctx context.Context, reportingAddress string, until time.Time, comment string) {
	addr, err := smtp.ParseAddress(reportingAddress)
	xcheckuserf(ctx, err, "parsing reporting address")

	ba := tlsrptdb.TLSRPTSuppressAddress{ReportingAddress: addr.String(), Until: until, Comment: comment}
	err = tlsrptdb.SuppressAdd(ctx, &ba)
	xcheckf(ctx, err, "adding address to suppresslist")
}

// TLSRPTSuppressList returns all reporting addresses on the suppress list.
func (Admin) TLSRPTSuppressList(ctx context.Context) []tlsrptdb.TLSRPTSuppressAddress {
	l, err := tlsrptdb.SuppressList(ctx)
	xcheckf(ctx, err, "listing reporting addresses in suppresslist")
	return l
}

// TLSRPTSuppressRemove removes a reporting address record from the suppress list.
func (Admin) TLSRPTSuppressRemove(ctx context.Context, id int64) {
	err := tlsrptdb.SuppressRemove(ctx, id)
	xcheckf(ctx, err, "removing reporting address from suppresslist")
}

// TLSRPTSuppressExtend updates the until field of a suppressed reporting address record.
func (Admin) TLSRPTSuppressExtend(ctx context.Context, id int64, until time.Time) {
	err := tlsrptdb.SuppressUpdate(ctx, id, until)
	xcheckf(ctx, err, "updating reporting address in suppresslist")
}
