package output

import (
	"encoding/json"
	"fmt"
	"os"
)

type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Print renders tool-specific or generic output.
func Print(tool string, raw string, jsonOut bool) {
	switch tool {
	case "validations_list":
		PrintValidationsList(raw, jsonOut)
	default:
		// Fallback: raw
		if jsonOut {
			// Try to pretty-print JSON if valid
			var v any
			if err := json.Unmarshal([]byte(raw), &v); err == nil {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				_ = enc.Encode(v)
				return
			}
		}
		fmt.Println(raw)
	}
}

// PrintToolList prints a simple list of tools (name and description).
func PrintToolList(tools []ToolInfo, jsonOut bool) {
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(tools)
		return
	}
	if len(tools) == 0 {
		fmt.Println("No Kiali tools available")
		return
	}
	for _, t := range tools {
		fmt.Printf("- %s: %s\n", t.Name, t.Description)
	}
}
