package warp

import (
	"time"
)

type Hook interface {
	Name() string
	AfterInit()
	AfterComm(*AfterCommData)
	AfterConn(*AfterConnData)
}

type AfterCommData struct {
	ConnID     string
	OccurredAt time.Time
	Data
	Direction
}

type AfterConnData struct {
	ConnID     string
	OccurredAt time.Time
	MailFrom   []byte
	MailTo     []byte
	Elapse
}
