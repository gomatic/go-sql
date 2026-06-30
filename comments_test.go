package sql_test

import (
	"reflect"
	"testing"

	sql "github.com/gomatic/go-sql"
)

func TestCommentsExtractsBothKindsInOrder(t *testing.T) {
	got, err := sql.Comments("select 1 -- line\n, 2 /* block */ from t")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-- line", "/* block */"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestCommentsEmptyWhenNone(t *testing.T) {
	got, err := sql.Comments("select 1 from t")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %#v, want empty", got)
	}
}

func TestCommentsReportsScanFailure(t *testing.T) {
	_, err := sql.Comments("'unterminated")
	if err == nil {
		t.Fatal("want scan error, got nil")
	}
}
