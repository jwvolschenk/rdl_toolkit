package rdl

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
)

// TestValidate_CleanFixture runs every check on the sample fixture and expects
// zero errors. Warnings are allowed but currently none are emitted.
func TestValidate_CleanFixture(t *testing.T) {
	doc := mustLoad(t)
	r := doc.Validate()
	for _, i := range r.Issues {
		if i.Severity == SeverityError {
			t.Errorf("unexpected error on clean fixture: %+v", i)
		}
	}
	if !r.Pass {
		t.Errorf("Pass = false; want true. Issues: %+v", r.Issues)
	}
}

// TestValidate_BrokenFieldReference introduces a Fields!X.Value where X is not
// defined anywhere. Validator must flag it.
func TestValidate_BrokenFieldReference(t *testing.T) {
	doc := mustLoad(t)
	// Append a Textbox containing a bogus field reference inside the PageHeader.
	ph := xmlquery.FindOne(doc.Root(), "//PageHeader/ReportItems")
	if ph == nil {
		t.Fatal("fixture missing PageHeader/ReportItems")
	}
	xmlquery.AddChild(ph, mustParseFragment(t,
		`<Textbox Name="Bogus"><Paragraphs><Paragraph><TextRuns><TextRun><Value>=Fields!DoesNotExist.Value</Value></TextRun></TextRuns></Paragraph></Paragraphs></Textbox>`))

	r := doc.Validate()
	expectIssue(t, r, "Fields!DoesNotExist.Value")
}

// TestValidate_BrokenDatasetReference makes a Tablix point at a missing DataSet.
func TestValidate_BrokenDatasetReference(t *testing.T) {
	doc := mustLoad(t)
	// Change the SalesTable Tablix's DataSetName to Ghost.
	dn := xmlquery.FindOne(doc.Root(), "//Tablix/DataSetName")
	if dn == nil {
		t.Fatal("fixture missing Tablix/DataSetName")
	}
	setNodeText(dn, "Ghost")

	r := doc.Validate()
	expectIssue(t, r, "references DataSet \"Ghost\"")
}

// TestValidate_BrokenDataSourceReference makes a DataSet's Query point at a
// missing DataSource.
func TestValidate_BrokenDataSourceReference(t *testing.T) {
	doc := mustLoad(t)
	dsn := xmlquery.FindOne(doc.Root(), "//DataSet[@Name='Sales']/Query/DataSourceName")
	if dsn == nil {
		t.Fatal("fixture missing Sales/Query/DataSourceName")
	}
	setNodeText(dsn, "GhostSource")

	r := doc.Validate()
	expectIssue(t, r, "references DataSource \"GhostSource\"")
}

// TestValidate_DuplicateDataSource clones Source1 inside DataSources.
func TestValidate_DuplicateDataSource(t *testing.T) {
	doc := mustLoad(t)
	src := xmlquery.FindOne(doc.Root(), "//DataSource[@Name='Source1']")
	if src == nil {
		t.Fatal("fixture missing DataSource Source1")
	}
	// Add a second Source1 with a fresh DataSourceID to avoid other checks tripping.
	dup := mustParseFragment(t,
		`<DataSource Name="Source1"><ConnectionProperties><DataProvider>SQL</DataProvider><ConnectString>x</ConnectString></ConnectionProperties></DataSource>`)
	xmlquery.AddSibling(src, dup)

	r := doc.Validate()
	expectIssue(t, r, "DataSource \"Source1\" defined 2 times")
}

// TestValidate_BrokenTablixColumns messes with the column/cell alignment.
func TestValidate_BrokenTablixColumns(t *testing.T) {
	doc := mustLoad(t)
	// Remove the first TablixMember from column hierarchy. Now cols(3) != members(2).
	member := xmlquery.FindOne(doc.Root(), "//TablixColumnHierarchy/TablixMembers/TablixMember")
	if member == nil {
		t.Fatal("fixture missing column hierarchy member")
	}
	xmlquery.RemoveFromTree(member)

	r := doc.Validate()
	expectIssue(t, r, "TablixColumnHierarchy member count")
}

// TestValidate_ParameterLayoutOrphan verifies removing a ReportParameter while
// leaving its layout cell in place triggers an error.
func TestValidate_ParameterLayoutOrphan(t *testing.T) {
	doc := mustLoad(t)
	p := xmlquery.FindOne(doc.Root(), "//ReportParameter[@Name='StartDate']")
	if p == nil {
		t.Fatal("fixture missing StartDate parameter")
	}
	xmlquery.RemoveFromTree(p)

	r := doc.Validate()
	expectIssue(t, r, "\"StartDate\"")
}

// TestValidate_TextboxMissingParagraphs flags bare Textbox shells that Visual
// Studio cannot deserialize.
func TestValidate_TextboxMissingParagraphs(t *testing.T) {
	doc := mustLoad(t)
	tb := xmlquery.FindOne(doc.Root(), "//Tablix//Textbox")
	if tb == nil {
		t.Fatal("fixture missing tablix textbox")
	}
	paragraphs := child(tb, "Paragraphs")
	if paragraphs == nil {
		t.Fatal("fixture textbox already missing Paragraphs")
	}
	xmlquery.RemoveFromTree(paragraphs)

	r := doc.Validate()
	expectIssue(t, r, "missing mandatory <Paragraphs>")
}

// TestValidate_MalformedXML is a top-level test: Validate(path) should return
// a non-nil error when the file is not well-formed XML.
func TestValidate_MalformedXML(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "bad.rdl")
	if err := writeFileBytes(t, p, []byte("\xef\xbb\xbf<?xml version=\"1.0\"?>\n<Report><Unclosed></NotMatched>")); err != nil {
		t.Fatal(err)
	}
	_, err := Validate(p)
	if err == nil {
		t.Errorf("expected error for malformed XML")
	}
}

// TestValidate_JSONRoundTrip confirms the report JSON-serialises cleanly.
func TestValidate_JSONRoundTrip(t *testing.T) {
	doc := mustLoad(t)
	r := doc.Validate()
	// Inject a fake issue to ensure slice is non-empty in JSON.
	r.Issues = append(r.Issues, Issue{Severity: SeverityWarning, Message: "test"})
	out, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), "\"pass\":") {
		t.Errorf("JSON missing 'pass' field: %s", out)
	}
	if !strings.Contains(string(out), "\"severity\":\"warning\"") {
		t.Errorf("JSON missing severity: %s", out)
	}
}

// ── helpers ────────────────────────────────────────────────────────────────

func expectIssue(t *testing.T, r *ValidationReport, needle string) {
	t.Helper()
	for _, i := range r.Issues {
		if strings.Contains(i.Message, needle) {
			return
		}
	}
	t.Errorf("expected an issue containing %q; got:", needle)
	for _, i := range r.Issues {
		t.Errorf("  [%s] %s", i.Severity, i.Message)
	}
}

func mustParseFragment(t *testing.T, xmlStr string) *xmlquery.Node {
	t.Helper()
	n, err := parseFragment(xmlStr)
	if err != nil {
		t.Fatalf("parseFragment: %v", err)
	}
	return n
}
