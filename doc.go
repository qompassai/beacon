/*
Command beacon is a modern, secure, full-featured, open source mail server for
low-maintenance self-hosted email.

# Commands

	beacon [-config config/beacon.conf] [-pedantic] ...
	beacon serve
	beacon quickstart [-existing-webserver] [-hostname host] user@domain [user | uid]
	beacon stop
	beacon setaccountpassword account
	beacon setadminpassword
	beacon loglevels [level [pkg]]
	beacon queue list
	beacon queue kick [-id id] [-todomain domain] [-recipient address] [-transport transport]
	beacon queue drop [-id id] [-todomain domain] [-recipient address]
	beacon queue dump id
	beacon import maildir accountname mailboxname maildir
	beacon import mbox accountname mailboxname mbox
	beacon export maildir dst-dir account-path [mailbox]
	beacon export mbox dst-dir account-path [mailbox]
	beacon localserve
	beacon help [command ...]
	beacon backup dest-dir
	beacon verifydata data-dir
	beacon config test
	beacon config dnscheck domain
	beacon config dnsrecords domain
	beacon config describe-domains >domains.conf
	beacon config describe-static >beacon.conf
	beacon config account add account address
	beacon config account rm account
	beacon config address add address account
	beacon config address rm address
	beacon config domain add domain account [localpart]
	beacon config domain rm domain
	beacon config describe-sendmail >/etc/beaconsubmit.conf
	beacon config printservice >beacon.service
	beacon config ensureacmehostprivatekeys
	beacon example [name]
	beacon checkupdate
	beacon cid cid
	beacon clientconfig domain
	beacon dane dial host:port
	beacon dane dialmx domain [destination-host]
	beacon dane makerecord usage selector matchtype [certificate.pem | publickey.pem | privatekey.pem]
	beacon dns lookup [ptr | mx | cname | ips | a | aaaa | ns | txt | srv | tlsa] name
	beacon dkim gened25519 >$selector._domainkey.$domain.ed25519.privatekey.pkcs8.pem
	beacon dkim genrsa >$selector._domainkey.$domain.rsa2048.privatekey.pkcs8.pem
	beacon dkim lookup selector domain
	beacon dkim txt <$selector._domainkey.$domain.key.pkcs8.pem
	beacon dkim verify message
	beacon dkim sign message
	beacon dmarc lookup domain
	beacon dmarc parsereportmsg message ...
	beacon dmarc verify remoteip mailfromaddress helodomain < message
	beacon dmarc checkreportaddrs domain
	beacon dnsbl check zone ip
	beacon dnsbl checkhealth zone
	beacon mtasts lookup domain
	beacon retrain accountname
	beacon sendmail [-Fname] [ignoredflags] [-t] [<message]
	beacon spf check domain ip
	beacon spf lookup domain
	beacon spf parse txtrecord
	beacon tlsrpt lookup domain
	beacon tlsrpt parsereportmsg message ...
	beacon version
	beacon bumpuidvalidity account [mailbox]
	beacon reassignuids account [mailboxid]
	beacon fixuidmeta account
	beacon fixmsgsize [account]
	beacon reparse [account]
	beacon ensureparsed account
	beacon recalculatemailboxcounts account
	beacon message parse message.eml
	beacon reassignthreads [account]

Many commands talk to a running beacon instance, through the ctl file in the data
directory. Specify the configuration file (that holds the path to the data
directory) through the -config flag or BEACONCONF environment variable.

# beacon serve

Start beacon, serving SMTP/IMAP/HTTPS.

Incoming email is accepted over SMTP. Email can be retrieved by users using
IMAP. HTTP listeners are started for the admin/account web interfaces, and for
automated TLS configuration. Missing essential TLS certificates are immediately
requested, other TLS certificates are requested on demand.

Only implemented on unix systems, not Windows.

	usage: beacon serve

# beacon quickstart

Quickstart generates configuration files and prints instructions to quickly set up a beacon instance.

Quickstart writes configuration files, prints initial admin and account
passwords, DNS records you should create. If you run it on Linux it writes a
systemd service file and prints commands to enable and start beacon as service.

The user or uid is optional, defaults to "beacon", and is the user or uid/gid beacon
will run as after initialization.

Quickstart assumes beacon will run on the machine you run quickstart on and uses
its host name and public IPs. On many systems the hostname is not a fully
qualified domain name, but only the first dns "label", e.g. "mail" in case of
"mail.example.org". If so, quickstart does a reverse DNS lookup to find the
hostname, and as fallback uses the label plus the domain of the email address
you specified. Use flag -hostname to explicitly specify the hostname beacon will
run on.

Mox is by far easiest to operate if you let it listen on port 443 (HTTPS) and
80 (HTTP). TLS will be fully automatic with ACME with Let's Encrypt.

You can run beacon along with an existing webserver, but because of MTA-STS and
autoconfig, you'll need to forward HTTPS traffic for two domains to beacon. Run
"beacon quickstart -existing-webserver ..." to generate configuration files and
instructions for configuring beacon along with an existing webserver.

But please first consider configuring beacon on port 443. It can itself serve
domains with HTTP/HTTPS, including with automatic TLS with ACME, is easily
configured through both configuration files and admin web interface, and can act
as a reverse proxy (and static file server for that matter), so you can forward
traffic to your existing backend applications. Look for "WebHandlers:" in the
output of "beacon config describe-domains" and see the output of "beacon example
webhandlers".

	usage: beacon quickstart [-existing-webserver] [-hostname host] user@domain [user | uid]
	  -existing-webserver
	    	use if a webserver is already running, so beacon won't listen on port 80 and 443; you'll have to provide tls certificates/keys, and configure the existing webserver as reverse proxy, forwarding requests to beacon.
	  -hostname string
	    	hostname beacon will run on, by default the hostname of the machine quickstart runs on; if specified, the IPs for the hostname are configured for the public listener

# beacon stop

Shut beacon down, giving connections maximum 3 seconds to stop before closing them.

While shutting down, new IMAP and SMTP connections will get a status response
indicating temporary unavailability. Existing connections will get a 3 second
period to finish their transaction and shut down. Under normal circumstances,
only IMAP has long-living connections, with the IDLE command to get notified of
new mail deliveries.

	usage: beacon stop

# beacon setaccountpassword

Set new password an account.

The password is read from stdin. Secrets derived from the password, but not the
password itself, are stored in the account database. The stored secrets are for
authentication with: scram-sha-256, scram-sha-1, cram-md5, plain text (bcrypt
hash).

The parameter is an account name, as configured under Accounts in domains.conf
and as present in the data/accounts/ directory, not a configured email address
for an account.

	usage: beacon setaccountpassword account

# beacon setadminpassword

Set a new admin password, for the web interface.

The password is read from stdin. Its bcrypt hash is stored in a file named
"adminpasswd" in the configuration directory.

	usage: beacon setadminpassword

# beacon loglevels

Print the log levels, or set a new default log level, or a level for the given package.

By default, a single log level applies to all logging in beacon. But for each
"pkg", an overriding log level can be configured. Examples of packages:
smtpserver, smtpclient, queue, imapserver, spf, dkim, dmarc, junk, message,
etc.

Specify a pkg and an empty level to clear the configured level for a package.

Valid labels: error, info, debug, trace, traceauth, tracedata.

	usage: beacon loglevels [level [pkg]]

# beacon queue list

List messages in the delivery queue.

This prints the message with its ID, last and next delivery attempts, last
error.

	usage: beacon queue list

# beacon queue kick

Schedule matching messages in the queue for immediate delivery.

Messages deliveries are normally attempted with exponential backoff. The first
retry after 7.5 minutes, and doubling each time. Kicking messages sets their
next scheduled attempt to now, it can cause delivery to fail earlier than
without rescheduling.

With the -transport flag, future delivery attempts are done using the specified
transport. Transports can be configured in beacon.conf, e.g. to submit to a remote
queue over SMTP.

	usage: beacon queue kick [-id id] [-todomain domain] [-recipient address] [-transport transport]
	  -id int
	    	id of message in queue
	  -recipient string
	    	recipient email address
	  -todomain string
	    	destination domain of messages
	  -transport string
	    	transport to use for the next delivery

# beacon queue drop

Remove matching messages from the queue.

Dangerous operation, this completely removes the message. If you want to store
the message, use "queue dump" before removing.

	usage: beacon queue drop [-id id] [-todomain domain] [-recipient address]
	  -id int
	    	id of message in queue
	  -recipient string
	    	recipient email address
	  -todomain string
	    	destination domain of messages

# beacon queue dump

Dump a message from the queue.

The message is printed to stdout and is in standard internet mail format.

	usage: beacon queue dump id

# beacon import maildir

Import a maildir into an account.

The mbox/maildir archive is accessed and imported by the running beacon process, so
it must have access to the archive files. The default suggested systemd service
file isolates beacon from most of the file system, with only the "data/" directory
accessible, so you may want to put the mbox/maildir archive files in a
directory like "data/import/" to make it available to beacon.

By default, messages will train the junk filter based on their flags and, if
"automatic junk flags" configuration is set, based on mailbox naming.

If the destination mailbox is the Sent mailbox, the recipients of the messages
are added to the message metadata, causing later incoming messages from these
recipients to be accepted, unless other reputation signals prevent that.

Users can also import mailboxes/messages through the account web page by
uploading a zip or tgz file with mbox and/or maildirs.

Mailbox flags, like "seen", "answered", will be imported. An optional
dovecot-keywords file can specify additional flags, like Forwarded/Junk/NotJunk.

	usage: beacon import maildir accountname mailboxname maildir

# beacon import mbox

Import an mbox into an account.

Using mbox is not recommended, maildir is a better defined format.

The mbox/maildir archive is accessed and imported by the running beacon process, so
it must have access to the archive files. The default suggested systemd service
file isolates beacon from most of the file system, with only the "data/" directory
accessible, so you may want to put the mbox/maildir archive files in a
directory like "data/import/" to make it available to beacon.

By default, messages will train the junk filter based on their flags and, if
"automatic junk flags" configuration is set, based on mailbox naming.

If the destination mailbox is the Sent mailbox, the recipients of the messages
are added to the message metadata, causing later incoming messages from these
recipients to be accepted, unless other reputation signals prevent that.

Users can also import mailboxes/messages through the account web page by
uploading a zip or tgz file with mbox and/or maildirs.

	usage: beacon import mbox accountname mailboxname mbox

# beacon export maildir

Export one or all mailboxes from an account in maildir format.

Export bypasses a running beacon instance. It opens the account mailbox/message
database file directly. This may block if a running beacon instance also has the
database open, e.g. for IMAP connections. To export from a running instance, use
the accounts web page.

	usage: beacon export maildir dst-dir account-path [mailbox]

# beacon export mbox

Export messages from one or all mailboxes in an account in mbox format.

Using mbox is not recommended. Maildir is a better format.

Export bypasses a running beacon instance. It opens the account mailbox/message
database file directly. This may block if a running beacon instance also has the
database open, e.g. for IMAP connections. To export from a running instance, use
the accounts web page.

For mbox export, "mboxrd" is used where message lines starting with the magic
"From " string are escaped by prepending a >. All ">*From " are escaped,
otherwise reconstructing the original could lose a ">".

	usage: beacon export mbox dst-dir account-path [mailbox]

# beacon localserve

Start a local SMTP/IMAP server that accepts all messages, useful when testing/developing software that sends email.

Localserve starts beacon with a configuration suitable for local email-related
software development/testing. It listens for SMTP/Submission(s), IMAP(s) and
HTTP(s), on the regular port numbers + 1000.

Data is stored in the system user's configuration directory under
"beacon-localserve", e.g. $HOME/.config/beacon-localserve/ on linux, but can be
overridden with the -dir flag. If the directory does not yet exist, it is
automatically initialized with configuration files, an account with email
address beacon@localhost and password beaconbeaconbeacon, and a newly generated self-signed
TLS certificate.

All incoming email to any address is accepted (if checks pass), unless the
recipient localpart ends with:

- "temperror": fail with a temporary error code
- "permerror": fail with a permanent error code
- [45][0-9][0-9]: fail with the specific error code
- "timeout": no response (for an hour)

If the localpart begins with "mailfrom" or "rcptto", the error is returned
during those commands instead of during "data".

	usage: beacon localserve
	  -dir string
	    	configuration storage directory (default "$userconfigdir/beacon-localserve")
	  -initonly
	    	write configuration files and exit
	  -ip string
	    	serve on this ip instead of default 127.0.0.1 and ::1. only used when writing configuration, at first launch.

# beacon help

Prints help about matching commands.

If multiple commands match, they are listed along with the first line of their help text.
If a single command matches, its usage and full help text is printed.

	usage: beacon help [command ...]

# beacon backup

Creates a backup of the data directory.

Backup creates consistent snapshots of the databases and message files and
copies other files in the data directory. Empty directories are not copied.
These files can then be stored elsewhere for long-term storage, or used to fall
back to should an upgrade fail. Simply copying files in the data directory
while beacon is running can result in unusable database files.

Message files never change (they are read-only, though can be removed) and are
hard-linked so they don't consume additional space. If hardlinking fails, for
example when the backup destination directory is on a different file system, a
regular copy is made. Using a destination directory like "data/tmp/backup"
increases the odds hardlinking succeeds: the default systemd service file
specifically mounts the data directory, causing attempts to hardlink outside it
to fail with an error about cross-device linking.

All files in the data directory that aren't recognized (i.e. other than known
database files, message files, an acme directory, the "tmp" directory, etc),
are stored, but with a warning.

A clean successful backup does not print any output by default. Use the
-verbose flag for details, including timing.

To restore a backup, first shut down beacon, move away the old data directory and
move an earlier backed up directory in its place, run "beacon verifydata",
possibly with the "-fix" option, and restart beacon. After the restore, you may
also want to run "beacon bumpuidvalidity" for each account for which messages in a
mailbox changed, to force IMAP clients to synchronize mailbox state.

Before upgrading, to check if the upgrade will likely succeed, first make a
backup, then use the new beacon binary to run "beacon verifydata" on the backup. This
can change the backup files (e.g. upgrade database files, move away
unrecognized message files), so you should make a new backup before actually
upgrading.

	usage: beacon backup dest-dir
	  -verbose
	    	print progress

# beacon verifydata

Verify the contents of a data directory, typically of a backup.

Verifydata checks all database files to see if they are valid BoltDB/bstore
databases. It checks that all messages in the database have a corresponding
on-disk message file and there are no unrecognized files. If option -fix is
specified, unrecognized message files are moved away. This may be needed after
a restore, because messages enqueued or delivered in the future may get those
message sequence numbers assigned and writing the message file would fail.
Consistency of message/mailbox UID, UIDNEXT and UIDVALIDITY is verified as
well.

Because verifydata opens the database files, schema upgrades may automatically
be applied. This can happen if you use a new beacon release. It is useful to run
"beacon verifydata" with a new binary before attempting an upgrade, but only on a
copy of the database files, as made with "beacon backup". Before upgrading, make a
new backup again since "beacon verifydata" may have upgraded the database files,
possibly making them potentially no longer readable by the previous version.

	usage: beacon verifydata data-dir
	  -fix
	    	fix fixable problems, such as moving away message files not referenced by their database
	  -skip-size-check
	    	skip the check for message size

# beacon config test

Parses and validates the configuration files.

If valid, the command exits with status 0. If not valid, all errors encountered
are printed.

	usage: beacon config test

# beacon config dnscheck

Check the DNS records with the configuration for the domain, and print any errors/warnings.

	usage: beacon config dnscheck domain

# beacon config dnsrecords

Prints annotated DNS records as zone file that should be created for the domain.

The zone file can be imported into existing DNS software. You should review the
DNS records, especially if your domain previously/currently has email
configured.

	usage: beacon config dnsrecords domain

# beacon config describe-domains

Prints an annotated empty configuration for use as domains.conf.

The domains configuration file contains the domains and their configuration,
and accounts and their configuration. This includes the configured email
addresses. The beacon admin web interface, and the beacon command line interface, can
make changes to this file. Mox automatically reloads this file when it changes.

Like the static configuration, the example domains.conf printed by this command
needs modifications to make it valid.

	usage: beacon config describe-domains >domains.conf

# beacon config describe-static

Prints an annotated empty configuration for use as beacon.conf.

The static configuration file cannot be reloaded while beacon is running. Mox has
to be restarted for changes to the static configuration file to take effect.

This configuration file needs modifications to make it valid. For example, it
may contain unfinished list items.

	usage: beacon config describe-static >beacon.conf

# beacon config account add

Add an account with an email address and reload the configuration.

Email can be delivered to this address/account. A password has to be configured
explicitly, see the setaccountpassword command.

	usage: beacon config account add account address

# beacon config account rm

Remove an account and reload the configuration.

Email addresses for this account will also be removed, and incoming email for
these addresses will be rejected.

	usage: beacon config account rm account

# beacon config address add

Adds an address to an account and reloads the configuration.

If address starts with a @ (i.e. a missing localpart), this is a catchall
address for the domain.

	usage: beacon config address add address account

# beacon config address rm

Remove an address and reload the configuration.

Incoming email for this address will be rejected after removing an address.

	usage: beacon config address rm address

# beacon config domain add

Adds a new domain to the configuration and reloads the configuration.

The account is used for the postmaster mailboxes the domain, including as DMARC and
TLS reporting. Localpart is the "username" at the domain for this account. If
must be set if and only if account does not yet exist.

	usage: beacon config domain add domain account [localpart]

# beacon config domain rm

Remove a domain from the configuration and reload the configuration.

This is a dangerous operation. Incoming email delivery for this domain will be
rejected.

	usage: beacon config domain rm domain

# beacon config describe-sendmail

Describe configuration for beacon when invoked as sendmail.

	usage: beacon config describe-sendmail >/etc/beaconsubmit.conf

# beacon config printservice

Prints a systemd unit service file for beacon.

This is the same file as generated using quickstart. If the systemd service file
has changed with a newer version of beacon, use this command to generate an up to
date version.

	usage: beacon config printservice >beacon.service

# beacon config ensureacmehostprivatekeys

Ensure host private keys exist for TLS listeners with ACME.

In beacon.conf, each listener can have TLS configured. Long-lived private key files
can be specified, which will be used when requesting ACME certificates.
Configuring these private keys makes it feasible to publish DANE TLSA records
for the corresponding public keys in DNS, protected with DNSSEC, allowing TLS
certificate verification without depending on a list of Certificate Authorities
(CAs). Previous versions of beacon did not pre-generate private keys for use with
ACME certificates, but would generate private keys on-demand. By explicitly
configuring private keys, they will not change automatedly with new
certificates, and the DNS TLSA records stay valid.

This command looks for listeners in beacon.conf with TLS with ACME configured. For
each missing host private key (of type rsa-2048 and ecdsa-p256) a key is written
to config/hostkeys/. If a certificate exists in the ACME "cache", its private
key is copied. Otherwise a new private key is generated. Snippets for manually
updating/editing beacon.conf are printed.

After running this command, and updating beacon.conf, run "beacon config dnsrecords"
for a domain and create the TLSA DNS records it suggests to enable DANE.

	usage: beacon config ensureacmehostprivatekeys

# beacon example

List available examples, or print a specific example.

	usage: beacon example [name]

# beacon checkupdate

Check if a newer version of beacon is available.

A single DNS TXT lookup to _updates.xbeacon.nl tells if a new version is
available. If so, a changelog is fetched from https://updates.xbeacon.nl, and the
individual entries verified with a builtin public key. The changelog is
printed.

	usage: beacon checkupdate

# beacon cid

Turn an ID from a Received header into a cid, for looking up in logs.

A cid is essentially a connection counter initialized when beacon starts. Each log
line contains a cid. Received headers added by beacon contain a unique ID that can
be decrypted to a cid by admin of a beacon instance only.

	usage: beacon cid cid

# beacon clientconfig

Print the configuration for email clients for a domain.

Sending email is typically not done on the SMTP port 25, but on submission
ports 465 (with TLS) and 587 (without initial TLS, but usually added to the
connection with STARTTLS). For IMAP, the port with TLS is 993 and without is
143.

Without TLS/STARTTLS, passwords are sent in clear text, which should only be
configured over otherwise secured connections, like a VPN.

	usage: beacon clientconfig domain

# beacon dane dial

Dial the address using TLS with certificate verification using DANE.

Data is copied between connection and stdin/stdout until either side closes the
connection.

	usage: beacon dane dial host:port
	  -usages string
	    	allowed usages for dane, comma-separated list (default "pkix-ta,pkix-ee,dane-ta,dane-ee")

# beacon dane dialmx

Connect to MX server for domain using STARTTLS verified with DANE.

If no destination host is specified, regular delivery logic is used to find the
hosts to attempt delivery too. This involves following CNAMEs for the domain,
looking up MX records, and possibly falling back to the domain name itself as
host.

If a destination host is specified, that is the only candidate host considered
for dialing.

With a list of destinations gathered, each is dialed until a successful SMTP
session verified with DANE has been initialized, including EHLO and STARTTLS
commands.

Once connected, data is copied between connection and stdin/stdout, until
either side closes the connection.

This command follows the same logic as delivery attempts made from the queue,
sharing most of its code.

	usage: beacon dane dialmx domain [destination-host]
	  -ehlohostname string
	    	hostname to send in smtp ehlo command (default "localhost")

# beacon dane makerecord

Print TLSA record for given certificate/key and parameters.

Valid values:
- usage: pkix-ta (0), pkix-ee (1), dane-ta (2), dane-ee (3)
- selector: cert (0), spki (1)
- matchtype: full (0), sha2-256 (1), sha2-512 (2)

Common DANE TLSA record parameters are: dane-ee spki sha2-256, or 3 1 1,
followed by a sha2-256 hash of the DER-encoded "SPKI" (subject public key info)
from the certificate. An example DNS zone file entry:

	_25._tcp.example.com. TLSA 3 1 1 133b919c9d65d8b1488157315327334ead8d83372db57465ecabf53ee5748aee

The first usable information from the pem file is used to compose the TLSA
record. In case of selector "cert", a certificate is required. Otherwise the
"subject public key info" (spki) of the first certificate or public or private
key (pkcs#8, pkcs#1 or ec private key) is used.

	usage: beacon dane makerecord usage selector matchtype [certificate.pem | publickey.pem | privatekey.pem]

# beacon dns lookup

Lookup DNS name of given type.

Lookup always prints whether the response was DNSSEC-protected.

Examples:

beacon dns lookup ptr 1.1.1.1
beacon dns lookup mx xbeacon.nl
beacon dns lookup txt _dmarc.xbeacon.nl.
beacon dns lookup tlsa _25._tcp.xbeacon.nl

	usage: beacon dns lookup [ptr | mx | cname | ips | a | aaaa | ns | txt | srv | tlsa] name

# beacon dkim gened25519

Generate a new ed25519 key for use with DKIM.

Ed25519 keys are much smaller than RSA keys of comparable cryptographic
strength. This is convenient because of maximum DNS message sizes. At the time
of writing, not many mail servers appear to support ed25519 DKIM keys though,
so it is recommended to sign messages with both RSA and ed25519 keys.

	usage: beacon dkim gened25519 >$selector._domainkey.$domain.ed25519.privatekey.pkcs8.pem

# beacon dkim genrsa

Generate a new 2048 bit RSA private key for use with DKIM.

The generated file is in PEM format, and has a comment it is generated for use
with DKIM, by beacon.

	usage: beacon dkim genrsa >$selector._domainkey.$domain.rsa2048.privatekey.pkcs8.pem

# beacon dkim lookup

Lookup and print the DKIM record for the selector at the domain.

	usage: beacon dkim lookup selector domain

# beacon dkim txt

Print a DKIM DNS TXT record with the public key derived from the private key read from stdin.

The DNS should be configured as a TXT record at $selector._domainkey.$domain.

	usage: beacon dkim txt <$selector._domainkey.$domain.key.pkcs8.pem

# beacon dkim verify

Verify the DKIM signatures in a message and print the results.

The message is parsed, and the DKIM-Signature headers are validated. Validation
of older messages may fail because the DNS records have been removed or changed
by now, or because the signature header may have specified an expiration time
that was passed.

	usage: beacon dkim verify message

# beacon dkim sign

Sign a message, adding DKIM-Signature headers based on the domain in the From header.

The message is parsed, the domain looked up in the configuration files, and
DKIM-Signature headers generated. The message is printed with the DKIM-Signature
headers prepended.

	usage: beacon dkim sign message

# beacon dmarc lookup

Lookup dmarc policy for domain, a DNS TXT record at _dmarc.<domain>, validate and print it.

	usage: beacon dmarc lookup domain

# beacon dmarc parsereportmsg

Parse a DMARC report from an email message, and print its extracted details.

DMARC reports are periodically mailed, if requested in the DMARC DNS record of
a domain. Reports are sent by mail servers that received messages with our
domain in a From header. This may or may not be legatimate email. DMARC reports
contain summaries of evaluations of DMARC and DKIM/SPF, which can help
understand email deliverability problems.

	usage: beacon dmarc parsereportmsg message ...

# beacon dmarc verify

Parse an email message and evaluate it against the DMARC policy of the domain in the From-header.

mailfromaddress and helodomain are used for SPF validation. If both are empty,
SPF validation is skipped.

mailfromaddress should be the address used as MAIL FROM in the SMTP session.
For DSN messages, that address may be empty. The helo domain was specified at
the beginning of the SMTP transaction that delivered the message. These values
can be found in message headers.

	usage: beacon dmarc verify remoteip mailfromaddress helodomain < message

# beacon dmarc checkreportaddrs

For each reporting address in the domain's DMARC record, check if it has opted into receiving reports (if needed).

A DMARC record can request reports about DMARC evaluations to be sent to an
email/http address. If the organizational domains of that of the DMARC record
and that of the report destination address do not match, the destination
address must opt-in to receiving DMARC reports by creating a DMARC record at
<dmarcdomain>._report._dmarc.<reportdestdomain>.

	usage: beacon dmarc checkreportaddrs domain

# beacon dnsbl check

Test if IP is in the DNS blocklist of the zone, e.g. bl.spamcop.net.

If the IP is in the blocklist, an explanation is printed. This is typically a
URL with more information.

	usage: beacon dnsbl check zone ip

# beacon dnsbl checkhealth

Check the health of the DNS blocklist represented by zone, e.g. bl.spamcop.net.

The health of a DNS blocklist can be checked by querying for 127.0.0.1 and
127.0.0.2. The second must and the first must not be present.

	usage: beacon dnsbl checkhealth zone

# beacon mtasts lookup

Lookup the MTASTS record and policy for the domain.

MTA-STS is a mechanism for a domain to specify if it requires TLS connections
for delivering email. If a domain has a valid MTA-STS DNS TXT record at
_mta-sts.<domain> it signals it implements MTA-STS. A policy can then be
fetched at https://mta-sts.<domain>/.well-known/mta-sts.txt. The policy
specifies the mode (enforce, testing, none), which MX servers support TLS and
should be used, and how long the policy can be cached.

	usage: beacon mtasts lookup domain

# beacon retrain

Recreate and retrain the junk filter for the account.

Useful after having made changes to the junk filter configuration, or if the
implementation has changed.

	usage: beacon retrain accountname

# beacon sendmail

Sendmail is a drop-in replacement for /usr/sbin/sendmail to deliver emails sent by unix processes like cron.

If invoked as "sendmail", it will act as sendmail for sending messages. Its
intention is to let processes like cron send emails. Messages are submitted to
an actual mail server over SMTP. The destination mail server and credentials are
configured in /etc/beaconsubmit.conf, see beacon config describe-sendmail. The From
message header is rewritten to the configured address. When the addressee
appears to be a local user, because without @, the message is sent to the
configured default address.

If submitting an email fails, it is added to a directory beaconsubmit.failures in
the user's home directory.

Most flags are ignored to fake compatibility with other sendmail
implementations. A single recipient or the -t flag with a To-header is required.
With the -t flag, Cc and Bcc headers are not handled specially, so Bcc is not
removed and the addresses do not receive the email.

/etc/beaconsubmit.conf should be group-readable and not readable by others and this
binary should be setgid that group:

	groupadd beaconsubmit
	install -m 2755 -o root -g beaconsubmit beacon /usr/sbin/sendmail
	touch /etc/beaconsubmit.conf
	chown root:beaconsubmit /etc/beaconsubmit.conf
	chmod 640 /etc/beaconsubmit.conf
	# edit /etc/beaconsubmit.conf


	usage: beacon sendmail [-Fname] [ignoredflags] [-t] [<message]

# beacon spf check

Check the status of IP for the policy published in DNS for the domain.

IPs may be allowed to send for a domain, or disallowed, and several shades in
between. If not allowed, an explanation may be provided by the policy. If so,
the explanation is printed. The SPF mechanism that matched (if any) is also
printed.

	usage: beacon spf check domain ip

# beacon spf lookup

Lookup the SPF record for the domain and print it.

	usage: beacon spf lookup domain

# beacon spf parse

Parse the record as SPF record. If valid, nothing is printed.

	usage: beacon spf parse txtrecord

# beacon tlsrpt lookup

Lookup the TLSRPT record for the domain.

A TLSRPT record typically contains an email address where reports about TLS
connectivity should be sent. Mail servers attempting delivery to our domain
should attempt to use TLS. TLSRPT lets them report how many connection
successfully used TLS, and how what kind of errors occurred otherwise.

	usage: beacon tlsrpt lookup domain

# beacon tlsrpt parsereportmsg

Parse and print the TLSRPT in the message.

The report is printed in formatted JSON.

	usage: beacon tlsrpt parsereportmsg message ...

# beacon version

Prints this beacon version.

	usage: beacon version

# beacon bumpuidvalidity

Change the IMAP UID validity of the mailbox, causing IMAP clients to refetch messages.

This can be useful after manually repairing metadata about the account/mailbox.

Opens account database file directly. Ensure beacon does not have the account
open, or is not running.

	usage: beacon bumpuidvalidity account [mailbox]

# beacon reassignuids

Reassign UIDs in one mailbox or all mailboxes in an account and bump UID validity, causing IMAP clients to refetch messages.

Opens account database file directly. Ensure beacon does not have the account
open, or is not running.

	usage: beacon reassignuids account [mailboxid]

# beacon fixuidmeta

Fix inconsistent UIDVALIDITY and UIDNEXT in messages/mailboxes/account.

The next UID to use for a message in a mailbox should always be higher than any
existing message UID in the mailbox. If it is not, the mailbox UIDNEXT is
updated.

Each mailbox has a UIDVALIDITY sequence number, which should always be lower
than the per-account next UIDVALIDITY to use. If it is not, the account next
UIDVALIDITY is updated.

Opens account database file directly. Ensure beacon does not have the account
open, or is not running.

	usage: beacon fixuidmeta account

# beacon fixmsgsize

Ensure message sizes in the database matching the sum of the message prefix length and on-disk file size.

Messages with an inconsistent size are also parsed again.

If an inconsistency is found, you should probably also run "beacon
bumpuidvalidity" on the mailboxes or entire account to force IMAP clients to
refetch messages.

	usage: beacon fixmsgsize [account]

# beacon reparse

# Parse all messages in the account or all accounts again

Can be useful after upgrading beacon with improved message parsing. Messages are
parsed in batches, so other access to the mailboxes/messages are not blocked
while reparsing all messages.

	usage: beacon reparse [account]

# beacon ensureparsed

Ensure messages in the database have a pre-parsed MIME form in the database.

	usage: beacon ensureparsed account
	  -all
	    	store new parsed message for all messages

# beacon recalculatemailboxcounts

Recalculate message counts for all mailboxes in the account, and total message size for quota.

When a message is added to/removed from a mailbox, or when message flags change,
the total, unread, unseen and deleted messages are accounted, the total size of
the mailbox, and the total message size for the account. In case of a bug in
this accounting, the numbers could become incorrect. This command will find, fix
and print them.

	usage: beacon recalculatemailboxcounts account

# beacon message parse

Parse message, print JSON representation.

	usage: beacon message parse message.eml

# beacon reassignthreads

Reassign message threads.

For all accounts, or optionally only the specified account.

Threading for all messages in an account is first reset, and new base subject
and normalized message-id saved with the message. Then all messages are
evaluated and matched against their parents/ancestors.

Messages are matched based on the References header, with a fall-back to an
In-Reply-To header, and if neither is present/valid, based only on base
subject.

A References header typically points to multiple previous messages in a
hierarchy. From oldest ancestor to most recent parent. An In-Reply-To header
would have only a message-id of the parent message.

A message is only linked to a parent/ancestor if their base subject is the
same. This ensures unrelated replies, with a new subject, are placed in their
own thread.

The base subject is lower cased, has whitespace collapsed to a single
space, and some components removed: leading "Re:", "Fwd:", "Fw:", or bracketed
tag (that mailing lists often add, e.g. "[listname]"), trailing "(fwd)", or
enclosing "[fwd: ...]".

Messages are linked to all their ancestors. If an intermediate parent/ancestor
message is deleted in the future, the message can still be linked to the earlier
ancestors. If the direct parent already wasn't available while matching, this is
stored as the message having a "missing link" to its stored ancestors.

	usage: beacon reassignthreads [account]
*/
package main

// NOTE: DO NOT EDIT, this file is generated by gendoc.sh.
