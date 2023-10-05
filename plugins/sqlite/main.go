package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/linyows/warp"
	_ "github.com/mattn/go-sqlite3"
)

const (
	prefix          string = "sqlite-plugin"
	commQuery       string = "insert into communications (id, connection_id, occurred_at, direction, data) values ($1, $2, $3, $4, $5)"
	connQuery       string = "insert into connections (id, occurred_at, mail_from, mail_to, elapse) values ($1, $2, $3, $4, $5)"
	connCreateTable string = `create table if not exists connections (
    id text primary key,
    mail_from text,
    mail_to text,
    occurred_at datetime default CURRENT_TIMESTAMP,
    elapse integer);`
	commCreateTable string = `create table if not exists communications (
    id text primary key,
    connection_id text,
    direction text,
    data text,
    occurred_at datetime default CURRENT_TIMESTAMP)`
)

type Sqlite struct {
	pool *sql.DB // Database connection pool.
}

func (s *Sqlite) Conn() (*sql.DB, error) {
	if s.pool != nil {
		return s.pool, nil
	}

	dsn := os.Getenv("DSN")
	if len(dsn) == 0 {
		return nil, fmt.Errorf("missing dsn for sqlite, please set `DSN`")
	}

	var err error
	s.pool, err = sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open error: %s(%#v)\n", err.Error(), err)
	}

	return s.pool, nil
}

func (s *Sqlite) AfterInit() {
	conn, err := s.Conn()
	if err != nil {
		fmt.Printf("[%s] %s(%#v)\n", prefix, err.Error(), err)
		return
	}

	_, err = conn.Exec(connCreateTable)
	if err != nil {
		fmt.Printf("[%s] db exec error: %s(%#v)\n", prefix, err.Error(), err)
	}

	_, err = conn.Exec(commCreateTable)
	if err != nil {
		fmt.Printf("[%s] db exec error: %s(%#v)\n", prefix, err.Error(), err)
	}
}

func (s *Sqlite) AfterComm(d *warp.AfterCommData) {
	conn, err := s.Conn()
	if err != nil {
		fmt.Printf("[%s] %s(%#v)\n", prefix, err.Error(), err)
		return
	}

	_, err = conn.Exec(
		commQuery,
		warp.GenID().String(),
		d.ConnID,
		d.OccurredAt.Format(warp.TimeFormat),
		d.Direction,
		d.Data,
	)
	if err != nil {
		fmt.Printf("[%s] db exec error: %s(%#v)\n", prefix, err.Error(), err)
	}
}

func (s *Sqlite) AfterConn(d *warp.AfterConnData) {
	conn, err := s.Conn()
	if err != nil {
		fmt.Printf("[%s] %s(%#v)\n", prefix, err.Error(), err)
		return
	}

	_, err = conn.Exec(
		connQuery,
		d.ConnID,
		d.OccurredAt.Format(warp.TimeFormat),
		d.MailFrom,
		d.MailTo,
		d.Elapse,
	)
	if err != nil {
		fmt.Printf("[%s] db exec error: %s(%#v)\n", prefix, err.Error(), err)
	}
}

var Hook Sqlite //nolint
