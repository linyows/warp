package warp

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

const (
	mysqlCommQuery string = "insert into communications (id, connection_id, occurred_at, direction, data) values (?, ?, ?, ?, ?)"
	mysqlConnQuery string = "insert into connections (id, occurred_at, mail_from, mail_to, elapse) values (?, ?, ?, ?, ?)"
)

type HookMysql struct {
	pool *sql.DB // Database connection pool.
}

func (h *HookMysql) Name() string {
	return "mysql"
}

func (h *HookMysql) conn() (*sql.DB, error) {
	if h.pool != nil {
		return h.pool, nil
	}

	dsn := os.Getenv("DSN")
	if len(dsn) == 0 {
		return nil, fmt.Errorf("missing dsn for mysql, please set `DSN`")
	}

	var err error
	h.pool, err = sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open error: s%s\n", err)
	}

	return h.pool, nil
}

func (h *HookMysql) AfterInit() {
}

func (h *HookMysql) AfterComm(d *AfterCommData) {
	conn, err := h.conn()
	if err != nil {
		fmt.Printf("[%s] %s\n", h.Name(), err)
		return
	}

	_, err = conn.Exec(
		mysqlCommQuery,
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

func (h *HookMysql) AfterConn(d *AfterConnData) {
	conn, err := h.conn()
	if err != nil {
		fmt.Printf("[%s] %s\n", h.Name(), err)
		return
	}

	_, err = conn.Exec(
		mysqlConnQuery,
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
