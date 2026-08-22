package tombstone

import "time"

type Tombstone struct {
	PackID         string
	RevokedAt      time.Time
	ReasonCode     string
	ChecksumSHA256 string
}
