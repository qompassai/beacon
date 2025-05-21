<!-- /qompassai/beacon/README.md -->
<!-- --------------------------------- -->
<!-- Copyright (C) 2025 Qompass AI, All rights reserved -->

<h2> Qompass AI Beacon: Self-hosted mail made simple </h2>

![Repository Views](https://komarev.com/ghpvc/?username=qompassai-beacon)
![GitHub all releases](https://img.shields.io/github/downloads/qompassai/beacon/total?style=flat-square)

<br>
  <a href="https://www.gnu.org/licenses/agpl-3.0"><img src="https://img.shields.io/badge/License-AGPL%20v3-blue.svg" alt="License: AGPL v3"></a>
  <a href="./LICENSE-QCDA"><img src="https://img.shields.io/badge/license-Q--CDA-lightgrey.svg" alt="License: Q-CDA"></a>
</p>

## Features

- Quick and easy to start/maintain mail server, for your own domain(s).

- SMTP (with extensions) for receiving, submitting and delivering email.

- IMAP4 (with extensions) for giving email clients access to email.

- Webmail for reading/sending email from the browser.

- SPF/DKIM/DMARC for authenticating messages/delivery, also DMARC aggregate
  reports.

- Reputation tracking, learning (per user) host-, domain- and
  sender address-based reputation from (Non-)Junk email classification.

- Bayesian spam filtering that learns (per user) from (Non-)Junk email.

- Slowing down senders with no/low reputation or questionable email content
  (similar to greylisting). Rejected emails are stored in a mailbox called Rejects
  for a short period, helping with misclassified legitimate synchronous
  signup/login/transactional emails.

- Internationalized email, with unicode in email address usernames
  ("localparts"), and in domain names (IDNA).
- Automatic TLS with ACME, for use with Let's Encrypt and other CA's.

- DANE and MTA-STS for inbound and outbound delivery over SMTP with STARTTLS,
  including REQUIRETLS and with incoming/outgoing TLSRPT reporting.
- Web admin interface that helps you set up your domains and accounts
  (instructions to create DNS records, configure
  SPF/DKIM/DMARC/TLSRPT/MTA-STS), for status information, managing
  accounts/domains, and modifying the configuration file.
- Account autodiscovery (with SRV records, Microsoft-style, Thunderbird-style,
  and Apple device management profiles) for easy account setup (though client
  support is limited).
- Webserver with serving static files and forwarding requests (reverse
  proxy), so port 443 can also be used to serve websites.
- Prometheus metrics and structured logging for operational insight.
- "beacon" localserve" subcommand for running beacon locally for email-related
  testing/developing, including pedantic mode.
- Most non-server Go packages beacon consists of are written to be reusable.

