package rdl

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Register adds an RDL file entry to a .rptproj file alphabetically.
func Register(rdlPath, projPath string) (string, error) {
	rdlName := filepath.Base(rdlPath)

	projData, err := os.ReadFile(projPath)
	if err != nil {
		return "", fmt.Errorf("reading project file: %w", err)
	}

	// Check if already registered
	searchStr := []byte(fmt.Sprintf(`Include="%s"`, rdlName))
	if bytes.Contains(projData, searchStr) {
		return fmt.Sprintf("'%s' is already registered in %s", rdlName, projPath), nil
	}

	// Find all <Report Include="..." /> entries to determine insertion point
	entries := findAllReportEntries(projData)

	// Find last </ItemGroup>
	itemGroupEnd := []byte("</ItemGroup>")
	lastItemGroup := bytes.LastIndex(projData, itemGroupEnd)
	if lastItemGroup == -1 {
		return "", fmt.Errorf("no </ItemGroup> found in project file")
	}

	// Insert new entry
	newEntry := fmt.Sprintf("    <Report Include=\"%s\" />\n", rdlName)

	if len(entries) > 0 {
		// Insert alphabetically among existing entries
		insertIdx := -1
		for _, entry := range entries {
			if rdlName < entry {
				searchEntry := []byte(fmt.Sprintf(`Include="%s"`, entry))
				pos := bytes.Index(projData, searchEntry)
				if pos != -1 {
					lineStart := bytes.LastIndex(projData[:pos], []byte("\n"))
					if lineStart == -1 {
						lineStart = 0
					} else {
						lineStart++
					}
					insertIdx = lineStart
				}
				break
			}
		}
		if insertIdx == -1 {
			// Insert after last report entry
			lastEntry := entries[len(entries)-1]
			searchEntry := []byte(fmt.Sprintf(`Include="%s"`, lastEntry))
			pos := bytes.Index(projData, searchEntry)
			if pos != -1 {
				lineEnd := bytes.Index(projData[pos:], []byte("\n"))
				if lineEnd != -1 {
					insertIdx = pos + lineEnd + 1
				}
			}
		}

		if insertIdx != -1 {
			var result []byte
			result = append(result, projData[:insertIdx]...)
			result = append(result, []byte(newEntry)...)
			result = append(result, projData[insertIdx:]...)
			projData = result
		}
	} else {
		// No existing reports - insert before last </ItemGroup>
		var result []byte
		result = append(result, projData[:lastItemGroup]...)
		result = append(result, []byte(newEntry)...)
		result = append(result, projData[lastItemGroup:]...)
		projData = result
	}

	if err := os.WriteFile(projPath, projData, 0644); err != nil {
		return "", fmt.Errorf("writing project file: %w", err)
	}

	return fmt.Sprintf("Registered '%s' in %s (alphabetically)", rdlName, projPath), nil
}

func findAllReportEntries(data []byte) []string {
	var entries []string
	reportTag := []byte(`<Report Include="`)
	idx := 0
	for {
		pos := bytes.Index(data[idx:], reportTag)
		if pos == -1 {
			break
		}
		pos += idx + len(reportTag)
		end := bytes.IndexByte(data[pos:], '"')
		if end == -1 {
			break
		}
		entries = append(entries, string(data[pos:pos+end]))
		idx = pos + end
	}
	sort.Strings(entries)
	return entries
}
