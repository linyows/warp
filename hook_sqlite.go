package warp

import (
	"database/sql"
	"fmt"
	"os"
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

type HookSqlite struct {
	pool *sql.DB // Database connection pool.
}

func (h *HookSqlite) Name() string {
	return "sqlite"
}

func (h *HookSqlite) conn() (*sql.DB, error) {
	if h.pool != nil {
		return h.pool, nil
	}

	dsn := os.Getenv("DSN")
	if len(dsn) == 0 {
		return nil, fmt.Errorf("missing dsn for sqlite, please set `DSN`")
	}

	var err error
	h.pool, err = sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open error: %s(%#v)\n", err.Error(), err)
	}

	return h.pool, nil
}

func (h *HookSqlite) AfterInit() {
	conn, err := h.conn()
	if err != nil {
		fmt.Printf("[%s] %s\n", h.Name(), err)
		return
	}

	_, err = conn.Exec(sqliteConnCreateTable)
	if err != nil {
		fmt.Printf("[%s] db exec error: %s\n", h.Name(), err)
	}

	_, err = conn.Exec(sqliteCommCreateTable)
	if err != nil {
		fmt.Printf("[%s] db exec error: %s\n", h.Name(), err)
	}
}

func (h *HookSqlite) AfterComm(d *AfterCommData) {
	conn, err := h.conn()
	if err != nil {
		fmt.Printf("[%s] %s\n", h.Name(), err)
		return
	}

	_, err = conn.Exec(
		sqliteCommQuery,
		GenID().String(),
		d.ConnID,
		d.OccurredAt.Format(TimeFormat),
		d.Direction,
		d.Data,
	)
	if err != nil {
		fmt.Printf("[%s] db exec error: %s\n", h.Name(), err)
	}
}

func (h *HookSqlite) AfterConn(d *AfterConnData) {
	conn, err := h.conn()
	if err != nil {
		fmt.Printf("[%s] %s\n", h.Name(), err)
		return
	}

	_, err = conn.Exec(
		sqliteConnQuery,
		d.ConnID,
		d.OccurredAt.Format(TimeFormat),
		d.MailFrom,
		d.MailTo,
		d.Elapse,
	)
	if err != nil {
		fmt.Printf("[%s] db exec error: %s\n", h.Name(), err)
	}
}
