package beacon

import (
	"context"
	"time"
)

// Sleep for d, but return as soon as ctx is done.
//
// Used for a few places where sleep is used to push back on clients, but where
// shutting down should abort the sleep.
func Sleep(ctx context.Context, d time.Duration) {
	t := time.NewTicker(d)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
	}
}
