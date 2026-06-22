package rdl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMultiTablix_ByTargeting verifies that per-tablix operations hit the
// named tablix and leave the other one alone.
func TestMultiTablix_ByTargeting(t *testing.T) {
	doc, err := Load(filepath.Join("testdata", "multi_tablix.rdl"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	tabs := doc.ListTablixes()
	if len(tabs) != 2 {
		t.Fatalf("expected 2 tablixes, got %d", len(tabs))
	}
	if tabs[0].Name != "SalesTablix" || tabs[1].Name != "ReturnsTablix" {
		t.Errorf("names: %+v", tabs)
	}

	// Set cell (0,0) of ReturnsTablix; SalesTablix must be untouched.
	if _, err := doc.SetTablixCell("ReturnsTablix", 0, 0, CellValue{Value: "RET ID"}); err != nil {
		t.Fatalf("SetTablixCell: %v", err)
	}
	re := reloadSaved(t, doc)
	for _, tab := range re.ListTablixes() {
		c00 := tab.Rows[0].Cells[0].Value
		if tab.Name == "SalesTablix" && c00 != "Id" {
			t.Errorf("SalesTablix (0,0) changed unexpectedly: %q", c00)
		}
		if tab.Name == "ReturnsTablix" && c00 != "RET ID" {
			t.Errorf("ReturnsTablix (0,0) = %q, want 'RET ID'", c00)
		}
	}
}

// TestMultiTablix_AddColumnToOnlyOne proves that AddTablixColumn modifies only
// the targeted tablix.
func TestMultiTablix_AddColumnToOnlyOne(t *testing.T) {
	doc, err := Load(filepath.Join("testdata", "multi_tablix.rdl"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := doc.AddTablixColumn("SalesTablix", "2cm", -1); err != nil {
		t.Fatalf("AddTablixColumn: %v", err)
	}
	re := reloadSaved(t, doc)
	for _, tab := range re.ListTablixes() {
		want := 2
		if tab.Name == "SalesTablix" {
			want = 3
		}
		if len(tab.Columns) != want {
			t.Errorf("%s columns = %d, want %d", tab.Name, len(tab.Columns), want)
		}
	}
}

// TestMultiTablix_ValidatorPasses makes sure the new fixture is structurally
// clean so the validator has a known-good multi-tablix baseline.
func TestMultiTablix_ValidatorPasses(t *testing.T) {
	doc, err := Load(filepath.Join("testdata", "multi_tablix.rdl"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	r := doc.Validate()
	for _, i := range r.Issues {
		if i.Severity == SeverityError {
			t.Errorf("unexpected error: %+v", i)
		}
	}
	if !r.Pass {
		t.Errorf("multi_tablix fixture should validate cleanly")
	}
}

// TestDryRun_DoesNotWrite confirms every wrapper's dryRun path leaves the
// file untouched. Uses ManageDataSources as a representative case.
func TestDryRun_DoesNotWrite(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.rdl")
	if err := mustLoad(t).Save(src); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(src)

	summary, err := ManageDataSources(src, DataSourceOps{
		Add: []DataSourceAdd{{Name: "Ghost", Provider: "SQL", ConnectString: "x"}},
	}, true)
	if err != nil {
		t.Fatalf("ManageDataSources dryRun: %v", err)
	}
	if !strings.HasPrefix(summary, "[DRY RUN]") {
		t.Errorf("summary should be prefixed with [DRY RUN]; got %q", summary)
	}

	after, _ := os.ReadFile(src)
	if string(before) != string(after) {
		t.Errorf("file was modified despite dryRun=true")
	}

	// Confirm the addition didn't actually happen.
	doc, _ := Load(src)
	for _, ds := range doc.ListDataSources() {
		if ds.Name == "Ghost" {
			t.Errorf("Ghost DataSource was actually added despite dryRun")
		}
	}
}

// TestDryRun_CloneReturnsPlausibleID confirms Clone with dryRun returns an ID
// but doesn't write the target file.
func TestDryRun_CloneReturnsPlausibleID(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.rdl")
	if err := mustLoad(t).Save(src); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(tmp, "cloned.rdl")

	id, err := Clone(src, target, true)
	if err != nil {
		t.Fatalf("Clone dryRun: %v", err)
	}
	if id == "" {
		t.Errorf("Clone returned empty ID in dry-run")
	}
	// Target file must not exist.
	if _, err := os.Stat(target); err == nil {
		t.Errorf("target file was created despite dryRun")
	}
}
