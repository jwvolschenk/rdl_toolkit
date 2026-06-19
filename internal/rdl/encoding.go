package rdl

import (
	"fmt"
)

// FixEncoding ensures UTF-8 BOM and CRLF line endings.
func FixEncoding(path string) (string, error) {
	raw, hasBOM, err := ReadRDL(path)
	if err != nil {
		return "", err
	}

	// Normalize line endings to CRLF
	raw = normalizeCRLF(raw)

	// Ensure BOM
	if !hasBOM {
		raw = append(bomBytes, raw...)
	}

	if err := WriteRDL(path, raw); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}

	status := "already correct"
	if !hasBOM {
		status = "added BOM"
	}
	return fmt.Sprintf("Fixed encoding for %s (%s, CRLF ensured)", path, status), nil
}
