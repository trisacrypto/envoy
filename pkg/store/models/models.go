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

// Scanner is an interface for *sql.Rows and *sql.Row so that models can implement how
// they scan fields into their struct without having to specify every field every time.
type Scanner interface {
	Scan(dest ...any) error
}
