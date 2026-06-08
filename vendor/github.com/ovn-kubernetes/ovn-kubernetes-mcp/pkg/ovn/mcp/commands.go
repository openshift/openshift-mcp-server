package mcp

import (
	"fmt"

	ovntypes "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/ovn/types"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils"
)

const defaultMaxLines = 100

// validateDatabase validates that the database is a valid OVN database.
func validateDatabase(db ovntypes.Database) error {
	switch db {
	case ovntypes.NorthboundDB, ovntypes.SouthboundDB:
		return nil
	default:
		return fmt.Errorf("invalid database %q: must be 'nbdb' or 'sbdb'", db)
	}
}

// validateRecordName validates that a record identifier is safe and non-empty.
func validateRecordName(record string) error {
	return utils.ValidateSafeString(record, "record identifier", false, utils.ShellMetaCharactersTypeDefault)
}

// validateDatapath validates that a datapath name is safe and non-empty.
func validateDatapath(datapath string) error {
	return utils.ValidateSafeString(datapath, "datapath name", false, utils.ShellMetaCharactersTypeDefault)
}

// validateMicroflow validates that a microflow specification is safe and non-empty.
// Microflow specs can contain && for logical AND, so we allow & but block other dangerous chars.
func validateMicroflow(microflow string) error {
	return utils.ValidateSafeString(microflow, "microflow specification", false, utils.ShellMetaCharactersTypeAllowBracketsAllowAmp)
}

// validateColumnSpec validates that a column specification is safe (can be empty).
func validateColumnSpec(columns string) error {
	return utils.ValidateSafeString(columns, "column specification", true, utils.ShellMetaCharactersTypeDefault)
}

// getDBCommand returns the appropriate command (ovn-nbctl or ovn-sbctl) for the given database.
func getDBCommand(db ovntypes.Database) string {
	if db == ovntypes.SouthboundDB {
		return "ovn-sbctl"
	}
	return "ovn-nbctl"
}
