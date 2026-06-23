package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/rdl-toolkit/internal/rdl"
)

func TestSuccessResultJSON(t *testing.T) {
	res, err := successResult("rdl_inspect", "report.rdl", false, map[string]any{"n": 1}, "ok")
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatal("expected success result")
	}
	body := toolResultText(t, res)
	var r Result
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, body)
	}
	if !r.OK || r.Tool != "rdl_inspect" || r.File != "report.rdl" {
		t.Errorf("result: %+v", r)
	}
}

func TestErrorResultJSON(t *testing.T) {
	res, err := errorResult("ARG_MISSING", "file is required", "", map[string]any{"param": "file"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected error result")
	}
	body := toolResultText(t, res)
	var e ErrorResult
	if err := json.Unmarshal([]byte(body), &e); err != nil {
		t.Fatal(err)
	}
	if e.OK || e.Code != "ARG_MISSING" {
		t.Errorf("error result: %+v", e)
	}
}

func TestMapError_AgentError(t *testing.T) {
	ae := rdl.NewNotFoundError("Tablix", "Missing", []string{"T1"})
	res, err := mapError(ae)
	if err != nil {
		t.Fatal(err)
	}
	body := toolResultText(t, res)
	var e ErrorResult
	if err := json.Unmarshal([]byte(body), &e); err != nil {
		t.Fatal(err)
	}
	if e.Code != "NOT_FOUND" {
		t.Errorf("code = %s", e.Code)
	}
}

func TestRequireStringMissing(t *testing.T) {
	req := gomcp.CallToolRequest{
		Params: gomcp.CallToolParams{
			Name:      "rdl_validate",
			Arguments: map[string]any{},
		},
	}
	_, res, _ := requireString(req, "file")
	if res == nil || !res.IsError {
		t.Fatal("expected ARG_MISSING error result")
	}
	body := toolResultText(t, res)
	if !strings.Contains(body, "ARG_MISSING") {
		t.Errorf("body: %s", body)
	}
}

func TestMutationSummary(t *testing.T) {
	s := mutationSummary(rdl.MutationOutcome{Action: "added", Name: "DS1"})
	if !strings.Contains(s, "DS1") {
		t.Errorf("summary: %s", s)
	}
	sk := mutationSummary(rdl.MutationOutcome{Action: "add", Name: "DS1", Skipped: true})
	if !strings.Contains(sk, "Skipped") {
		t.Errorf("skipped summary: %s", sk)
	}
}

func toolResultText(t *testing.T, res *gomcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("empty content")
	}
	tc, ok := gomcp.AsTextContent(res.Content[0])
	if !ok || tc == nil {
		t.Fatalf("not text content: %T", res.Content[0])
	}
	return tc.Text
}
