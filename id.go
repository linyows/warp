package warp

import (
	"math/rand"
	"time"

	"github.com/oklog/ulid"
)

func GenID() ulid.ULID {
	seed := time.Now().UnixNano()
	entropy := rand.New(rand.NewSource(seed))
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy)
}
