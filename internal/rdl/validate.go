package rdl

import (
	"fmt"
	"regexp"
	"strings"
)

// Validate performs structural validation on an RDL file.
func Validate(path string) (string, error) {
	doc, err := LoadRDL(path)
	if err != nil {
		return "", err
	}

	var issues []string
	content := string(doc.Content)

	// Check cell/column count consistency in Tablix elements
	tablixIssues := validateTablixStructure(doc.Content)
	issues = append(issues, tablixIssues...)

	// Check field-source consistency
	fieldIssues := validateFieldSources(doc.Content)
	issues = append(issues, fieldIssues...)

	// Check row hierarchy
	hierarchyIssues := validateRowHierarchy(doc.Content)
	issues = append(issues, hierarchyIssues...)

	// Check for basic XML well-formedness (simple check)
	if !strings.Contains(content, "<Report") {
		issues = append(issues, "Missing root <Report> element")
	}

	if len(issues) == 0 {
		return fmt.Sprintf("Validation passed for %s (no issues found)", path), nil
	}

	summary := fmt.Sprintf("Validation found %d issue(s) in %s:\n", len(issues), path)
	for _, issue := range issues {
		summary += "  - " + issue + "\n"
	}
	return summary, nil
}

func validateTablixStructure(data []byte) []string {
	var issues []string

	// Count TablixColumnHierarchy columns
	colPattern := regexp.MustCompile(`<TablixColumnHierarchy>.*?</TablixColumnHierarchy>`)
	bodyPattern := regexp.MustCompile(`<TablixBody>.*?</TablixBody>`)

	colMatches := colPattern.FindAll(data, -1)
	bodyMatches := bodyPattern.FindAll(data, -1)

	if len(colMatches) != len(bodyMatches) {
		issues = append(issues, fmt.Sprintf("TablixColumnHierarchy count (%d) != TablixBody count (%d)", len(colMatches), len(bodyMatches)))
	}

	// For each TablixBody, check that all rows have the same number of cells
	for i, body := range bodyMatches {
		rows := findAllSimpleElements(body, "TablixRow")
		if len(rows) == 0 {
			issues = append(issues, fmt.Sprintf("TablixBody %d has no TablixRows", i))
			continue
		}

		expectedCells := -1
		for j, row := range rows {
			cellCount := countSimpleElements(row, "TablixCell")
			if expectedCells == -1 {
				expectedCells = cellCount
			} else if cellCount != expectedCells {
				issues = append(issues, fmt.Sprintf("TablixBody %d, row %d: has %d cells, expected %d", i, j, cellCount, expectedCells))
			}
		}
	}

	return issues
}

func validateFieldSources(data []byte) []string {
	var issues []string

	// Find all DataSet names
	dsPattern := regexp.MustCompile(`<DataSet Name="([^"]+)"`)
	dsMatches := dsPattern.FindAllSubmatch(data, -1)
	dsNames := make(map[string]bool)
	for _, m := range dsMatches {
		dsNames[string(m[1])] = true
	}

	// Find all Field references (Fields!X.Value)
	fieldRefPattern := regexp.MustCompile(`Fields!([^.]+)\.Value`)
	fieldRefs := fieldRefPattern.FindAllSubmatch(data, -1)
	refFields := make(map[string]bool)
	for _, m := range fieldRefs {
		refFields[string(m[1])] = true
	}

	// Find all defined fields
	fieldDefPattern := regexp.MustCompile(`<Field Name="([^"]+)"`)
	fieldDefs := fieldDefPattern.FindAllSubmatch(data, -1)
	defFields := make(map[string]bool)
	for _, m := range fieldDefs {
		defFields[string(m[1])] = true
	}

	// Check that referenced fields are defined
	for field := range refFields {
		if !defFields[field] {
			issues = append(issues, fmt.Sprintf("Field reference 'Fields!%s.Value' used but field not defined in any DataSet", field))
		}
	}

	return issues
}

func validateRowHierarchy(data []byte) []string {
	var issues []string

	// Simple check: TablixRowHierarchy TablixMembers count should match TablixRows count
	hierarchyPattern := regexp.MustCompile(`<TablixRowHierarchy>.*?</TablixRowHierarchy>`)
	bodyPattern := regexp.MustCompile(`<TablixBody>.*?</TablixBody>`)

	hierarchies := hierarchyPattern.FindAll(data, -1)
	bodies := bodyPattern.FindAll(data, -1)

	for i := 0; i < len(hierarchies) && i < len(bodies); i++ {
		hierMembers := countSimpleElements(hierarchies[i], "TablixMember")
		bodyRows := countSimpleElements(bodies[i], "TablixRow")

		// The hierarchy has one extra member for the header
		if hierMembers > 0 && bodyRows > 0 && hierMembers != bodyRows+1 && hierMembers != bodyRows {
			issues = append(issues, fmt.Sprintf("TablixRowHierarchy %d: %d TablixMembers but %d TablixRows (expected members == rows or rows+1)", i, hierMembers, bodyRows))
		}
	}

	return issues
}

func findAllSimpleElements(data []byte, tag string) [][]byte {
	startTag := []byte("<" + tag)
	endTag := []byte("</" + tag + ">")
	var results [][]byte
	idx := 0
	for {
		start := indexByte(data[idx:], startTag)
		if start == -1 {
			break
		}
		start += idx
		end := indexByte(data[start:], endTag)
		if end == -1 {
			break
		}
		end += start + len(endTag)
		results = append(results, data[start:end])
		idx = end
	}
	return results
}

func countSimpleElements(data []byte, tag string) int {
	return len(findAllSimpleElements(data, tag))
}
