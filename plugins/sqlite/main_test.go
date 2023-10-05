package main

import (
	"database/sql/driver"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/linyows/warp"
	_ "github.com/mattn/go-sqlite3"
)

func TestConst(t *testing.T) {
	var expect string
	var got string

	expect = "sqlite-plugin"
	got = prefix
	if got != expect {
		t.Errorf("expected %s, got %s", expect, got)
	}

	expect = "insert into communications (id, connection_id, occurred_at, direction, data) values ($1, $2, $3, $4, $5)"
	got = commQuery
	if got != expect {
		t.Errorf("expected %s, got %s", expect, got)
	}

	expect = "insert into connections (id, occurred_at, mail_from, mail_to, elapse) values ($1, $2, $3, $4, $5)"
	got = connQuery
	if got != expect {
		t.Errorf("expected %s, got %s", expect, got)
	}
}

func TestWriter(t *testing.T) {
	expectError := "missing dsn for sqlite, please set `DSN`"
	sqlite := Sqlite{}
	_, err := sqlite.Conn()

	if err != nil && fmt.Sprintf("%s", err) != expectError {
		t.Errorf("expected %s, got %s", expectError, err)
	}
}

type AnyID struct{}

func (a AnyID) Match(v driver.Value) bool {
	_, ok := v.(string)
	return ok
}

func TestAfterComm(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ti := time.Date(2023, time.August, 16, 14, 48, 0, 0, time.UTC)

	mock.ExpectExec("insert into communications").WithArgs(
		AnyID{},
		"abcdefg",
		ti.Format(warp.TimeFormat),
		"--",
		[]byte("hello"),
	).WillReturnResult(sqlmock.NewResult(1, 1))

	data := &warp.AfterCommData{
		ConnID:     "abcdefg",
		OccurredAt: ti,
		Data:       []byte("hello"),
		Direction:  "--",
	}

	sqlite := Sqlite{pool: db}
	sqlite.AfterComm(data)
}

func TestAfterConn(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ti := time.Date(2023, time.August, 16, 14, 48, 0, 0, time.UTC)

	mock.ExpectExec("insert into connections").WithArgs(
		"abcdefg",
		ti.Format(warp.TimeFormat),
		[]byte("alice@example.local"),
		[]byte("bob@example.test"),
		20,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	data := &warp.AfterConnData{
		ConnID:     "abcdefg",
		OccurredAt: ti,
		MailFrom:   []byte("alice@example.local"),
		MailTo:     []byte("bob@example.test"),
		Elapse:     20,
	}

	sqlite := Sqlite{pool: db}
	sqlite.AfterConn(data)
}

func TestIntegration(t *testing.T) {
	err := os.Setenv("DSN", "../../testdata/warp.sqlite")
	if err != nil {
		t.Fatalf("Setenv error: '%s'", err)
	}

	sqlite := &Sqlite{}
	sqlite.AfterInit()

	id := warp.GenID().String()

	sqlite.AfterComm(&warp.AfterCommData{
		ConnID:     id,
		OccurredAt: time.Now(),
		Data:       []byte("hello"),
		Direction:  "->",
	})

	sqlite.AfterConn(&warp.AfterConnData{
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
