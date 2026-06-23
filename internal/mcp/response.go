package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/rdl-toolkit/internal/rdl"
)

// Result is the success envelope returned by every MCP tool.
type Result struct {
	OK      bool            `json:"ok"`
	Tool    string          `json:"tool"`
	File    string          `json:"file,omitempty"`
	DryRun  bool            `json:"dryRun,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Summary string          `json:"summary,omitempty"`
}

// ErrorResult is the error envelope returned when a tool fails.
type ErrorResult struct {
	OK      bool           `json:"ok"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Hint    string         `json:"hint,omitempty"`
	Context map[string]any `json:"context,omitempty"`
}

func successResult(tool, file string, dryRun bool, data any, summary string) (*gomcp.CallToolResult, error) {
	r := Result{OK: true, Tool: tool, File: file, DryRun: dryRun, Summary: summary}
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return errorResult("INTERNAL", fmt.Sprintf("marshalling result: %v", err), "", nil)
		}
		r.Data = b
	}
	return resultJSON(r)
}

func successFromOutcome(tool, file string, dryRun bool, out rdl.MutationOutcome) (*gomcp.CallToolResult, error) {
	summary := mutationSummary(out)
	return successResult(tool, file, dryRun, out, summary)
}

func mutationSummary(out rdl.MutationOutcome) string {
	if out.Skipped {
		return fmt.Sprintf("Skipped (already exists or no-op): %s", out.Name)
	}
	switch out.Action {
	case "added":
		return fmt.Sprintf("Added %s", out.Name)
	case "removed":
		if out.Count > 0 {
			return fmt.Sprintf("Removed %s (+ %d layout cell(s))", out.Name, out.Count)
		}
		return fmt.Sprintf("Removed %s", out.Name)
	case "renamed":
		return fmt.Sprintf("Renamed %s -> %s", out.OldName, out.NewName)
	case "updated", "command_updated":
		return fmt.Sprintf("Updated %s", out.Name)
	case "field_added":
		return fmt.Sprintf("Added field %s to DataSet %s", out.Name, out.NewName)
	case "fields_cleared":
		return fmt.Sprintf("Cleared %d field(s) from DataSet %s", out.Count, out.Name)
	case "filters_cleared":
		return fmt.Sprintf("Cleared filters from DataSet %s", out.Name)
	default:
		return out.Action
	}
}

func resultJSON(r Result) (*gomcp.CallToolResult, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return errorResult("INTERNAL", fmt.Sprintf("marshalling result: %v", err), "", nil)
	}
	return gomcp.NewToolResultText(string(b)), nil
}

func errorResult(code, message, hint string, ctx map[string]any) (*gomcp.CallToolResult, error) {
	e := ErrorResult{OK: false, Code: code, Message: message, Hint: hint, Context: ctx}
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return gomcp.NewToolResultError(message), nil
	}
	return gomcp.NewToolResultError(string(b)), nil
}

func mapError(err error) (*gomcp.CallToolResult, error) {
	if err == nil {
		return errorResult("INTERNAL", "unknown error", "", nil)
	}
	var ae *rdl.AgentError
	if errors.As(err, &ae) {
		return errorResult(string(ae.Code), ae.Message, ae.Hint, ae.Context)
	}
	if errors.Is(err, os.ErrNotExist) {
		return errorResult("FILE_NOT_FOUND", err.Error(), "", nil)
	}
	return errorResult("IO_ERROR", err.Error(), "", nil)
}

func requireString(request gomcp.CallToolRequest, param string) (string, *gomcp.CallToolResult, error) {
	v, err := request.RequireString(param)
	if err != nil {
		res, _ := errorResult("ARG_MISSING", fmt.Sprintf("%s is required", param), "", map[string]any{"param": param})
		return "", res, err
	}
	return v, nil, nil
}

func requireInt(request gomcp.CallToolRequest, param string) (int, *gomcp.CallToolResult, error) {
	v, err := request.RequireInt(param)
	if err != nil {
		res, _ := errorResult("ARG_MISSING", fmt.Sprintf("%s is required", param), "", map[string]any{"param": param})
		return 0, res, err
	}
	return int(v), nil, nil
}

const serverInstructions = `RDL Toolkit MCP workflow for SSRS report editing:
1. Call rdl_inspect, then the relevant rdl_list_* tool before any mutation.
2. Run mutations with dryRun:true first to preview.
3. Call rdl_validate and check data.pass before applying.
4. Re-run mutations with dryRun:false to write.
5. For tablix edits, call rdl_list_tablixes first. Use rdl_rebuild_tablix for full restructure; use rdl_tablix_* for small edits.
6. For VB expressions in cells, include leading '=' in value OR set expression:true.
7. Each datasource/dataset/parameter operation is a separate atomic tool (rdl_add_*, rdl_remove_*, etc.).`
