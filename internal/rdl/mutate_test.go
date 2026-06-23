package rdl

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
)

// saveToTmp saves doc to a fresh temp file and returns the path.
func saveToTmp(t *testing.T, doc *Document) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "out.rdl")
	if err := doc.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}
	return p
}

// reloadSaved saves doc and reloads it, so we test the persisted state
// (catches cases where in-memory mutations don't survive serialization).
func reloadSaved(t *testing.T, doc *Document) *Document {
	t.Helper()
	p := saveToTmp(t, doc)
	re, err := Load(p)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	return re
}

// ── Clone / SetReportID ────────────────────────────────────────────────────

func TestSetReportID(t *testing.T) {
	doc := mustLoad(t)
	orig := doc.ReportID()
	doc.SetReportID("deadbeef-dead-dead-dead-deadbeefdead")
	if doc.ReportID() != "deadbeef-dead-dead-dead-deadbeefdead" {
		t.Errorf("ReportID = %q", doc.ReportID())
	}
	if doc.ReportID() == orig {
		t.Errorf("ReportID unchanged")
	}
}

func TestClone(t *testing.T) {
	tmp := t.TempDir()
	dst := filepath.Join(tmp, "cloned.rdl")
	id, err := Clone(filepath.Join("testdata", "sample.rdl"), dst, false)
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if id == "" {
		t.Fatalf("Clone returned empty ID")
	}
	clone, err := Load(dst)
	if err != nil {
		t.Fatalf("Load clone: %v", err)
	}
	if clone.ReportID() != id {
		t.Errorf("clone ReportID = %q, want %q", clone.ReportID(), id)
	}
	orig, _ := Load(filepath.Join("testdata", "sample.rdl"))
	if clone.ReportID() == orig.ReportID() {
		t.Errorf("clone has same ReportID as source")
	}
}

// ── Metadata ────────────────────────────────────────────────────────────────

func TestUpdateMetadata_Description(t *testing.T) {
	doc := mustLoad(t)
	n, err := doc.UpdateMetadata(MetadataUpdate{Description: "New|Description"})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("changed count = %d, want 1", n)
	}
	got := reloadSaved(t, doc).GetMetadata().Description
	if got != "New|Description" {
		t.Errorf("Description = %q", got)
	}
}

func TestUpdateMetadata_TitleByTextbox(t *testing.T) {
	doc := mustLoad(t)
	n, err := doc.UpdateMetadata(MetadataUpdate{Title: "New Sales Report", TitleTextbox: "ReportTitle"})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("changed count = %d, want 1", n)
	}
	re := reloadSaved(t, doc)
	tb := xmlquery.FindOne(re.Root(), "//PageHeader//Textbox[@Name='ReportTitle']")
	if tb == nil {
		t.Fatalf("ReportTitle textbox missing after save")
	}
	val := xmlquery.FindOne(tb, ".//Value")
	if val == nil {
		t.Fatal("ReportTitle Value element missing")
	}
	v := strings.TrimSpace(val.InnerText())
	if v != "New Sales Report" {
		t.Errorf("ReportTitle value = %q, want 'New Sales Report'", v)
	}
}

func TestUpdateMetadata_OrientationSwap(t *testing.T) {
	doc := mustLoad(t)
	if doc.GetMetadata().Orientation != "Portrait" {
		t.Fatalf("fixture should start Portrait; got %q", doc.GetMetadata().Orientation)
	}
	if _, err := doc.UpdateMetadata(MetadataUpdate{Orientation: "Landscape"}); err != nil {
		t.Fatal(err)
	}
	re := reloadSaved(t, doc)
	if re.GetMetadata().PageWidth != "29.7cm" || re.GetMetadata().PageHeight != "21cm" {
		t.Errorf("Landscape dimensions wrong: w=%q h=%q",
			re.GetMetadata().PageWidth, re.GetMetadata().PageHeight)
	}
}

// ── DataSources ─────────────────────────────────────────────────────────────

func TestManageDataSources_AddRemove(t *testing.T) {
	doc := mustLoad(t)
	startCount := len(doc.ListDataSources())
	summary := doc.ManageDataSources(DataSourceOps{
		Add: []DataSourceAdd{{
			Name: "NewDS", Provider: "SQL",
			ConnectString: "Data Source=server;Initial Catalog=db",
		}},
	})
	if !strings.Contains(summary, "Added DataSource 'NewDS'") {
		t.Errorf("summary missing add: %q", summary)
	}
	re := reloadSaved(t, doc)
	if got := len(re.ListDataSources()); got != startCount+1 {
		t.Errorf("after add: %d datasources, want %d", got, startCount+1)
	}

	// Idempotency: adding again should skip.
	re2 := reloadSaved(t, doc)
	summary2 := re2.ManageDataSources(DataSourceOps{
		Add: []DataSourceAdd{{Name: "NewDS", Provider: "SQL", ConnectString: "x"}},
	})
	if !strings.Contains(summary2, "already exists") {
		t.Errorf("idempotency: expected 'already exists' in %q", summary2)
	}

	// Remove it.
	summary3 := re.ManageDataSources(DataSourceOps{Remove: []string{"NewDS"}})
	if !strings.Contains(summary3, "Removed DataSource 'NewDS'") {
		t.Errorf("summary missing remove: %q", summary3)
	}
	re3 := reloadSaved(t, re)
	if got := len(re3.ListDataSources()); got != startCount {
		t.Errorf("after remove: %d datasources, want %d", got, startCount)
	}
}

func TestManageDataSources_Rename(t *testing.T) {
	doc := mustLoad(t)
	doc.ManageDataSources(DataSourceOps{
		Rename: []RenamePair{{Old: "Source1", New: "PrimarySource"}},
	})
	re := reloadSaved(t, doc)
	names := []string{}
	for _, ds := range re.ListDataSources() {
		names = append(names, ds.Name)
	}
	if !contains(names, "PrimarySource") || contains(names, "Source1") {
		t.Errorf("rename failed; got %v", names)
	}
	// The Sales dataset should reference the new name.
	sales := re.ListDataSets()[0]
	if sales.DataSource != "PrimarySource" {
		t.Errorf("DataSet.DataSource = %q, expected renamed PrimarySource", sales.DataSource)
	}
}

func TestManageDataSources_SecurityType(t *testing.T) {
	doc := mustLoad(t)
	doc.ManageDataSources(DataSourceOps{
		Add: []DataSourceAdd{{
			Name: "SecDS", Provider: "SQL",
			ConnectString: "Data Source=s", SecurityType: "Integrated",
		}},
	})
	re := reloadSaved(t, doc)
	for _, ds := range re.ListDataSources() {
		if ds.Name == "SecDS" && ds.SecurityType != "Integrated" {
			t.Errorf("SecDS SecurityType = %q, want Integrated", ds.SecurityType)
		}
	}
}

// ── DataSets ────────────────────────────────────────────────────────────────

func TestManageDataSets_AddWithColonInCmdText(t *testing.T) {
	// This is the regression test for the bug that motivated Phase 2:
	// SQL CommandText containing colons must survive intact.
	doc := mustLoad(t)
	const sql = "SELECT TOP 1:1 Id FROM t WHERE x:y"
	doc.ManageDataSets(DataSetOps{
		Add: []DataSetAdd{{
			Name: "ColonDS", DataSource: "Source1", CmdText: sql, Fields: []string{"Id"},
		}},
	})
	re := reloadSaved(t, doc)
	for _, ds := range re.ListDataSets() {
		if ds.Name == "ColonDS" {
			if ds.CommandText != sql {
				t.Errorf("CommandText corrupted: got %q, want %q", ds.CommandText, sql)
			}
			return
		}
	}
	t.Errorf("ColonDS not found after add")
}

func TestManageDataSets_AddFieldClearFilters(t *testing.T) {
	doc := mustLoad(t)
	// Regions has 1 filter; clear it.
	doc.ManageDataSets(DataSetOps{
		ClearFilters: []string{"Regions"},
		AddField:     []FieldAdd{{DataSet: "Regions", Field: "NewField"}},
	})
	re := reloadSaved(t, doc)
	for _, ds := range re.ListDataSets() {
		if ds.Name == "Regions" {
			if ds.FilterCount != 0 {
				t.Errorf("Regions FilterCount = %d, want 0", ds.FilterCount)
			}
			found := false
			for _, f := range ds.Fields {
				if f.Name == "NewField" {
					found = true
				}
			}
			if !found {
				t.Errorf("NewField not added; fields = %+v", ds.Fields)
			}
		}
	}
}

func TestManageDataSets_UpdateCommandText(t *testing.T) {
	doc := mustLoad(t)
	const newSQL = "SELECT NEW_FIELD FROM Sales WHERE 1=1"
	doc.ManageDataSets(DataSetOps{
		SetCommandText: []CmdTextUpdate{{DataSet: "Sales", CmdText: newSQL}},
	})
	re := reloadSaved(t, doc)
	if re.ListDataSets()[0].CommandText != newSQL {
		t.Errorf("CommandText = %q, want %q", re.ListDataSets()[0].CommandText, newSQL)
	}
}

// ── Parameters ──────────────────────────────────────────────────────────────

func TestManageParameters_AddRemove(t *testing.T) {
	doc := mustLoad(t)
	start := len(doc.ListParameters())
	doc.ManageParameters(ParameterOps{
		Add: []ParameterAdd{{Name: "Period", Type: "String", Prompt: "Select period"}},
	})
	re := reloadSaved(t, doc)
	if got := len(re.ListParameters()); got != start+1 {
		t.Errorf("after add: %d params, want %d", got, start+1)
	}
	// Verify prompt is set as provided.
	for _, p := range re.ListParameters() {
		if p.Name == "Period" && p.Prompt != "Select period" {
			t.Errorf("Period prompt = %q, want 'Select period'", p.Prompt)
		}
	}

	// Idempotency.
	re2 := reloadSaved(t, doc)
	summary := re2.ManageParameters(ParameterOps{
		Add: []ParameterAdd{{Name: "Period", Type: "String"}},
	})
	if !strings.Contains(summary, "already exists") {
		t.Errorf("idempotency: %q", summary)
	}

	// Remove also strips layout cells.
	re.ManageParameters(ParameterOps{Remove: []string{"Region"}})
	re3 := reloadSaved(t, re)
	for _, p := range re3.ListParameters() {
		if p.Name == "Region" {
			t.Errorf("Region still present after remove")
		}
	}
}

// contains is a tiny local helper.
func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
