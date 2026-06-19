package rdl

import (
	"fmt"
)

// ManageDataSources adds, removes, or renames DataSources.
func ManageDataSources(path string, adds [][3]string, removes []string, renames [][2]string) (string, error) {
	doc, err := LoadRDL(path)
	if err != nil {
		return "", err
	}

	summary := ""

	for _, name := range removes {
		count := removeDataSource(&doc.Content, name)
		summary += fmt.Sprintf("Removed DataSource '%s' (%d occurrence(s))\n", name, count)
	}

	for _, pair := range renames {
		count := renameDataSource(&doc.Content, pair[0], pair[1])
		summary += fmt.Sprintf("Renamed DataSource '%s' -> '%s' (%d occurrence(s))\n", pair[0], pair[1], count)
	}

	for _, add := range adds {
		addDataSource(&doc.Content, add[0], add[1], add[2])
		summary += fmt.Sprintf("Added DataSource '%s'\n", add[0])
	}

	if err := doc.Save(path); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}
	return summary, nil
}

func removeDataSource(data *[]byte, name string) int {
	count := 0
	startTag := []byte(fmt.Sprintf(`<DataSource Name="%s">`, name))
	endTag := []byte("</DataSource>")

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

func renameDataSource(data *[]byte, old, new string) int {
	count := 0
	oldAttr := []byte(fmt.Sprintf(`DataSourceName>%s</DataSourceName`, old))
	newAttr := []byte(fmt.Sprintf(`DataSourceName>%s</DataSourceName`, new))
	oldName := []byte(fmt.Sprintf(`<DataSource Name="%s">`, old))
	newName := []byte(fmt.Sprintf(`<DataSource Name="%s">`, new))

	newData, c1 := bytesReplaceAllCount(*data, oldAttr, newAttr)
	*data = newData
	count += c1
	newData, c2 := bytesReplaceAllCount(*data, oldName, newName)
	*data = newData
	count += c2
	return count
}

func addDataSource(data *[]byte, name, provider, connStr string) {
	closeTag := []byte("</DataSources>")
	idx := indexByte(*data, closeTag)
	if idx == -1 {
		return
	}

	ds := fmt.Sprintf(`  <DataSource Name="%s">
    <ConnectionProperties>
      <DataProvider>%s</DataProvider>
      <ConnectString>%s</ConnectString>
    </ConnectionProperties>
    <rd:SecurityType>None</rd:SecurityType>
    <rd:DataSourceID>%s</rd:DataSourceID>
  </DataSource>
`, name, provider, connStr, generateGUID())

	insert := []byte(ds)
	*data = append((*data)[:idx], append(insert, (*data)[idx:]...)...)
}
