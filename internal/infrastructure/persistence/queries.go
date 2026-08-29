package persistence

import (
	"fmt"
	"strings"
)

// dialect abstracts the small set of SQL differences between the SQLite and
// PostgreSQL drivers used by the repository layer. The bulk of queries are
// identical apart from placeholders; only boolean literals, upserts and
// date/time arithmetic need per-dialect rendering.
type dialect int

const (
	dialectSQLite dialect = iota
	dialectPostgres
)

// placeholders renders positional placeholders for n values.
// SQLite: ?, ?, ? - PostgreSQL: $1, $2, $3.
func (d dialect) placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	if d == dialectPostgres {
		parts := make([]string, n)
		for i := range parts {
			parts[i] = fmt.Sprintf("$%d", i+1)
		}
		return strings.Join(parts, ", ")
	}
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

// boolLiteral renders a boolean value for the dialect.
func (d dialect) boolLiteral(v bool) string {
	if d == dialectPostgres {
		if v {
			return "true"
		}
		return "false"
	}
	if v {
		return "1"
	}
	return "0"
}

// insertOrIgnore returns the dialect's insert-or-ignore keyword sequence for
// the given table and conflict columns.
func (d dialect) insertOrIgnore(table string, columns []string) string {
	if d == dialectPostgres {
		return fmt.Sprintf("ON CONFLICT (%s) DO NOTHING", strings.Join(columns, ", "))
	}
	return "OR IGNORE"
}

// intervalHours renders "value +/- N hours" arithmetic on a stored datetime
// column for the dialect.
func (d dialect) intervalHours(column string, op string, hours int) string {
	if d == dialectPostgres {
		return fmt.Sprintf("(%s::timestamptz %s %d * interval '1 hour')", column, op, hours)
	}
	return fmt.Sprintf("datetime(%s, '%s%d hours')", column, op, hours)
}

// intervalHoursParam renders "value +/- (? hours)" arithmetic using a
// parameterized hour count.
func (d dialect) intervalHoursParam(column string, op string, param string) string {
	if d == dialectPostgres {
		return fmt.Sprintf("(%s::timestamptz %s (%s::int * interval '1 hour'))", column, op, param)
	}
	return fmt.Sprintf("datetime(%s, '%s' || ? || ' hours')", column, op)
}

// dateOf renders "date part of column".
func (d dialect) dateOf(column string) string {
	if d == dialectPostgres {
		return column + "::date"
	}
	return "DATE(" + column + ")"
}

// minuteString renders column as 'YYYY-MM-DD HH:MM' text for comparisons.
func (d dialect) minuteString(column string) string {
	if d == dialectPostgres {
		return fmt.Sprintf("to_char(%s, 'YYYY-MM-DD HH24:MI')", column)
	}
	return fmt.Sprintf("strftime('%%Y-%%m-%%d %%H:%%M', %s)", column)
}

// leftRight emulates substr(column, 1, n) and substr(column, -n).
func (d dialect) leftRight(column string, start, length int) string {
	if d == dialectPostgres {
		if start == 1 {
			return fmt.Sprintf("left(%s, %d)", column, length)
		}
		return fmt.Sprintf("right(%s, %d)", column, length)
	}
	return fmt.Sprintf("substr(%s, %d, %d)", column, start, length)
}

// charLength renders the character length of column.
func (d dialect) charLength(column string) string {
	if d == dialectPostgres {
		return "char_length(" + column + ")"
	}
	return "length(" + column + ")"
}
