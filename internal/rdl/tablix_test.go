package rdl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
)

// helper: load fresh fixture for mutation tests (mustLoad is in inspect_test.go)
// and return the first tablix's name.
func tablixNameOfFirst(t *testing.T, doc *Document) string {
	t.Helper()
	tab := xmlquery.FindOne(doc.Root(), "//Tablix")
	if tab == nil {
		t.Fatal("no tablix in fixture")
	}
	return tab.SelectAttr("Name")
}

// ── Rebuild ────────────────────────────────────────────────────────────────

func TestRebuildTablix_Basic(t *testing.T) {
	doc := mustLoad(t)
	name := tablixNameOfFirst(t, doc)
	spec := TablixSpec{
		Name:    name,
		Columns: []float64{2.5, 5.0, 2.5},
		Dataset: "Sales",
		Rows: []TablixRow{
			{Height: "0.6cm", Cells: []TablixCell{
				{Textbox: "X_H1", Value: "A"},
				{Textbox: "X_H2", Value: "B"},
				{Textbox: "X_H3", Value: "C"},
			}},
			{Height: "0.5cm", Cells: []TablixCell{
				{Textbox: "X_D1", Value: "=Fields!A.Value"},
				{Textbox: "X_D2", Value: "=Fields!B.Value"},
				{Textbox: "X_D3", Value: "=Fields!C.Value"},
			}},
		},
	}
	summary, err := doc.RebuildTablix(spec)
	if err != nil {
		t.Fatalf("RebuildTablix: %v", err)
	}
	if !strings.Contains(summary, "Rebuilt Tablix") {
		t.Errorf("summary: %q", summary)
	}

	// Reload and verify structure.
	re := reloadSaved(t, doc)
	tablixes := re.ListTablixes()
	if len(tablixes) != 1 {
		t.Fatalf("expected 1 tablix, got %d", len(tablixes))
	}
	tab := tablixes[0]
	if len(tab.Columns) != 3 {
		t.Errorf("columns: %d", len(tab.Columns))
	}
	if len(tab.Rows) != 2 {
		t.Errorf("rows: %d", len(tab.Rows))
	}
	if tab.Rows[0].Cells[0].Value != "A" {
		t.Errorf("cell (0,0) value = %q, want 'A'", tab.Rows[0].Cells[0].Value)
	}
}

func TestRebuildTablix_ColspanNoPlaceholders(t *testing.T) {
	doc := mustLoad(t)
	name := tablixNameOfFirst(t, doc)
	// Two cells in a row: first has Colspan=2 (spans cols 0+1), second is regular (col 2).
	// Effective width = 2 + 1 = 3 = column count.
	spec := TablixSpec{
		Name:    name,
		Columns: []float64{2.5, 2.5, 5.0},
		Rows: []TablixRow{
			{Height: "0.5cm", Cells: []TablixCell{
				{Textbox: "Wide", Value: "span", Colspan: 2},
				{Textbox: "Narrow", Value: "one"},
			}},
		},
	}
	if _, err := doc.RebuildTablix(spec); err != nil {
		t.Fatalf("RebuildTablix: %v", err)
	}

	// Verify no placeholder cells: row should have exactly 2 cells.
	re := reloadSaved(t, doc)
	tab := re.ListTablixes()[0]
	if len(tab.Rows[0].Cells) != 2 {
		t.Errorf("row has %d cells, expected 2 (NO placeholders for colspan)", len(tab.Rows[0].Cells))
	}
	// Verify ColSpan attribute survived.
	if tab.Rows[0].Cells[0].Colspan != 2 {
		t.Errorf("cell 0 colspan = %d, want 2", tab.Rows[0].Cells[0].Colspan)
	}
}

func TestRebuildTablix_CellCountMismatch(t *testing.T) {
	doc := mustLoad(t)
	name := tablixNameOfFirst(t, doc)
	spec := TablixSpec{
		Name:    name,
		Columns: []float64{1, 1, 1}, // 3 columns
		Rows: []TablixRow{
			{Cells: []TablixCell{
				{Value: "a"}, {Value: "b"}, // only 2 cells
			}},
		},
	}
	_, err := doc.RebuildTablix(spec)
	if err == nil {
		t.Errorf("expected error for cell/column mismatch")
	}
}

// ── SetCell ────────────────────────────────────────────────────────────────

func TestSetTablixCell_UpdateExisting(t *testing.T) {
	doc := mustLoad(t)
	name := tablixNameOfFirst(t, doc)
	// Cell (0,0) is the header 'Id' textbox in the fixture.
	summary, err := doc.SetTablixCell(name, 0, 0, CellValue{Value: "New Header"})
	if err != nil {
		t.Fatalf("SetTablixCell: %v", err)
	}
	if !strings.Contains(summary, "(0,0)") {
		t.Errorf("summary: %q", summary)
	}

	re := reloadSaved(t, doc)
	cell := re.ListTablixes()[0].Rows[0].Cells[0]
	if cell.Value != "New Header" {
		t.Errorf("cell value = %q, want 'New Header'", cell.Value)
	}
}

func TestSetTablixCell_OutOfRange(t *testing.T) {
	doc := mustLoad(t)
	name := tablixNameOfFirst(t, doc)
	_, err := doc.SetTablixCell(name, 99, 0, CellValue{Value: "X"})
	if err == nil {
		t.Errorf("expected error for row 99")
	}
	_, err = doc.SetTablixCell(name, 0, 99, CellValue{Value: "X"})
	if err == nil {
		t.Errorf("expected error for col 99")
	}
}

// ── AddRow / RemoveRow ─────────────────────────────────────────────────────

func TestAddTablixRow_Append(t *testing.T) {
	doc := mustLoad(t)
	name := tablixNameOfFirst(t, doc)
	origRows := len(doc.ListTablixes()[0].Rows)

	summary, err := doc.AddTablixRow(name, RowSpec{Height: "0.7cm", Cells: []TablixCell{
		{Textbox: "New_A", Value: "X"},
		{Textbox: "New_B", Value: "Y"},
		{Textbox: "New_C", Value: "Z"},
	}}, -1)
	if err != nil {
		t.Fatalf("AddTablixRow: %v", err)
	}
	if !strings.Contains(summary, "Added row") {
		t.Errorf("summary: %q", summary)
	}

	re := reloadSaved(t, doc)
	got := len(re.ListTablixes()[0].Rows)
	if got != origRows+1 {
		t.Errorf("row count = %d, want %d", got, origRows+1)
	}
}

func TestAddRemoveTablixRow_RoundTrip(t *testing.T) {
	doc := mustLoad(t)
	name := tablixNameOfFirst(t, doc)
	orig := len(doc.ListTablixes()[0].Rows)

	if _, err := doc.AddTablixRow(name, RowSpec{
		Cells: []TablixCell{{Value: "a"}, {Value: "b"}, {Value: "c"}},
	}, -1); err != nil {
		t.Fatalf("AddTablixRow: %v", err)
	}
	re := reloadSaved(t, doc)
	if got := len(re.ListTablixes()[0].Rows); got != orig+1 {
		t.Fatalf("after add: %d rows, want %d", got, orig+1)
	}

	// Remove the row we just added (last index).
	if _, err := re.RemoveTablixRow(name, orig); err != nil {
		t.Fatalf("RemoveTablixRow: %v", err)
	}
	re2 := reloadSaved(t, re)
	if got := len(re2.ListTablixes()[0].Rows); got != orig {
		t.Errorf("after remove: %d rows, want %d", got, orig)
	}
}

// ── AddColumn / RemoveColumn ───────────────────────────────────────────────

func TestAddTablixColumn_Append(t *testing.T) {
	doc := mustLoad(t)
	name := tablixNameOfFirst(t, doc)
	origCols := len(doc.ListTablixes()[0].Columns)

	if _, err := doc.AddTablixColumn(name, "4cm", -1); err != nil {
		t.Fatalf("AddTablixColumn: %v", err)
	}
	re := reloadSaved(t, doc)
	tab := re.ListTablixes()[0]
	if len(tab.Columns) != origCols+1 {
		t.Errorf("columns: %d, want %d", len(tab.Columns), origCols+1)
	}
	if tab.Columns[len(tab.Columns)-1].Width != "4cm" {
		t.Errorf("last column width = %q", tab.Columns[len(tab.Columns)-1].Width)
	}
	// Each row should have a new cell at end.
	for i, r := range tab.Rows {
		if len(r.Cells) != origCols+1 {
			t.Errorf("row %d has %d cells, want %d", i, len(r.Cells), origCols+1)
		}
	}
}

func TestRemoveTablixColumn(t *testing.T) {
	doc := mustLoad(t)
	name := tablixNameOfFirst(t, doc)
	origCols := len(doc.ListTablixes()[0].Columns)

	if _, err := doc.RemoveTablixColumn(name, 0); err != nil {
		t.Fatalf("RemoveTablixColumn: %v", err)
	}
	re := reloadSaved(t, doc)
	tab := re.ListTablixes()[0]
	if len(tab.Columns) != origCols-1 {
		t.Errorf("columns: %d, want %d", len(tab.Columns), origCols-1)
	}
	for i, r := range tab.Rows {
		if len(r.Cells) != origCols-1 {
			t.Errorf("row %d has %d cells, want %d", i, len(r.Cells), origCols-1)
		}
	}
}

// ── End-to-end RebuildTablix via file wrapper ──────────────────────────────

func TestRebuildTablix_FileWrapper(t *testing.T) {
	tmp := t.TempDir()
	rdlPath := filepath.Join(tmp, "report.rdl")
	// Copy fixture to tmp via save.
	src := mustLoad(t)
	if err := src.Save(rdlPath); err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join(tmp, "spec.json")
	specJSON := `{"name":"SalesTable","columns":[3.0,3.0],"dataset":"Sales","rows":[{"height":"0.5cm","cells":[{"textbox":"A","value":"X"},{"textbox":"B","value":"Y"}]}]}`
	if err := writeFileBytes(t, specPath, []byte(specJSON)); err != nil {
		t.Fatal(err)
	}

	summary, err := RebuildTablix(rdlPath, specPath, false)
	if err != nil {
		t.Fatalf("RebuildTablix file wrapper: %v", err)
	}
	if !strings.Contains(summary, "Rebuilt Tablix") {
		t.Errorf("summary: %q", summary)
	}
	doc, _ := Load(rdlPath)
	tab := doc.ListTablixes()[0]
	if len(tab.Columns) != 2 {
		t.Errorf("post-rebuild columns: %d", len(tab.Columns))
	}
}

func writeFileBytes(t *testing.T, path string, data []byte) error {
	t.Helper()
	return os.WriteFile(path, data, 0644)
}
