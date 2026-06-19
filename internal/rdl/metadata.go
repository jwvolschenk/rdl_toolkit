package rdl

import (
	"fmt"
)

// UpdateMetadata updates description, title, and/or orientation in the RDL.
func UpdateMetadata(path, description, title, orientation string) (int, error) {
	doc, err := LoadRDL(path)
	if err != nil {
		return 0, err
	}

	count := 0

	if description != "" {
		c := replaceSimpleElement(&doc.Content, "Description", description)
		count += c
	}

	if title != "" {
		c := replaceSimpleElement(&doc.Content, "Language", title)
		// Title in RDL is typically in the report body or as a textbox
		// Also try replacing the Description if it contains title info
		_ = c
		// Look for Title element or textbox with title
		c2 := replaceOrInsertElement(&doc.Content, "Author", title)
		count += c2
	}

	if orientation != "" {
		c := replaceSimpleElement(&doc.Content, "PageWidth", "")
		_ = c
		if orientation == "Landscape" {
			replaceSimpleElement(&doc.Content, "PageWidth", "29.7cm")
			replaceSimpleElement(&doc.Content, "PageHeight", "21cm")
		} else {
			replaceSimpleElement(&doc.Content, "PageWidth", "21cm")
			replaceSimpleElement(&doc.Content, "PageHeight", "29.7cm")
		}
		count++
	}

	if err := doc.Save(path); err != nil {
		return 0, fmt.Errorf("writing file: %w", err)
	}
	return count, nil
}

func replaceSimpleElement(data *[]byte, tag, value string) int {
	startTag := []byte("<" + tag + ">")
	endTag := []byte("</" + tag + ">")

	start := indexByte(*data, startTag)
	if start == -1 {
		return 0
	}
	start += len(startTag)
	end := indexByte((*data)[start:], endTag)
	if end == -1 {
		return 0
	}

	var result []byte
	result = append(result, (*data)[:start]...)
	result = append(result, []byte(value)...)
	result = append(result, (*data)[start+end:]...)
	*data = result
	return 1
}

func replaceOrInsertElement(data *[]byte, tag, value string) int {
	startTag := []byte("<" + tag + ">")
	endTag := []byte("</" + tag + ">")

	start := indexByte(*data, startTag)
	if start == -1 {
		return 0
	}
	start += len(startTag)
	end := indexByte((*data)[start:], endTag)
	if end == -1 {
		return 0
	}

	var result []byte
	result = append(result, (*data)[:start]...)
	result = append(result, []byte(value)...)
	result = append(result, (*data)[start+end:]...)
	*data = result
	return 1
}
