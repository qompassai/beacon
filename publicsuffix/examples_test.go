package publicsuffix_test

import (
	"context"
	"fmt"

	"golang.org/x/exp/slog"

	"github.com/qompassai/beacon/dns"
	"github.com/qompassai/beacon/publicsuffix"
)

func ExampleLookup() {
	// Lookup the organizational domain for sub.example.org.
	orgDom := publicsuffix.Lookup(context.Background(), slog.Default(), dns.Domain{ASCII: "sub.example.org"})
	fmt.Println(orgDom)
	// Output: example.org
}
