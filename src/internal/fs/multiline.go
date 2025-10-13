package fs

import "strings"

// EncodeMultilineValue prepares a string with line breaks for storage in the env config file.
func EncodeMultilineValue(value string) string {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(normalized, "\n", "\\n")
}

// DecodeMultilineValue restores a previously encoded multiline string.
func DecodeMultilineValue(value string) string {
	if value == "" {
		return ""
	}
	decoded := strings.ReplaceAll(value, "\\n", "\n")
	return decoded
}
