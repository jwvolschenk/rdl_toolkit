package rdl

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"regexp"
)

const (
	RDLNamespace = "http://schemas.microsoft.com/sqlserver/reporting/2016/01/reportdefinition"
	RDNamespace  = "http://schemas.microsoft.com/SQLServer/reporting/reportdesigner"
	DFNamespace  = "http://schemas.microsoft.com/sqlserver/reporting/2016/01/reportdefinition/defaultfontfamily"
)

var (
	bomBytes = []byte{0xEF, 0xBB, 0xBF}
)

// ReadRDL reads an RDL file and returns raw bytes preserving BOM/encoding info.
// Returns content without BOM, and whether BOM was present.
func ReadRDL(path string) ([]byte, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false, fmt.Errorf("reading file: %w", err)
	}
	hasBOM := bytes.HasPrefix(raw, bomBytes)
	if hasBOM {
		raw = raw[3:]
	}
	return raw, hasBOM, nil
}

// WriteRDL writes RDL content with UTF-8 BOM and CRLF line endings.
func WriteRDL(path string, content []byte) error {
	content = normalizeCRLF(content)
	out := append(bomBytes, content...)
	return os.WriteFile(path, out, 0644)
}

// normalizeCRLF ensures all line endings are CRLF.
func normalizeCRLF(data []byte) []byte {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
	data = bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n"))
	return data
}

// RDLDocument represents a parsed RDL XML document for manipulation.
type RDLDocument struct {
	Content []byte
}

// LoadRDL loads an RDL file into an RDLDocument.
func LoadRDL(path string) (*RDLDocument, error) {
	raw, _, err := ReadRDL(path)
	if err != nil {
		return nil, err
	}
	raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
	raw = bytes.ReplaceAll(raw, []byte("\r"), []byte("\n"))
	return &RDLDocument{Content: raw}, nil
}

// Save writes the document back to disk with BOM and CRLF.
func (d *RDLDocument) Save(path string) error {
	return WriteRDL(path, d.Content)
}

// CountOccurrences counts non-overlapping occurrences of substr in data.
func CountOccurrences(data, substr []byte) int {
	count := 0
	idx := 0
	for {
		pos := bytes.Index(data[idx:], substr)
		if pos == -1 {
			break
		}
		count++
		idx += pos + len(substr)
	}
	return count
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

// regex helper
func replaceRegex(data []byte, pattern string, repl []byte) ([]byte, int) {
	re := regexp.MustCompile(pattern)
	matches := re.FindAll(data, -1)
	return re.ReplaceAll(data, repl), len(matches)
}

// DatasourceInfo holds parsed DataSource information.
type DataSourceInfo struct {
	Name          string
	Provider      string
	ConnectString string
}

// DatasetAddInfo holds info for adding a dataset.
type DatasetAddInfo struct {
	Name    string
	DS      string
	CmdText string
	Fields  []string
}

// ParamAddInfo holds info for adding a parameter.
type ParamAddInfo struct {
	Name    string
	Type    string
	Default string
	Hidden  string
}

// bytesEqual compares two byte slices.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// indexByte finds the first occurrence of sep in data.
func indexByte(data, sep []byte) int {
	for i := 0; i <= len(data)-len(sep); i++ {
		if bytesEqual(data[i:i+len(sep)], sep) {
			return i
		}
	}
	return -1
}

// bytesReplaceAll replaces all occurrences of old with new in data.
func bytesReplaceAll(data, old, new []byte) []byte {
	var result []byte
	i := 0
	for i <= len(data)-len(old) {
		if bytesEqual(data[i:i+len(old)], old) {
			result = append(result, new...)
			i += len(old)
		} else {
			result = append(result, data[i])
			i++
		}
	}
	if i < len(data) {
		result = append(result, data[i:]...)
	}
	return result
}

// bytesReplaceAllCount replaces all occurrences and returns count.
func bytesReplaceAllCount(data, old, new []byte) ([]byte, int) {
	count := 0
	var result []byte
	i := 0
	for i <= len(data)-len(old) {
		if bytesEqual(data[i:i+len(old)], old) {
			result = append(result, new...)
			i += len(old)
			count++
		} else {
			result = append(result, data[i])
			i++
		}
	}
	if i < len(data) {
		result = append(result, data[i:]...)
	}
	return result, count
}

// countSub counts non-overlapping occurrences of sub in data.
func countSub(data, sub []byte) int {
	count := 0
	for i := 0; i <= len(data)-len(sub); i++ {
		if bytesEqual(data[i:i+len(sub)], sub) {
			count++
			i += len(sub) - 1
		}
	}
	return count
}

// countInElements counts occurrences of substr within elements with the given tag.
func countInElements(data []byte, tag string, substr []byte) int {
	count := 0
	startTag := []byte("<" + tag + ">")
	endTag := []byte("</" + tag + ">")
	idx := 0
	for {
		start := indexByte(data[idx:], startTag)
		if start == -1 {
			break
		}
		start += idx + len(startTag)
		end := indexByte(data[start:], endTag)
		if end == -1 {
			break
		}
		content := data[start : start+end]
		count += countSub(content, substr)
		idx = start + end + len(endTag)
	}
	return count
}

// replaceInElements replaces substr within elements with the given tag.
func replaceInElements(data []byte, tag string, old, new []byte) []byte {
	startTag := []byte("<" + tag + ">")
	endTag := []byte("</" + tag + ">")
	var result []byte
	idx := 0
	for {
		start := indexByte(data[idx:], startTag)
		if start == -1 {
			result = append(result, data[idx:]...)
			break
		}
		start += idx
		result = append(result, data[idx:start+len(startTag)]...)
		start += len(startTag)
		end := indexByte(data[start:], endTag)
		if end == -1 {
			result = append(result, data[start:]...)
			break
		}
		content := data[start : start+end]
		content = bytesReplaceAll(content, old, new)
		result = append(result, content...)
		result = append(result, endTag...)
		idx = start + end + len(endTag)
	}
	return result
}
