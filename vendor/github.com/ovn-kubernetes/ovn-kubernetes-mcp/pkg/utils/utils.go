package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

var (
	// shellMetaCharacters is the pattern for shell metacharacters.
	shellMetaCharacters = regexp.MustCompile(`[;&|$` + "`" + `<>\\()]`)

	// shellMetaCharactersNoBrackets is like shellMetaCharacters but allows brackets.
	shellMetaCharactersNoBrackets = regexp.MustCompile(`[;&|$` + "`" + `<>\\]`)

	// shellMetaCharactersNoBracketsNoAmp is like shellMetaCharactersNoBrackets but allows & for
	// commands which use && for logical AND operations.
	shellMetaCharactersNoBracketsNoAmp = regexp.MustCompile(`[;|$` + "`" + `<>\\]`)

	// shellMetaCharactersNoBracketsSpecialCharacters is like shellMetaCharactersNoBrackets but also
	// disallows newline, NUL byte, and single quote (').
	shellMetaCharactersNoBracketsSpecialCharacters = regexp.MustCompile(`[;&|$` + "`" + `\n\x00` + `'<>\\]`)

	// pathUnsafeChar matches the first rune not allowed in a mount path (alphanumeric, /, -, _, ., ~).
	pathUnsafeChar = regexp.MustCompile(`[^a-zA-Z0-9/_.~-]`)
)

// ShellMetaCharactersType is the type of shell metacharacters to validate.
type ShellMetaCharactersType string

// ShellMetaCharactersType values.
const (
	ShellMetaCharactersTypeDefault                   ShellMetaCharactersType = "default"
	ShellMetaCharactersTypeAllowBrackets             ShellMetaCharactersType = "allow_brackets"
	ShellMetaCharactersTypeAllowBracketsAllowAmp     ShellMetaCharactersType = "allow_brackets_and_amp"
	ShellMetaCharactersTypeDisallowSpecialCharacters ShellMetaCharactersType = "disallow_special_characters"
)

// StripEmptyLines strips empty lines from a slice of strings. It will return a new slice of strings
// with the empty lines removed.
func StripEmptyLines(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	result := []string{}
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result
}

// GetGitRepositoryRoot returns the root directory of the git repository. It will return an error
// if the current directory is not a git repository or the root directory cannot be found.
func GetGitRepositoryRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	for {
		gitDirPath := filepath.Join(currentDir, ".git")
		if _, err := os.Stat(gitDirPath); err == nil {
			return currentDir, nil // Found .git directory, this is the root
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir { // Reached the file system root
			return "", fmt.Errorf("failed to find git repository root")
		}
		currentDir = parentDir
	}
}

// validateShellMetacharacters validates whether the given parameter contains shell
// metacharacters or not. It returns an error if there are any shell metacharacters.
// If shellMetaCharactersType is ShellMetaCharactersTypeDefault, no shell metacharacters are allowed.
// If shellMetaCharactersType is ShellMetaCharactersTypeAllowBracketsAllowAmp, the ( and ) characters and the & character are allowed.
// If shellMetaCharactersType is ShellMetaCharactersTypeAllowBrackets, the ( and ) characters are allowed.
// If shellMetaCharactersType is ShellMetaCharactersTypeDisallowSpecialCharacters, the new line, null byte,
// and special characters are disallowed.
func validateShellMetacharacters(param string, shellMetaCharactersType ShellMetaCharactersType) error {
	switch shellMetaCharactersType {
	case ShellMetaCharactersTypeAllowBracketsAllowAmp:
		if shellMetaCharactersNoBracketsNoAmp.MatchString(param) {
			return fmt.Errorf("invalid use of metacharacters in parameter: %s", param)
		}
	case ShellMetaCharactersTypeAllowBrackets:
		if shellMetaCharactersNoBrackets.MatchString(param) {
			return fmt.Errorf("invalid use of metacharacters in parameter: %s", param)
		}
	case ShellMetaCharactersTypeDisallowSpecialCharacters:
		if shellMetaCharactersNoBracketsSpecialCharacters.MatchString(param) {
			return fmt.Errorf("invalid use of metacharacters in parameter: %s", param)
		}
	case ShellMetaCharactersTypeDefault:
		if shellMetaCharacters.MatchString(param) {
			return fmt.Errorf("invalid use of metacharacters in parameter: %s", param)
		}
	default:
		return fmt.Errorf("invalid shell metacharacters type: %s", shellMetaCharactersType)
	}

	return nil
}

// ValidateSafeString is same as validateShellMetacharacters but it also checks if the string is empty.
// If allowEmpty is true, an empty string is allowed and no error is returned.
func ValidateSafeString(value, fieldName string, allowEmpty bool, shellMetaCharactersType ShellMetaCharactersType) error {
	if value == "" {
		if allowEmpty {
			return nil
		}
		return fmt.Errorf("%s cannot be empty", fieldName)
	}

	if err := validateShellMetacharacters(value, shellMetaCharactersType); err != nil {
		return fmt.Errorf("invalid %s: contains potentially dangerous characters: %w", fieldName, err)
	}
	return nil
}

// ValidatePath validates that a path is safe to use as a filesystem path.
// It ensures the path:
// - Is absolute (starts with /)
// - Does not contain path traversal patterns (..)
// - Contains only safe characters
func ValidatePath(path, pathType string, allowEmpty bool) error {
	if path == "" {
		if allowEmpty {
			return nil
		}
		return fmt.Errorf("%s cannot be empty", pathType)
	}

	// Ensure path is absolute
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%s must be an absolute path (start with /), got: %s", pathType, path)
	}

	// Check for path traversal patterns: reject any path element that is exactly ".."
	if slices.Contains(strings.Split(path, string(filepath.Separator)), "..") {
		return fmt.Errorf("%s contains path traversal element '..': %s", pathType, path)
	}

	// Reject null bytes, control characters, shell specials, and other disallowed runes.
	if loc := pathUnsafeChar.FindStringIndex(path); loc != nil {
		i := loc[0]
		r, size := utf8.DecodeRuneInString(path[i:])
		if r == utf8.RuneError && size <= 1 {
			return fmt.Errorf("%s contains invalid/unsafe byte at position %d: 0x%02X", pathType, i, path[i])
		}
		return fmt.Errorf("%s contains unsafe character at position %d: %c (U+%04X)", pathType, i, r, r)
	}

	return nil
}
