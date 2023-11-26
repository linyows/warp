package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/glebarez/go-sqlite"
	"github.com/linyows/warp"
)

const (
	sqliteCommQuery       string = "insert into communications (id, connection_id, occurred_at, direction, data) values ($1, $2, $3, $4, $5)"
	sqliteConnQuery       string = "insert into connections (id, occurred_at, mail_from, mail_to, elapse) values ($1, $2, $3, $4, $5)"
	sqliteConnCreateTable string = `
	create table if not exists connections (
    id text primary key,
    mail_from text,
    mail_to text,
    occurred_at datetime default CURRENT_TIMESTAMP,
    elapse integer
	)`
	sqliteCommCreateTable string = `
	create table if not exists communications (
    id text primary key,
    connection_id text,
    direction text,
    data text,
    occurred_at datetime default CURRENT_TIMESTAMP
	)`
)

type Sqlite struct {
	pool *sql.DB // Database connection pool.
}

func (s *Sqlite) Name() string {
	return "sqlite"
}

func (s *Sqlite) conn() (*sql.DB, error) {
	if s.pool != nil {
		return s.pool, nil
	}

	dsn := os.Getenv("DSN")
	if len(dsn) == 0 {
		return nil, fmt.Errorf("missing dsn for sqlite, please set `DSN`")
	}

	var err error
	s.pool, err = sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open error: %s(%#v)\n", err.Error(), err)
	}

	return s.pool, nil
}

func (s *Sqlite) AfterInit() {
	conn, err := s.conn()
	if err != nil {
		fmt.Printf("[%s] %s\n", s.Name(), err)
		return
	}

	_, err = conn.Exec(sqliteConnCreateTable)
	if err != nil {
		fmt.Printf("[%s] db exec error: %s\n", s.Name(), err)
	}

	_, err = conn.Exec(sqliteCommCreateTable)
	if err != nil {
		fmt.Printf("[%s] db exec error: %s\n", s.Name(), err)
	}
}

func (s *Sqlite) AfterComm(d *warp.AfterCommData) {
	conn, err := s.conn()
	if err != nil {
		fmt.Printf("[%s] %s\n", s.Name(), err)
		return
	}

	_, err = conn.Exec(
		sqliteCommQuery,
		warp.GenID().String(),
		d.ConnID,
		d.OccurredAt.Format(warp.TimeFormat),
		d.Direction,
		d.Data,
	)
	if err != nil {
		fmt.Printf("[%s] db exec error: %s\n", s.Name(), err)
	}
}

func (s *Sqlite) AfterConn(d *warp.AfterConnData) {
	conn, err := s.conn()
	if err != nil {
		fmt.Printf("[%s] %s\n", s.Name(), err)
		return
	}

	_, err = conn.Exec(
		sqliteConnQuery,
		d.ConnID,
		d.OccurredAt.Format(warp.TimeFormat),
		d.MailFrom,
		d.MailTo,
		d.Elapse,
	)
	if err != nil {
		fmt.Printf("[%s] db exec error: %s\n", s.Name(), err)
	}
}

var Hook Sqlite //nolint
