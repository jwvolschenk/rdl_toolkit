package rdl

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// TablixSpec defines the JSON structure for rebuilding a Tablix.
type TablixSpec struct {
	Columns []float64  `json:"columns"`
	Dataset string     `json:"dataset"`
	Rows    []TablixRow `json:"rows"`
}

type TablixRow struct {
	Height string      `json:"height"`
	Cells  []TablixCell `json:"cells"`
}

type TablixCell struct {
	Textbox string      `json:"textbox"`
	Value   string      `json:"value"`
	Colspan int         `json:"colspan"`
	Style   *CellStyle  `json:"style"`
	Format  string      `json:"format"`
}

type CellStyle struct {
	FontSize   string `json:"font_size"`
	FontWeight string `json:"font_weight"`
	FontColor  string `json:"font_color"`
	TextAlign  string `json:"text_align"`
	BgColor    string `json:"bg_color"`
}

// RebuildTablix rebuilds the first Tablix from a JSON spec.
func RebuildTablix(rdlPath, specPath string) (string, error) {
	specData, err := os.ReadFile(specPath)
	if err != nil {
		return "", fmt.Errorf("reading spec: %w", err)
	}

	var spec TablixSpec
	if err := json.Unmarshal(specData, &spec); err != nil {
		return "", fmt.Errorf("parsing spec JSON: %w", err)
	}

	doc, err := LoadRDL(rdlPath)
	if err != nil {
		return "", err
	}

	// Find the first Tablix element
	tablixStart := []byte("<Tablix")
	tablixEnd := []byte("</Tablix>")

	startIdx := indexByte(doc.Content, tablixStart)
	if startIdx == -1 {
		return "", fmt.Errorf("no Tablix found in report")
	}

	// Find the end of opening tag
	openEnd := indexByte(doc.Content[startIdx:], []byte(">"))
	if openEnd == -1 {
		return "", fmt.Errorf("malformed Tablix element")
	}
	openEnd += startIdx

	endIdx := indexByte(doc.Content[startIdx:], tablixEnd)
	if endIdx == -1 {
		return "", fmt.Errorf("Tablix closing tag not found")
	}
	endIdx += startIdx + len(tablixEnd)

	// Build new Tablix XML
	newTablix := buildTablixXML(spec)

	// Replace
	var result []byte
	result = append(result, doc.Content[:startIdx]...)
	result = append(result, []byte(newTablix)...)
	result = append(result, doc.Content[endIdx:]...)
	doc.Content = result

	if err := doc.Save(rdlPath); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}

	totalCells := 0
	for _, row := range spec.Rows {
		totalCells += len(row.Cells)
	}
	return fmt.Sprintf("Rebuilt Tablix with %d columns, %d rows, %d cells",
		len(spec.Columns), len(spec.Rows), totalCells), nil
}

func buildTablixXML(spec TablixSpec) string {
	var b strings.Builder

	b.WriteString(`<Tablix>
  <TablixBody>
    <TablixColumns>`)

	// Column widths
	for _, w := range spec.Columns {
		fmt.Fprintf(&b, `
      <TablixColumn>
        <Width>%.2fcm</Width>
      </TablixColumn>`, w)
	}

	b.WriteString(`
    </TablixColumns>
    <TablixRows>`)

	// Rows
	for _, row := range spec.Rows {
		fmt.Fprintf(&b, `
      <TablixRow>
        <Height>%s</Height>
        <TablixCells>`, row.Height)

		for _, cell := range row.Cells {
			fmt.Fprintf(&b, `
          <TablixCell>
            <CellContents>
              <Textbox Name="%s">`, cell.Textbox)

			// Value
			if cell.Value != "" {
				fmt.Fprintf(&b, `
                <Paragraphs>
                  <Paragraph>
                    <TextRuns>
                      <TextRun>
                        <Value>%s</Value>`, xmlEscapeText(cell.Value))

				if cell.Style != nil || cell.Format != "" {
					b.WriteString(`
                        <Style>`)
					if cell.Style != nil {
						if cell.Style.FontWeight != "" {
							fmt.Fprintf(&b, `
                          <FontWeight>%s</FontWeight>`, cell.Style.FontWeight)
						}
						if cell.Style.FontSize != "" {
							fmt.Fprintf(&b, `
                          <FontSize>%s</FontSize>`, cell.Style.FontSize)
						}
						if cell.Style.FontColor != "" {
							fmt.Fprintf(&b, `
                          <Color>%s</Color>`, cell.Style.FontColor)
						}
					}
					if cell.Format != "" {
						fmt.Fprintf(&b, `
                          <Format>%s</Format>`, cell.Format)
					}
					b.WriteString(`
                        </Style>`)
				}

				b.WriteString(`
                      </TextRun>
                    </TextRuns>`)
				if cell.Style != nil && cell.Style.TextAlign != "" {
					fmt.Fprintf(&b, `
                    <Style>
                      <TextAlign>%s</TextAlign>
                    </Style>`, cell.Style.TextAlign)
				}
				b.WriteString(`
                  </Paragraph>
                </Paragraphs>`)
			}

			// Style (background)
			if cell.Style != nil && cell.Style.BgColor != "" {
				fmt.Fprintf(&b, `
                <Style>
                  <BackgroundColor>%s</BackgroundColor>
                </Style>`, cell.Style.BgColor)
			}

			b.WriteString(`
              </Textbox>`)

			// Colspan
			if cell.Colspan > 1 {
				fmt.Fprintf(&b, `
              <ColSpan>%d</ColSpan>`, cell.Colspan)
			}

			b.WriteString(`
            </CellContents>
          </TablixCell>`)
		}

		b.WriteString(`
        </TablixCells>
      </TablixRow>`)
	}

	b.WriteString(`
    </TablixRows>
  </TablixBody>
  <TablixColumnHierarchy>
    <TablixMembers>`)

	for range spec.Columns {
		b.WriteString(`
      <TablixMember />`)
	}

	b.WriteString(`
    </TablixMembers>
  </TablixColumnHierarchy>
  <TablixRowHierarchy>
    <TablixMembers>
      <TablixMember>
        <KeepWithGroup>After</KeepWithGroup>
      </TablixMember>`)

	for range spec.Rows[1:] {
		b.WriteString(`
      <TablixMember>
        <Group Name="Details" />
      </TablixMember>`)
	}

	b.WriteString(`
    </TablixMembers>
  </TablixRowHierarchy>
  <DataSetName>` + spec.Dataset + `</DataSetName>
</Tablix>`)

	return b.String()
}

func xmlEscapeText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
