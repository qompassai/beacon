package dns

import (
	"errors"
	"testing"
)

func TestParseDomain(t *testing.T) {
	test := func(lax bool, s string, exp Domain, expErr error) {
		t.Helper()
		var dom Domain
		var err error
		if lax {
			dom, err = ParseDomainLax(s)
		} else {
			dom, err = ParseDomain(s)
		}
		if (err == nil) != (expErr == nil) || expErr != nil && !errors.Is(err, expErr) {
			t.Fatalf("parse domain %q: err %v, expected %v", s, err, expErr)
		}
		if expErr == nil && dom != exp {
			t.Fatalf("parse domain %q: got %#v, epxected %#v", s, dom, exp)
		}
	}

	// We rely on normalization of names throughout the code base.
	test(false, "xbeacon.nl", Domain{"xbeacon.nl", ""}, nil)
	test(false, "XBEACON.NL", Domain{"xbeacon.nl", ""}, nil)
	test(false, "TEST☺.XBEACON.NL", Domain{"xn--test-3o3b.xbeacon.nl", "test☺.xbeacon.nl"}, nil)
	test(false, "TEST☺.XBEACON.NL", Domain{"xn--test-3o3b.xbeacon.nl", "test☺.xbeacon.nl"}, nil)
	test(false, "ℂᵤⓇℒ。𝐒🄴", Domain{"curl.se", ""}, nil) // https://daniel.haxx.se/blog/2022/12/14/idn-is-crazy/
	test(false, "xbeacon.nl.", Domain{}, errTrailingDot)

	test(false, "_underscore.xbeacon.nl", Domain{}, errIDNA)
	test(true, "_underscore.xbeacon.NL", Domain{ASCII: "_underscore.xbeacon.nl"}, nil)
	test(true, "_underscore.☺.xbeacon.nl", Domain{}, errUnderscore)
	test(true, "_underscore.xn--test-3o3b.xbeacon.nl", Domain{}, errUnderscore)
}
