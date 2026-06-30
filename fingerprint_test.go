package sql_test

import (
	"errors"
	"testing"

	sql "github.com/gomatic/go-sql"
)

func TestFingerprintIgnoresFormattingCaseAndLiterals(t *testing.T) {
	a, err := sql.Fingerprint("SELECT a,b FROM t WHERE x = 1")
	if err != nil {
		t.Fatal(err)
	}
	b, err := sql.Fingerprint("select a\n, b\nfrom t\nwhere x = 2")
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("fingerprints differ: %q vs %q", a, b)
	}
}

func TestFingerprintReportsParseFailure(t *testing.T) {
	_, err := sql.Fingerprint("select from from")
	if !errors.Is(err, sql.ErrFingerprint) {
		t.Fatalf("got %v, want ErrFingerprint", err)
	}
}
