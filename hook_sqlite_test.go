package warp

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/glebarez/go-sqlite"
)

func TestHookSqliteConst(t *testing.T) {
	var expect string
	var got string

	expect = "insert into communications (id, connection_id, occurred_at, direction, data) values ($1, $2, $3, $4, $5)"
	got = sqliteCommQuery
	if got != expect {
		t.Errorf("expected %s, got %s", expect, got)
	}

	expect = "insert into connections (id, occurred_at, mail_from, mail_to, elapse) values ($1, $2, $3, $4, $5)"
	got = sqliteConnQuery
	if got != expect {
		t.Errorf("expected %s, got %s", expect, got)
	}
}

func TestHookSqliteName(t *testing.T) {
	sqlite := &HookSqlite{}
	expect := "sqlite"
	got := sqlite.Name()
	if got != expect {
		t.Errorf("expected %s, got %s", expect, got)
	}
}

func TestHookSqliteConn(t *testing.T) {
	expectError := "missing dsn for sqlite, please set `DSN`"
	sqlite := &HookSqlite{}
	_, err := sqlite.conn()

	if err != nil && fmt.Sprintf("%s", err) != expectError {
		t.Errorf("expected %s, got %s", expectError, err)
	}
}

func TestHookSqliteAfterComm(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ti := time.Date(2023, time.August, 16, 14, 48, 0, 0, time.UTC)

	mock.ExpectExec("insert into communications").WithArgs(
		AnyID{},
		"abcdefg",
		ti.Format(TimeFormat),
		"--",
		[]byte("hello"),
	).WillReturnResult(sqlmock.NewResult(1, 1))

	data := &AfterCommData{
		ConnID:     "abcdefg",
		OccurredAt: ti,
		Data:       []byte("hello"),
		Direction:  "--",
	}

	sqlite := &HookSqlite{pool: db}
	sqlite.AfterComm(data)
}

func TestHookSqliteAfterConn(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ti := time.Date(2023, time.August, 16, 14, 48, 0, 0, time.UTC)

	mock.ExpectExec("insert into connections").WithArgs(
		"abcdefg",
		ti.Format(TimeFormat),
		[]byte("alice@example.local"),
		[]byte("bob@example.test"),
		20,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	data := &AfterConnData{
		ConnID:     "abcdefg",
		OccurredAt: ti,
		MailFrom:   []byte("alice@example.local"),
		MailTo:     []byte("bob@example.test"),
		Elapse:     20,
	}

	sqlite := &HookSqlite{pool: db}
	sqlite.AfterConn(data)
}

func TestHookSqliteIntegration(t *testing.T) {
	err := os.Setenv("DSN", "./testdata/warp.sqlite")
	if err != nil {
		t.Fatalf("Setenv error: '%s'", err)
	}

	sqlite := &HookSqlite{}
	sqlite.AfterInit()

	id := GenID().String()

	sqlite.AfterComm(&AfterCommData{
		ConnID:     id,
		OccurredAt: time.Now(),
		Data:       []byte("hello"),
		Direction:  "->",
	})

	sqlite.AfterConn(&AfterConnData{
		ConnID:     id,
		OccurredAt: time.Now(),
		MailFrom:   []byte("alice@example.local"),
		MailTo:     []byte("bob@example.test"),
		Elapse:     1234,
	})

	row := sqlite.pool.QueryRow(`select data from communications where connection_id = $1`, id)
	if row == nil {
		t.Fatalf("sqlite QueryRow error: '%s'", err)
	}
	var res string
	err = row.Scan(&res)
	if err != nil {
		t.Error("Failed to db.Scan:", err)
	}
}
