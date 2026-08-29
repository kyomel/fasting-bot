package domain

import (
	"database/sql/driver"
	"fmt"
	"strconv"

	"github.com/google/uuid"
)

// ID is a branded identifier for domain entities. PostgreSQL stores it as a
// native UUID column; the legacy SQLite path stores the previous integer id
// serialized as text until the full cutover.
type ID string

// NewID returns a time-ordered UUID v7, suitable as a primary key.
func NewID() (ID, error) {
	value, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return ID(value.String()), nil
}

// Scan implements sql.Scanner: accepts UUID text from PostgreSQL, integer ids
// from the legacy SQLite schema, or a plain string.
func (id *ID) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*id = ""
	case string:
		*id = ID(v)
	case []byte:
		*id = ID(v)
	case int64:
		*id = ID(strconv.FormatInt(v, 10))
	default:
		return fmt.Errorf("cannot scan %T into domain.ID", src)
	}
	return nil
}

// Value implements driver.Valuer so domain.ID binds as text.
func (id ID) Value() (driver.Value, error) {
	return string(id), nil
}
