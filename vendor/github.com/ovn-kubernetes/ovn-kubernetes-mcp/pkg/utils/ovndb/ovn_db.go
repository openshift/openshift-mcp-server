package ovndb

import (
	"fmt"
	"regexp"
)

// validOVNTableNamePattern matches valid OVN table names: start with letter, alphanumeric
// and underscores.
var validOVNTableNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// ValidateOVNTableName validates that a table name is safe and non-empty.
// Table names should only contain alphanumeric characters and underscores.
func ValidateOVNTableName(table string) error {
	if table == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	if !validOVNTableNamePattern.MatchString(table) {
		return fmt.Errorf("invalid table name %q: must start with a letter and contain only alphanumeric characters and underscores", table)
	}

	return nil
}
