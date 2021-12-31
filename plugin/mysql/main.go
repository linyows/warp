package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/linyows/warp"
)

const (
	commQuery string = `insert into communications(id, connection_id, occurred_at, direction, data) values(?, ?, ?, ?, ?)`
	connQuery string = `insert into connections(id, occurred_at, mail_from, mail_to) values(?, ?, ?, ?)`
)

type Mysql struct {
	pool *sql.DB // Database connection pool.
}

func (m *Mysql) Conn() *sql.DB {
	if m.pool != nil {
		return m.pool
	}

	dsn := os.Getenv("DSN")
	if len(dsn) == 0 {
		panic("missing dsn for mysql")
	}

	var err error
	m.pool, err = sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("db open error: %#v\n", err)
	}

	return m.pool
}

func (m *Mysql) AfterComm(d *warp.AfterCommData) {
	_, err := m.Conn().Exec(
		commQuery,
		warp.GenID().String(),
		d.ConnID,
		d.OccurredAt.Format(warp.TimeFormat),
		d.Direction,
		d.Data,
	)
	if err != nil {
		fmt.Printf("db exec error: %#v\n", err)
	}
}

func (m *Mysql) AfterConn(d *warp.AfterConnData) {
	_, err := m.Conn().Exec(
		connQuery,
		d.ConnID,
		d.OccurredAt.Format(warp.TimeFormat),
		d.MailFrom,
		d.MailTo,
	)
	if err != nil {
		fmt.Printf("db exec error: %#v\n", err)
	}
}

var Hook Mysql //nolint:unused
