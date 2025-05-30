package imapserver

import (
	"testing"

	"github.com/qompassai/beacon/imapclient"
)

func TestUnselect(t *testing.T) {
	tc := start(t)
	defer tc.close()

	tc.client.Login("mjl@beacon.example", "testtest")
	tc.client.Select("inbox")

	tc.transactf("bad", "unselect bogus") // Leftover data.
	tc.transactf("ok", "unselect")
	tc.transactf("no", "fetch 1 all") // Invalid when not selected.

	tc.client.Select("inbox")
	tc.client.Append("inbox", nil, nil, []byte(exampleMsg))
	tc.client.StoreFlagsAdd("1", true, `\Deleted`)
	tc.transactf("ok", "unselect")
	tc.transactf("ok", "status inbox (messages)")
	tc.xuntagged(imapclient.UntaggedStatus{Mailbox: "Inbox", Attrs: map[string]int64{"MESSAGES": 1}}) // Message not removed.
}
