package rdl

import (
	"path/filepath"
	"testing"
)

func mustLoad(t *testing.T) *Document {
	t.Helper()
	doc, err := Load(filepath.Join("testdata", "sample.rdl"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return doc
}

func TestOverview(t *testing.T) {
	o := mustLoad(t).Overview()
	checks := []struct{ name, want string }{
		{"ReportID", "33333333-3333-3333-3333-333333333333"},
		{"Description", "Sales|Q1|Detail"},
		{"Language", "en-GB"},
		{"Author", "Test Author"},
		{"PageWidth", "21cm"},
		{"PageHeight", "29.7cm"},
		{"Orientation", "Portrait"},
	}
	for _, c := range checks {
		got := ""
		switch c.name {
		case "ReportID":
			got = o.ReportID
		case "Description":
			got = o.Description
		case "Language":
			got = o.Language
		case "Author":
			got = o.Author
		case "PageWidth":
			got = o.PageWidth
		case "PageHeight":
			got = o.PageHeight
		case "Orientation":
			got = o.Orientation
		}
		if got != c.want {
			t.Errorf("%s = %q, want %q", c.name, got, c.want)
		}
	}
	if o.DataSourceCount != 2 {
		t.Errorf("DataSourceCount = %d, want 2", o.DataSourceCount)
	}
	if o.DataSetCount != 2 {
		t.Errorf("DataSetCount = %d, want 2", o.DataSetCount)
	}
	if o.ParameterCount != 2 {
		t.Errorf("ParameterCount = %d, want 2", o.ParameterCount)
	}
	if o.TablixCount != 1 {
		t.Errorf("TablixCount = %d, want 1", o.TablixCount)
	}
}

func TestListDataSources(t *testing.T) {
	ds := mustLoad(t).ListDataSources()
	if len(ds) != 2 {
		t.Fatalf("got %d DataSources, want 2", len(ds))
	}
	want0 := DataSourceSummary{
		Name:          "Source1",
		Provider:      "SQL",
		ConnectString: "Data Source=server;Initial Catalog=db;Integrated Security=True",
		SecurityType:  "Integrated",
		DataSourceID:  "11111111-1111-1111-1111-111111111111",
	}
	if ds[0] != want0 {
		t.Errorf("ds[0] = %+v, want %+v", ds[0], want0)
	}
}

func TestListDataSets(t *testing.T) {
	dsets := mustLoad(t).ListDataSets()
	if len(dsets) != 2 {
		t.Fatalf("got %d DataSets, want 2", len(dsets))
	}
	sales := dsets[0]
	if sales.Name != "Sales" {
		t.Errorf("Name = %q, want Sales", sales.Name)
	}
	if sales.DataSource != "Source1" {
		t.Errorf("DataSource = %q, want Source1", sales.DataSource)
	}
	if len(sales.Fields) != 3 {
		t.Errorf("Fields count = %d, want 3", len(sales.Fields))
	}
	if sales.Fields[0].Name != "Id" || sales.Fields[0].DataField != "Id" {
		t.Errorf("Fields[0] = %+v", sales.Fields[0])
	}
	regions := dsets[1]
	if regions.FilterCount != 1 {
		t.Errorf("Regions FilterCount = %d, want 1", regions.FilterCount)
	}
}

func TestListParameters(t *testing.T) {
	ps := mustLoad(t).ListParameters()
	if len(ps) != 2 {
		t.Fatalf("got %d Parameters, want 2", len(ps))
	}
	region := ps[0]
	if region.Name != "Region" || region.DataType != "String" {
		t.Errorf("Region param = %+v", region)
	}
	if !region.Nullable || !region.AllowBlank {
		t.Errorf("Region Nullable=%v AllowBlank=%v, want both true", region.Nullable, region.AllowBlank)
	}
	if region.Default != "ALL" {
		t.Errorf("Region default = %q, want ALL", region.Default)
	}
	if region.Hidden {
		t.Errorf("Region should not be hidden")
	}
	start := ps[1]
	if !start.Hidden {
		t.Errorf("StartDate should be hidden")
	}
}

func TestListTablixes(t *testing.T) {
	ts := mustLoad(t).ListTablixes()
	if len(ts) != 1 {
		t.Fatalf("got %d Tablixes, want 1", len(ts))
	}
	tab := ts[0]
	if tab.Name != "SalesTable" {
		t.Errorf("Name = %q, want SalesTable", tab.Name)
	}
	if tab.DataSet != "Sales" {
		t.Errorf("DataSet = %q, want Sales", tab.DataSet)
	}
	if len(tab.Columns) != 3 {
		t.Errorf("Columns count = %d, want 3", len(tab.Columns))
	}
	if len(tab.Rows) != 2 {
		t.Errorf("Rows count = %d, want 2", len(tab.Rows))
	}
	if len(tab.Rows[0].Cells) != 3 {
		t.Errorf("Row 0 cells = %d, want 3", len(tab.Rows[0].Cells))
	}
	// Header cell
	if tab.Rows[0].Cells[0].Textbox != "HdrId" || tab.Rows[0].Cells[0].Value != "Id" {
		t.Errorf("Row 0 Cell 0 = %+v", tab.Rows[0].Cells[0])
	}
	// Detail cell (expression)
	if tab.Rows[1].Cells[0].Value != "=Fields!Id.Value" {
		t.Errorf("Row 1 Cell 0 Value = %q, want =Fields!Id.Value", tab.Rows[1].Cells[0].Value)
	}
}

func TestGetMetadata(t *testing.T) {
	m := mustLoad(t).GetMetadata()
	if m.Margins.Left != "2cm" || m.Margins.Right != "2cm" {
		t.Errorf("Margins = %+v", m.Margins)
	}
	if m.Orientation != "Portrait" {
		t.Errorf("Orientation = %q, want Portrait", m.Orientation)
	}
}
