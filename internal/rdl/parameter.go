package rdl

import (
	"fmt"
)

// ManageParameters adds or removes ReportParameters.
func ManageParameters(path string, adds []ParamAddInfo, removes []string) (string, error) {
	doc, err := LoadRDL(path)
	if err != nil {
		return "", err
	}

	summary := ""

	// Process removes
	for _, name := range removes {
		count := removeParameter(&doc.Content, name)
		summary += fmt.Sprintf("Removed parameter '%s' (%d occurrence(s))\n", name, count)
	}

	// Process adds
	for _, add := range adds {
		addParameter(&doc.Content, add)
		summary += fmt.Sprintf("Added parameter '%s' (type=%s, hidden=%s)\n", add.Name, add.Type, add.Hidden)
	}

	if err := doc.Save(path); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}
	return summary, nil
}

func removeParameter(data *[]byte, name string) int {
	count := 0
	startTag := []byte(fmt.Sprintf(`<ReportParameter Name="%s">`, name))
	endTag := []byte("</ReportParameter>")

	for {
		start := indexByte(*data, startTag)
		if start == -1 {
			break
		}
		end := indexByte((*data)[start:], endTag)
		if end == -1 {
			break
		}
		end += start + len(endTag)
		if end < len(*data) && (*data)[end] == '\n' {
			end++
		}
		*data = append((*data)[:start], (*data)[end:]...)
		count++
	}
	return count
}

func addParameter(data *[]byte, info ParamAddInfo) {
	closeTag := []byte("</ReportParameters>")
	idx := indexByte(*data, closeTag)
	if idx == -1 {
		return
	}

	defaultVal := ""
	if info.Default != "" {
		defaultVal = fmt.Sprintf("    <DefaultValue><Values><Value>%s</Value></Values></DefaultValue>\n", info.Default)
	}

	param := fmt.Sprintf(`  <ReportParameter Name="%s">
    <DataType>%s</DataType>
%s    <Hidden>%s</Hidden>
    <Prompt>%s</Prompt>
  </ReportParameter>
`, info.Name, info.Type, defaultVal, info.Hidden, info.Name)

	insert := []byte(param)
	*data = append((*data)[:idx], append(insert, (*data)[idx:]...)...)
}
