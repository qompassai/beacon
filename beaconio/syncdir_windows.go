package beaconio

import (
	"github.com/qompassai/beacon/mlog"
)

// SyncDir opens a directory and syncs its contents to disk.
// SyncDir is a no-op on Windows.
func SyncDir(log mlog.Log, dir string) error {
	// todo: how to sync a directory on windows?
	return nil
}
