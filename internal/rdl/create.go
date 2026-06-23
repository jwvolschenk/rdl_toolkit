package rdl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/antchfx/xmlquery"
)

// CreateSpec defines the parameters for creating a new RDL from scratch.
type CreateSpec struct {
	// Target file path
	Target string
	// Report title (placed in a title textbox)
	Title string
	// Page orientation: "Portrait" or "Landscape"
	Orientation string
	// Page width (default: 21cm for Portrait, 29.7cm for Landscape)
	PageWidth string
	// Page height (default: 29.7cm for Portrait, 21cm for Landscape)
	PageHeight string
	// Left margin (default: 1cm)
	LeftMargin string
	// Right margin (default: 1cm)
	RightMargin string
	// Top margin (default: 1cm)
	TopMargin string
	// Bottom margin (default: 1cm)
	BottomMargin string
	// Description metadata (pipe-delimited)
	Description string
	// Author (default: "Credo")
	Author string
	// Default font family (default: "Segoe UI")
	FontFamily string
}

// Create builds a minimal RDL skeleton from scratch. The resulting file has:
//   - An empty DataSources section
//   - An empty DataSets section
//   - An empty ReportParameters section
//   - A Body with a single Tablix (1 column, 1 row) bound to no dataset
//   - Page header/footer with title and execution time
//   - Proper XML namespaces and encoding (UTF-8 BOM + CRLF)
//
// The agent then uses rdl_add_datasource, rdl_add_dataset, rdl_rebuild_tablix,
// etc. to build up the report incrementally — no inherited baggage.
func Create(spec CreateSpec, dryRun bool) (string, error) {
	// Apply defaults
	if spec.Orientation == "" {
		spec.Orientation = "Portrait"
	}
	if spec.PageWidth == "" {
		if spec.Orientation == "Landscape" {
			spec.PageWidth = "29.7cm"
		} else {
			spec.PageWidth = "21cm"
		}
	}
	if spec.PageHeight == "" {
		if spec.Orientation == "Landscape" {
			spec.PageHeight = "21cm"
		} else {
			spec.PageHeight = "29.7cm"
		}
	}
	if spec.LeftMargin == "" {
		spec.LeftMargin = "1cm"
	}
	if spec.RightMargin == "" {
		spec.RightMargin = "1cm"
	}
	if spec.TopMargin == "" {
		spec.TopMargin = "1cm"
	}
	if spec.BottomMargin == "" {
		spec.BottomMargin = "1cm"
	}
	if spec.Author == "" {
		spec.Author = "Credo"
	}
	if spec.FontFamily == "" {
		spec.FontFamily = "Segoe UI"
	}

	// Calculate body width (page width - left margin - right margin)
	bodyWidth := subtractMargins(spec.PageWidth, spec.LeftMargin, spec.RightMargin)

	// Build the XML
	xml := buildSkeletonXML(spec, bodyWidth)

	if dryRun {
		return xml, nil
	}

	// Ensure target directory exists
	dir := filepath.Dir(spec.Target)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating target directory: %w", err)
	}

	// Parse and save using Document to get proper BOM + CRLF
	doc, err := parseXMLString(xml)
	if err != nil {
		return "", fmt.Errorf("parsing generated XML: %w", err)
	}

	newID := newUUID()
	doc.SetReportID(newID)

	if err := doc.Save(spec.Target); err != nil {
		return "", fmt.Errorf("writing target: %w", err)
	}

	return newID, nil
}

// buildSkeletonXML generates the minimal RDL XML string.
func buildSkeletonXML(spec CreateSpec, bodyWidth string) string {
	var b strings.Builder

	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	b.WriteString("\n")
	b.WriteString(`<Report MustUnderstand="df" xmlns="http://schemas.microsoft.com/sqlserver/reporting/2016/01/reportdefinition" xmlns:rd="http://schemas.microsoft.com/SQLServer/reporting/reportdesigner" xmlns:df="http://schemas.microsoft.com/sqlserver/reporting/2016/01/reportdefinition/defaultfontfamily">`)
	b.WriteString("\n")

	// Default font family
	b.WriteString(`  <df:DefaultFontFamily>`)
	b.WriteString(spec.FontFamily)
	b.WriteString(`</df:DefaultFontFamily>`)
	b.WriteString("\n")

	// Description
	b.WriteString(`  <Description>`)
	b.WriteString(escapeXML(spec.Description))
	b.WriteString(`</Description>`)
	b.WriteString("\n")

	// Author
	b.WriteString(`  <Author>`)
	b.WriteString(escapeXML(spec.Author))
	b.WriteString(`</Author>`)
	b.WriteString("\n")

	b.WriteString(`  <AutoRefresh>0</AutoRefresh>`)
	b.WriteString("\n")

	// DataSources (empty)
	b.WriteString(`  <DataSources />`)
	b.WriteString("\n")

	// DataSets (empty)
	b.WriteString(`  <DataSets />`)
	b.WriteString("\n")

	// ReportSections
	b.WriteString(`  <ReportSections>`)
	b.WriteString("\n")
	b.WriteString(`    <ReportSection>`)
	b.WriteString("\n")

	// Body
	b.WriteString(`      <Body>`)
	b.WriteString("\n")
	b.WriteString(`        <ReportItems>`)
	b.WriteString("\n")

	// Tablix (minimal: 1 column, 1 row, no dataset binding)
	b.WriteString(`          <Tablix Name="Tablix1">`)
	b.WriteString("\n")
	b.WriteString(`            <TablixBody>`)
	b.WriteString("\n")
	b.WriteString(`              <TablixColumns>`)
	b.WriteString("\n")
	b.WriteString(`                <TablixColumn>`)
	b.WriteString("\n")
	b.WriteString(`                  <Width>` + bodyWidth + `</Width>`)
	b.WriteString("\n")
	b.WriteString(`                </TablixColumn>`)
	b.WriteString("\n")
	b.WriteString(`              </TablixColumns>`)
	b.WriteString("\n")
	b.WriteString(`              <TablixRows>`)
	b.WriteString("\n")
	b.WriteString(`                <TablixRow>`)
	b.WriteString("\n")
	b.WriteString(`                  <Height>1cm</Height>`)
	b.WriteString("\n")
	b.WriteString(`                  <TablixCells>`)
	b.WriteString("\n")
	b.WriteString(`                    <TablixCell>`)
	b.WriteString("\n")
	b.WriteString(`                      <CellContents>`)
	b.WriteString("\n")
	b.WriteString(`                        <Textbox Name="Placeholder">`)
	b.WriteString("\n")
	b.WriteString(`                          <Paragraphs>`)
	b.WriteString("\n")
	b.WriteString(`                            <Paragraph>`)
	b.WriteString("\n")
	b.WriteString(`                              <TextRuns>`)
	b.WriteString("\n")
	b.WriteString(`                                <TextRun>`)
	b.WriteString("\n")
	b.WriteString(`                                  <Value>` + escapeXML(spec.Title) + `</Value>`)
	b.WriteString("\n")
	b.WriteString(`                                  <Style>`)
	b.WriteString("\n")
	b.WriteString(`                                    <FontWeight>Bold</FontWeight>`)
	b.WriteString("\n")
	b.WriteString(`                                  </Style>`)
	b.WriteString("\n")
	b.WriteString(`                                </TextRun>`)
	b.WriteString("\n")
	b.WriteString(`                              </TextRuns>`)
	b.WriteString("\n")
	b.WriteString(`                              <Style />`)
	b.WriteString("\n")
	b.WriteString(`                            </Paragraph>`)
	b.WriteString("\n")
	b.WriteString(`                          </Paragraphs>`)
	b.WriteString("\n")
	b.WriteString(`                          <rd:DefaultName>Placeholder</rd:DefaultName>`)
	b.WriteString("\n")
	b.WriteString(`                          <Style>`)
	b.WriteString("\n")
	b.WriteString(`                            <Border>`)
	b.WriteString("\n")
	b.WriteString(`                              <Style>None</Style>`)
	b.WriteString("\n")
	b.WriteString(`                            </Border>`)
	b.WriteString("\n")
	b.WriteString(`                          </Style>`)
	b.WriteString("\n")
	b.WriteString(`                        </Textbox>`)
	b.WriteString("\n")
	b.WriteString(`                      </CellContents>`)
	b.WriteString("\n")
	b.WriteString(`                    </TablixCell>`)
	b.WriteString("\n")
	b.WriteString(`                  </TablixCells>`)
	b.WriteString("\n")
	b.WriteString(`                </TablixRow>`)
	b.WriteString("\n")
	b.WriteString(`              </TablixRows>`)
	b.WriteString("\n")
	b.WriteString(`            </TablixBody>`)
	b.WriteString("\n")
	b.WriteString(`            <TablixColumnHierarchy>`)
	b.WriteString("\n")
	b.WriteString(`              <TablixMembers>`)
	b.WriteString("\n")
	b.WriteString(`                <TablixMember />`)
	b.WriteString("\n")
	b.WriteString(`              </TablixMembers>`)
	b.WriteString("\n")
	b.WriteString(`            </TablixColumnHierarchy>`)
	b.WriteString("\n")
	b.WriteString(`            <TablixRowHierarchy>`)
	b.WriteString("\n")
	b.WriteString(`              <TablixMembers>`)
	b.WriteString("\n")
	b.WriteString(`                <TablixMember>`)
	b.WriteString("\n")
	b.WriteString(`                  <KeepWithGroup>After</KeepWithGroup>`)
	b.WriteString("\n")
	b.WriteString(`                </TablixMember>`)
	b.WriteString("\n")
	b.WriteString(`              </TablixMembers>`)
	b.WriteString("\n")
	b.WriteString(`            </TablixRowHierarchy>`)
	b.WriteString("\n")
	b.WriteString(`            <DataSetName></DataSetName>`)
	b.WriteString("\n")
	b.WriteString(`            <Top>0cm</Top>`)
	b.WriteString("\n")
	b.WriteString(`            <Left>0cm</Left>`)
	b.WriteString("\n")
	b.WriteString(`            <Height>1cm</Height>`)
	b.WriteString("\n")
	b.WriteString(`            <Width>` + bodyWidth + `</Width>`)
	b.WriteString("\n")
	b.WriteString(`            <Style>`)
	b.WriteString("\n")
	b.WriteString(`              <Border>`)
	b.WriteString("\n")
	b.WriteString(`                <Style>None</Style>`)
	b.WriteString("\n")
	b.WriteString(`              </Border>`)
	b.WriteString("\n")
	b.WriteString(`            </Style>`)
	b.WriteString("\n")
	b.WriteString(`          </Tablix>`)
	b.WriteString("\n")

	b.WriteString(`        </ReportItems>`)
	b.WriteString("\n")
	b.WriteString(`        <Height>2cm</Height>`)
	b.WriteString("\n")
	b.WriteString(`        <Style>`)
	b.WriteString("\n")
	b.WriteString(`          <Border>`)
	b.WriteString("\n")
	b.WriteString(`            <Style>None</Style>`)
	b.WriteString("\n")
	b.WriteString(`          </Border>`)
	b.WriteString("\n")
	b.WriteString(`        </Style>`)
	b.WriteString("\n")
	b.WriteString(`      </Body>`)
	b.WriteString("\n")

	// Width
	b.WriteString(`      <Width>` + bodyWidth + `</Width>`)
	b.WriteString("\n")

	// Page
	b.WriteString(`      <Page>`)
	b.WriteString("\n")

	// PageHeader
	b.WriteString(`        <PageHeader>`)
	b.WriteString("\n")
	b.WriteString(`          <Height>1.5cm</Height>`)
	b.WriteString("\n")
	b.WriteString(`          <PrintOnFirstPage>true</PrintOnFirstPage>`)
	b.WriteString("\n")
	b.WriteString(`          <PrintOnLastPage>true</PrintOnLastPage>`)
	b.WriteString("\n")
	b.WriteString(`          <ReportItems>`)
	b.WriteString("\n")

	// Title textbox in page header
	b.WriteString(`            <Textbox Name="ReportTitle">`)
	b.WriteString("\n")
	b.WriteString(`              <CanGrow>true</CanGrow>`)
	b.WriteString("\n")
	b.WriteString(`              <KeepTogether>true</KeepTogether>`)
	b.WriteString("\n")
	b.WriteString(`              <Paragraphs>`)
	b.WriteString("\n")
	b.WriteString(`                <Paragraph>`)
	b.WriteString("\n")
	b.WriteString(`                  <TextRuns>`)
	b.WriteString("\n")
	b.WriteString(`                    <TextRun>`)
	b.WriteString("\n")
	b.WriteString(`                      <Value>` + escapeXML(spec.Title) + `</Value>`)
	b.WriteString("\n")
	b.WriteString(`                      <Style>`)
	b.WriteString("\n")
	b.WriteString(`                        <FontFamily>` + spec.FontFamily + ` Light</FontFamily>`)
	b.WriteString("\n")
	b.WriteString(`                        <FontSize>16pt</FontSize>`)
	b.WriteString("\n")
	b.WriteString(`                      </Style>`)
	b.WriteString("\n")
	b.WriteString(`                    </TextRun>`)
	b.WriteString("\n")
	b.WriteString(`                  </TextRuns>`)
	b.WriteString("\n")
	b.WriteString(`                  <Style />`)
	b.WriteString("\n")
	b.WriteString(`                </Paragraph>`)
	b.WriteString("\n")
	b.WriteString(`              </Paragraphs>`)
	b.WriteString("\n")
	b.WriteString(`              <rd:DefaultName>ReportTitle</rd:DefaultName>`)
	b.WriteString("\n")
	b.WriteString(`              <Top>0cm</Top>`)
	b.WriteString("\n")
	b.WriteString(`              <Left>0cm</Left>`)
	b.WriteString("\n")
	b.WriteString(`              <Height>1cm</Height>`)
	b.WriteString("\n")
	b.WriteString(`              <Width>` + bodyWidth + `</Width>`)
	b.WriteString("\n")
	b.WriteString(`              <Style>`)
	b.WriteString("\n")
	b.WriteString(`                <Border>`)
	b.WriteString("\n")
	b.WriteString(`                  <Style>None</Style>`)
	b.WriteString("\n")
	b.WriteString(`                </Border>`)
	b.WriteString("\n")
	b.WriteString(`              </Style>`)
	b.WriteString("\n")
	b.WriteString(`            </Textbox>`)
	b.WriteString("\n")

	b.WriteString(`          </ReportItems>`)
	b.WriteString("\n")
	b.WriteString(`          <Style>`)
	b.WriteString("\n")
	b.WriteString(`            <Border>`)
	b.WriteString("\n")
	b.WriteString(`              <Style>None</Style>`)
	b.WriteString("\n")
	b.WriteString(`            </Border>`)
	b.WriteString("\n")
	b.WriteString(`          </Style>`)
	b.WriteString("\n")
	b.WriteString(`        </PageHeader>`)
	b.WriteString("\n")

	// PageFooter
	b.WriteString(`        <PageFooter>`)
	b.WriteString("\n")
	b.WriteString(`          <Height>1cm</Height>`)
	b.WriteString("\n")
	b.WriteString(`          <PrintOnFirstPage>true</PrintOnFirstPage>`)
	b.WriteString("\n")
	b.WriteString(`          <PrintOnLastPage>true</PrintOnLastPage>`)
	b.WriteString("\n")
	b.WriteString(`          <ReportItems>`)
	b.WriteString("\n")
	b.WriteString(`            <Textbox Name="ExecutionTime">`)
	b.WriteString("\n")
	b.WriteString(`              <CanGrow>true</CanGrow>`)
	b.WriteString("\n")
	b.WriteString(`              <KeepTogether>true</KeepTogether>`)
	b.WriteString("\n")
	b.WriteString(`              <Paragraphs>`)
	b.WriteString("\n")
	b.WriteString(`                <Paragraph>`)
	b.WriteString("\n")
	b.WriteString(`                  <TextRuns>`)
	b.WriteString("\n")
	b.WriteString(`                    <TextRun>`)
	b.WriteString("\n")
	b.WriteString(`                      <Value>=Globals!ExecutionTime</Value>`)
	b.WriteString("\n")
	b.WriteString(`                      <Style />`)
	b.WriteString("\n")
	b.WriteString(`                    </TextRun>`)
	b.WriteString("\n")
	b.WriteString(`                  </TextRuns>`)
	b.WriteString("\n")
	b.WriteString(`                  <Style>`)
	b.WriteString("\n")
	b.WriteString(`                    <TextAlign>Right</TextAlign>`)
	b.WriteString("\n")
	b.WriteString(`                  </Style>`)
	b.WriteString("\n")
	b.WriteString(`                </Paragraph>`)
	b.WriteString("\n")
	b.WriteString(`              </Paragraphs>`)
	b.WriteString("\n")
	b.WriteString(`              <rd:DefaultName>ExecutionTime</rd:DefaultName>`)
	b.WriteString("\n")
	b.WriteString(`              <Top>0cm</Top>`)
	b.WriteString("\n")
	b.WriteString(`              <Left>` + subtractMargins(bodyWidth, "5cm", "0cm") + `</Left>`)
	b.WriteString("\n")
	b.WriteString(`              <Height>0.6cm</Height>`)
	b.WriteString("\n")
	b.WriteString(`              <Width>5cm</Width>`)
	b.WriteString("\n")
	b.WriteString(`              <Style>`)
	b.WriteString("\n")
	b.WriteString(`                <Border>`)
	b.WriteString("\n")
	b.WriteString(`                  <Style>None</Style>`)
	b.WriteString("\n")
	b.WriteString(`                </Border>`)
	b.WriteString("\n")
	b.WriteString(`              </Style>`)
	b.WriteString("\n")
	b.WriteString(`            </Textbox>`)
	b.WriteString("\n")
	b.WriteString(`          </ReportItems>`)
	b.WriteString("\n")
	b.WriteString(`          <Style>`)
	b.WriteString("\n")
	b.WriteString(`            <Border>`)
	b.WriteString("\n")
	b.WriteString(`              <Style>None</Style>`)
	b.WriteString("\n")
	b.WriteString(`            </Border>`)
	b.WriteString("\n")
	b.WriteString(`          </Style>`)
	b.WriteString("\n")
	b.WriteString(`        </PageFooter>`)
	b.WriteString("\n")

	// Page dimensions
	b.WriteString(`        <PageHeight>` + spec.PageHeight + `</PageHeight>`)
	b.WriteString("\n")
	b.WriteString(`        <PageWidth>` + spec.PageWidth + `</PageWidth>`)
	b.WriteString("\n")
	b.WriteString(`        <LeftMargin>` + spec.LeftMargin + `</LeftMargin>`)
	b.WriteString("\n")
	b.WriteString(`        <RightMargin>` + spec.RightMargin + `</RightMargin>`)
	b.WriteString("\n")
	b.WriteString(`        <TopMargin>` + spec.TopMargin + `</TopMargin>`)
	b.WriteString("\n")
	b.WriteString(`        <BottomMargin>` + spec.BottomMargin + `</BottomMargin>`)
	b.WriteString("\n")

	b.WriteString(`      </Page>`)
	b.WriteString("\n")
	b.WriteString(`    </ReportSection>`)
	b.WriteString("\n")
	b.WriteString(`  </ReportSections>`)
	b.WriteString("\n")

	// ReportParameters (empty)
	b.WriteString(`  <ReportParameters />`)
	b.WriteString("\n")

	b.WriteString(`</Report>`)
	b.WriteString("\n")

	return b.String()
}

// subtractMargins parses a dimension string like "21cm" and subtracts margins.
func subtractMargins(dimension, left, right string) string {
	d := parseDim(dimension)
	l := parseDim(left)
	r := parseDim(right)
	result := d - l - r
	if result < 0 {
		result = 0
	}
	return fmt.Sprintf("%.2fcm", result)
}

// parseDim parses a dimension string like "21cm", "1cm", "29.7cm" and returns the numeric value.
func parseDim(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "cm")
	s = strings.TrimSuffix(s, "mm")
	var v float64
	fmt.Sscanf(s, "%f", &v)
	if strings.Contains(s, "mm") || strings.HasSuffix(strings.TrimSpace(s), "mm") {
		// Already handled by trim
	}
	// If original had "mm", convert to cm
	if strings.HasSuffix(strings.TrimSpace(s), "mm") {
		v = v / 10.0
	}
	return v
}

// parseXMLString parses an XML string into a Document.
func parseXMLString(xml string) (*Document, error) {
	root, err := xmlquery.Parse(strings.NewReader(xml))
	if err != nil {
		return nil, fmt.Errorf("parsing XML: %w", err)
	}
	return &Document{root: root, hasBOM: true}, nil
}

// escapeXML escapes special XML characters.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
