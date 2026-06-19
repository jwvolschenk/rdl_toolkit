package rdl

import (
	"fmt"
	"strings"
)

// ManageDataSets adds, removes, or renames DataSets and adds fields.
func ManageDataSets(path string, adds []DatasetAddInfo, removes []string, renames [][2]string, addFields [][2]string) (string, error) {
	doc, err := LoadRDL(path)
	if err != nil {
		return "", err
	}

	summary := ""

	// Process removes
	for _, name := range removes {
		count := removeDataSet(&doc.Content, name)
		summary += fmt.Sprintf("Removed DataSet '%s' (%d occurrence(s))\n", name, count)
	}

	// Process renames
	for _, pair := range renames {
		count := renameDataSet(&doc.Content, pair[0], pair[1])
		summary += fmt.Sprintf("Renamed DataSet '%s' -> '%s' (%d occurrence(s))\n", pair[0], pair[1], count)
	}

	// Process adds
	for _, add := range adds {
		addDataSet(&doc.Content, add)
		summary += fmt.Sprintf("Added DataSet '%s'\n", add.Name)
	}

	// Process add-field
	for _, af := range addFields {
		dsName := af[0]
		fieldName := af[1]
		addFieldToDataSet(&doc.Content, dsName, fieldName)
		summary += fmt.Sprintf("Added field '%s' to DataSet '%s'\n", fieldName, dsName)
	}

	if err := doc.Save(path); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}
	return summary, nil
}

func removeDataSet(data *[]byte, name string) int {
	count := 0
	startTag := []byte(fmt.Sprintf(`<DataSet Name="%s">`, name))
	endTag := []byte("</DataSet>")

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

func renameDataSet(data *[]byte, old, new string) int {
	count := 0
	oldName := []byte(fmt.Sprintf(`<DataSet Name="%s">`, old))
	newName := []byte(fmt.Sprintf(`<DataSet Name="%s">`, new))
	*data, count = bytesReplaceAllCount(*data, oldName, newName)
	return count
}

func addDataSet(data *[]byte, info DatasetAddInfo) {
	closeTag := []byte("</DataSets>")
	idx := indexByte(*data, closeTag)
	if idx == -1 {
		return
	}

	var fields strings.Builder
	for _, f := range info.Fields {
		fmt.Fprintf(&fields, "      <Field Name=\"%s\">\n        <DataField>%s</DataField>\n      </Field>\n", f, f)
	}

	ds := fmt.Sprintf(`  <DataSet Name="%s">
    <Query>
      <DataSourceName>%s</DataSourceName>
      <CommandText>%s</CommandText>
    </Query>
    <Fields>
%s    </Fields>
  </DataSet>
`, info.Name, info.DS, info.CmdText, fields.String())

	insert := []byte(ds)
	*data = append((*data)[:idx], append(insert, (*data)[idx:]...)...)
}

func addFieldToDataSet(data *[]byte, dsName, fieldName string) {
	// Find the DataSet
	startTag := []byte(fmt.Sprintf(`<DataSet Name="%s">`, dsName))
	start := indexByte(*data, startTag)
	if start == -1 {
		return
	}

	// Find </Fields> within this DataSet
	searchFrom := start
	endTag := []byte("</DataSet>")
	dsEnd := indexByte((*data)[searchFrom:], endTag)
	if dsEnd == -1 {
		return
	}
	dsEnd += searchFrom

	fieldsEnd := []byte("</Fields>")
	fieldsIdx := indexByte((*data)[searchFrom:dsEnd], fieldsEnd)
	if fieldsIdx == -1 {
		return
	}
	fieldsIdx += searchFrom

	field := fmt.Sprintf("      <Field Name=\"%s\">\n        <DataField>%s</DataField>\n      </Field>\n", fieldName, fieldName)
	insert := []byte(field)
	*data = append((*data)[:fieldsIdx], append(insert, (*data)[fieldsIdx:]...)...)
}
