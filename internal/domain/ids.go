package domain

import (
	"database/sql/driver"
	"fmt"

	"github.com/google/uuid"
)

// ID is a branded UUID identifier for domain entities. PostgreSQL stores it
// as a native UUID column.
type ID string

// NewID returns a time-ordered UUID v7, suitable as a primary key.
func NewID() (ID, error) {
	value, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return ID(value.String()), nil
}

// Scan implements sql.Scanner: accepts UUID text from PostgreSQL.
func (id *ID) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*id = ""
	case string:
		*id = ID(v)
	case []byte:
		*id = ID(v)
	default:
		return fmt.Errorf("cannot scan %T into domain.ID", src)
	}
	return nil
}

// Value implements driver.Valuer so domain.ID binds as text.
func (id ID) Value() (driver.Value, error) {
	return string(id), nil
}
