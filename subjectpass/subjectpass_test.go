package subjectpass

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/qompassai/beacon/mlog"
	"github.com/qompassai/beacon/smtp"
)

func TestSubjectPass(t *testing.T) {
	log := mlog.New("subjectpass", nil)

	key := []byte("secret token")
	addr, _ := smtp.ParseAddress("beacon@beacon.example")
	sig := Generate(log.Logger, addr, key, time.Now())

	message := fmt.Sprintf("From: <beacon@beacon.example>\r\nSubject: let me in %s\r\n\r\nthe message", sig)
	if err := Verify(log.Logger, strings.NewReader(message), key, time.Hour); err != nil {
		t.Fatalf("verifyPassToken: %s", err)
	}

	if err := Verify(log.Logger, strings.NewReader(message), []byte("bad key"), time.Hour); err == nil {
		t.Fatalf("verifyPassToken did not fail")
	}

	sig = Generate(log.Logger, addr, key, time.Now().Add(-time.Hour-257))
	message = fmt.Sprintf("From: <beacon@beacon.example>\r\nSubject: let me in %s\r\n\r\nthe message", sig)
	if err := Verify(log.Logger, strings.NewReader(message), key, time.Hour); !errors.Is(err, ErrExpired) {
		t.Fatalf("verifyPassToken should have expired")
	}
}
