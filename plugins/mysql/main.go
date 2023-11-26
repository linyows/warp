package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/linyows/warp"
)

const (
	mysqlCommQuery string = "insert into communications (id, connection_id, occurred_at, direction, data) values (?, ?, ?, ?, ?)"
	mysqlConnQuery string = "insert into connections (id, occurred_at, mail_from, mail_to, elapse) values (?, ?, ?, ?, ?)"
)

type Mysql struct {
	pool *sql.DB // Database connection pool.
}

func (m *Mysql) Name() string {
	return "mysql"
}

func (m *Mysql) conn() (*sql.DB, error) {
	if m.pool != nil {
		return m.pool, nil
	}

	dsn := os.Getenv("DSN")
	if len(dsn) == 0 {
		return nil, fmt.Errorf("missing dsn for mysql, please set `DSN`")
	}

	var err error
	m.pool, err = sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open error: %s\n", err)
	}

	return m.pool, nil
}

func (m *Mysql) AfterInit() {
}

func (m *Mysql) AfterComm(d *warp.AfterCommData) {
	conn, err := m.conn()
	if err != nil {
		fmt.Printf("[%s] %s\n", m.Name(), err)
		return
	}

	_, err = conn.Exec(
		mysqlCommQuery,
		warp.GenID().String(),
		d.ConnID,
		d.OccurredAt.Format(warp.TimeFormat),
		d.Direction,
		d.Data,
	)
	if err != nil {
		fmt.Printf("[%s] db exec error: %s\n", m.Name(), err)
	}
}

func (m *Mysql) AfterConn(d *warp.AfterConnData) {
	conn, err := m.conn()
	if err != nil {
		fmt.Printf("[%s] %s\n", m.Name(), err)
		return
	}

	_, err = conn.Exec(
		mysqlConnQuery,
		d.ConnID,
		d.OccurredAt.Format(warp.TimeFormat),
		d.MailFrom,
		d.MailTo,
		d.Elapse,
	)
	if err != nil {
		fmt.Printf("[%s] db exec error: %s\n", m.Name(), err)
	}
}

var Hook Mysql //nolint
