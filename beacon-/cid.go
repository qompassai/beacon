package beacon

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/qompassai/beacon/mlog"
)

var cid atomic.Int64

func init() {
	cid.Store(time.Now().UnixMilli())
}

// Cid returns a new unique id to be used for connections/sessions/requests.
func Cid() int64 {
	return cid.Add(1)
}

// CidFromCtx returns the cid in the context, or 0.
func CidFromCtx(ctx context.Context) int64 {
	v := ctx.Value(mlog.CidKey)
	if v == nil {
		return 0
	}
	return v.(int64)
}
