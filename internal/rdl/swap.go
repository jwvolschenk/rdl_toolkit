package rdl

import (
	"fmt"
)

// SwapMacros replaces strings in ConnectString elements.
func SwapMacros(path string, pairs [][2]string) (int, error) {
	doc, err := LoadRDL(path)
	if err != nil {
		return 0, err
	}

	totalCount := 0
	for _, pair := range pairs {
		oldStr := []byte(pair[0])
		newStr := []byte(pair[1])
		count := countInElements(doc.Content, "ConnectString", oldStr)
		totalCount += count
		doc.Content = replaceInElements(doc.Content, "ConnectString", oldStr, newStr)
	}

	if err := doc.Save(path); err != nil {
		return 0, fmt.Errorf("writing file: %w", err)
	}
	return totalCount, nil
}

// SwapFields replaces Fields!X.Value references in Value elements.
func SwapFields(path string, pairs [][2]string) (int, error) {
	doc, err := LoadRDL(path)
	if err != nil {
		return 0, err
	}

	totalCount := 0
	for _, pair := range pairs {
		oldField := []byte("Fields!" + pair[0] + ".Value")
		newField := []byte("Fields!" + pair[1] + ".Value")
		count := countInElements(doc.Content, "Value", oldField)
		totalCount += count
		doc.Content = replaceInElements(doc.Content, "Value", oldField, newField)
	}

	if err := doc.Save(path); err != nil {
		return 0, fmt.Errorf("writing file: %w", err)
	}
	return totalCount, nil
}
