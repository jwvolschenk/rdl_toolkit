package rdl

import (
	"testing"
)

func TestNormalizeCellValue_Expression(t *testing.T) {
	v, err := normalizeCellValue(CellValue{Value: "Fields!X.Value", Expression: true})
	if err != nil {
		t.Fatal(err)
	}
	if v != "=Fields!X.Value" {
		t.Errorf("got %q, want leading =", v)
	}

	v, err = normalizeCellValue(CellValue{Value: "=Fields!X.Value", Expression: true})
	if err != nil || v != "=Fields!X.Value" {
		t.Errorf("double expression: %q err=%v", v, err)
	}

	_, err = normalizeCellValue(CellValue{Value: "=Fields!X.Value", Expression: false})
	if err == nil {
		t.Error("expected error when value has = but expression is false")
	}
}

func TestAddDataSourceOp_Idempotent(t *testing.T) {
	doc := mustLoad(t)
	out, err := doc.AddDataSourceOp(DataSourceAdd{
		Name: "NewDS", Provider: "SQL", ConnectString: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Skipped || out.Action != "added" {
		t.Errorf("first add: %+v", out)
	}

	out2, err := doc.AddDataSourceOp(DataSourceAdd{
		Name: "NewDS", Provider: "SQL", ConnectString: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out2.Skipped {
		t.Errorf("second add should skip: %+v", out2)
	}
}

func TestRemoveDataSourceOp_NotFound(t *testing.T) {
	doc := mustLoad(t)
	_, err := doc.RemoveDataSourceOp("NoSuchDS")
	if err == nil {
		t.Fatal("expected NOT_FOUND")
	}
	ae, ok := AsAgentError(err)
	if !ok || ae.Code != CodeNotFound {
		t.Errorf("got %v", err)
	}
}
