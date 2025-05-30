package smtpserver

import (
	"context"

	"github.com/qompassai/beacon/dkim"
	"github.com/qompassai/beacon/dns"
	"github.com/qompassai/beacon/mlog"
	"github.com/qompassai/beacon/publicsuffix"
	"github.com/qompassai/beacon/spf"
	"github.com/qompassai/beacon/store"
)

// Alignment compares the msgFromDomain with the dkim and spf results, and returns
// a validation, one of: Strict, Relaxed, None.
func alignment(ctx context.Context, log mlog.Log, msgFromDomain dns.Domain, dkimResults []dkim.Result, spfStatus spf.Status, spfIdentity *dns.Domain) store.Validation {
	var strict, relaxed bool
	msgFromOrgDomain := publicsuffix.Lookup(ctx, log.Logger, msgFromDomain)

	// todo: should take temperror and permerror into account.
	for _, dr := range dkimResults {
		if dr.Status != dkim.StatusPass || dr.Sig == nil {
			continue
		}
		if dr.Sig.Domain == msgFromDomain {
			strict = true
			break
		} else {
			relaxed = relaxed || msgFromOrgDomain == publicsuffix.Lookup(ctx, log.Logger, dr.Sig.Domain)
		}
	}
	if !strict && spfStatus == spf.StatusPass {
		strict = msgFromDomain == *spfIdentity
		relaxed = relaxed || msgFromOrgDomain == publicsuffix.Lookup(ctx, log.Logger, *spfIdentity)
	}
	if strict {
		return store.ValidationStrict
	}
	if relaxed {
		return store.ValidationRelaxed
	}
	return store.ValidationNone
}
