package models

import (
	"time"

	"github.com/oklog/ulid/v2"
)

// Model is the base model for all models stored in the database.
type Model struct {
	ID       ulid.ULID `json:"id"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
}
