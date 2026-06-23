package rdl

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
)

func TestApplyTheme_CopiesHeaderTheme(t *testing.T) {
	// Create a source report with HeaderTheme dataset.
	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "source.rdl")
	spec := CreateSpec{Target: srcPath, Title: "Source"}
	if _, err := Create(spec, false); err != nil {
		t.Fatalf("Create source: %v", err)
	}

	// Add a HeaderTheme dataset to the source manually.
	srcDoc, err := Load(srcPath)
	if err != nil {
		t.Fatalf("Load source: %v", err)
	}
	// Use ManageParameters to add pvc_Theme first (HeaderTheme needs it).
	srcDoc.ManageParameters(ParameterOps{
		Add: []ParameterAdd{
			{Name: "pvc_Theme", Type: "String", Prompt: "Theme"},
			{Name: "vc_ReportPack", Type: "String", Hidden: true},
		},
	})
	// Add a minimal HeaderTheme dataset.
	srcDS := xmlquery.FindOne(srcDoc.root, "//DataSets")
	if srcDS == nil {
		t.Fatal("source has no DataSets")
	}
	htDS := createElement("DataSet", [2]string{"Name", "HeaderTheme"})
	sds := createElement("SharedDataSet")
	sdsRef := elementWithText("SharedDataSetReference", "HeaderTheme")
	xmlquery.AddChild(sds, sdsRef)
	qps := createElement("QueryParameters")
	qp := createElement("QueryParameter", [2]string{"Name", "@pvc_theme"})
	xmlquery.AddChild(qp, elementWithText("Value", "=Parameters!pvc_Theme.Value"))
	xmlquery.AddChild(qps, qp)
	xmlquery.AddChild(sds, qps)
	xmlquery.AddChild(htDS, sds)
	fields := createElement("Fields")
	f := createElement("Field", [2]string{"Name", "Header_Name1_FontSize"})
	xmlquery.AddChild(f, elementWithText("DataField", "Header_Name1_FontSize"))
	xmlquery.AddChild(fields, f)
	xmlquery.AddChild(htDS, fields)
	appendIndented(srcDS, htDS, depthOf(srcDS)+1)
	if err := srcDoc.Save(srcPath); err != nil {
		t.Fatalf("save source: %v", err)
	}

	// Create a target report with no HeaderTheme.
	tgtPath := filepath.Join(t.TempDir(), "target.rdl")
	tgtSpec := CreateSpec{Target: tgtPath, Title: "Target"}
	if _, err := Create(tgtSpec, false); err != nil {
		t.Fatalf("Create target: %v", err)
	}

	// Apply theme.
	summary, err := ApplyTheme(srcPath, tgtPath, false)
	if err != nil {
		t.Fatalf("ApplyTheme: %v", err)
	}
	if !strings.Contains(summary, "HeaderTheme dataset") {
		t.Errorf("summary missing HeaderTheme: %s", summary)
	}

	// Verify target has HeaderTheme dataset.
	tgtDoc, err := Load(tgtPath)
	if err != nil {
		t.Fatalf("Load target: %v", err)
	}
	ht := xmlquery.FindOne(tgtDoc.root, "//DataSet[@Name='HeaderTheme']")
	if ht == nil {
		t.Fatal("target missing HeaderTheme dataset after ApplyTheme")
	}

	// Verify target has pvc_Theme parameter.
	hasTheme := false
	for _, p := range tgtDoc.ListParameters() {
		if p.Name == "pvc_Theme" {
			hasTheme = true
		}
	}
	if !hasTheme {
		t.Error("target missing pvc_Theme parameter after ApplyTheme")
	}
}

func TestApplyTheme_DryRun(t *testing.T) {
	srcPath := filepath.Join(t.TempDir(), "src.rdl")
	tgtPath := filepath.Join(t.TempDir(), "tgt.rdl")
	Create(CreateSpec{Target: srcPath, Title: "S"}, false)
	Create(CreateSpec{Target: tgtPath, Title: "T"}, false)

	summary, err := ApplyTheme(srcPath, tgtPath, true)
	if err != nil {
		t.Fatalf("ApplyTheme dry-run: %v", err)
	}
	// Source has no HeaderTheme but has page dimensions (from Create skeleton).
	if !strings.Contains(summary, "Applied theme") {
		t.Errorf("expected 'Applied theme', got: %s", summary)
	}
}

func TestApplyTheme_ReplacesExistingHeaderTheme(t *testing.T) {
	// Create source with HeaderTheme.
	srcPath := filepath.Join(t.TempDir(), "src.rdl")
	Create(CreateSpec{Target: srcPath, Title: "S"}, false)
	srcDoc, _ := Load(srcPath)
	srcDS := xmlquery.FindOne(srcDoc.root, "//DataSets")
	htDS := createElement("DataSet", [2]string{"Name", "HeaderTheme"})
	sds := createElement("SharedDataSet")
	xmlquery.AddChild(sds, elementWithText("SharedDataSetReference", "HeaderTheme"))
	xmlquery.AddChild(htDS, sds)
	appendIndented(srcDS, htDS, depthOf(srcDS)+1)
	srcDoc.Save(srcPath)

	// Create target that already has a HeaderTheme.
	tgtPath := filepath.Join(t.TempDir(), "tgt.rdl")
	Create(CreateSpec{Target: tgtPath, Title: "T"}, false)
	tgtDoc, _ := Load(tgtPath)
	tgtDS := xmlquery.FindOne(tgtDoc.root, "//DataSets")
	oldDS := createElement("DataSet", [2]string{"Name", "HeaderTheme"})
	oldSds := createElement("SharedDataSet")
	xmlquery.AddChild(oldSds, elementWithText("SharedDataSetReference", "OldTheme"))
	xmlquery.AddChild(oldDS, oldSds)
	appendIndented(tgtDS, oldDS, depthOf(tgtDS)+1)
	tgtDoc.Save(tgtPath)

	// Apply theme.
	_, err := ApplyTheme(srcPath, tgtPath, false)
	if err != nil {
		t.Fatalf("ApplyTheme: %v", err)
	}

	// Verify the old theme was replaced.
	tgtDoc2, _ := Load(tgtPath)
	refs := xmlquery.Find(tgtDoc2.root, "//SharedDataSetReference")
	for _, ref := range refs {
		if strings.TrimSpace(ref.InnerText()) == "OldTheme" {
			t.Error("old HeaderTheme still present after ApplyTheme")
		}
	}
}
