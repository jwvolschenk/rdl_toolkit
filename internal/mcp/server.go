package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rdl-toolkit/internal/rdl"
)

// NewServer creates and returns the RDL Toolkit MCP server with all tools registered.
func NewServer() *server.MCPServer {
	s := server.NewMCPServer(
		"rdl-toolkit",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	s.AddTools(
		// Read-only inspection (Phase 1)
		inspectTool("rdl_inspect",
			"Get a top-level summary of an RDL report: ReportID, language, page size/orientation, and counts of data sources, data sets, parameters, and tablixes. Use this first to orient yourself before any mutation. Always safe to call.",
			func(doc *rdl.Document) any { return doc.Overview() }),
		inspectTool("rdl_list_datasources",
			"List every <DataSource> in the report with its provider, connect string, security type, and datasource ID. Use before adding or renaming datasources to see what already exists.",
			func(doc *rdl.Document) any { return doc.ListDataSources() }),
		inspectTool("rdl_list_datasets",
			"List every <DataSet> with its bound datasource, command text, fields (name + data field), and filter count. Use before adding/removing fields or updating command text.",
			func(doc *rdl.Document) any { return doc.ListDataSets() }),
		inspectTool("rdl_list_parameters",
			"List every <ReportParameter> with its data type, nullable/allowblank/multivalue/hidden flags, prompt, and default value. Use before adding or removing parameters.",
			func(doc *rdl.Document) any { return doc.ListParameters() }),
		inspectTool("rdl_list_tablixes",
			"List every <Tablix> with its name, bound dataset, column widths, and per-row cell contents (textbox name + value/expression). Use to understand table structure before rebuilding or editing.",
			func(doc *rdl.Document) any { return doc.ListTablixes() }),
		inspectTool("rdl_get_metadata",
			"Get report-level metadata: ReportID, description, language, author, page width/height/orientation, and the four margins. These are the fields that rdl_update_metadata can change.",
			func(doc *rdl.Document) any { return doc.GetMetadata() }),

		// Mutations (existing)
		cloneTool(),
		updateMetadataTool(),
		swapMacrosTool(),
		swapFieldsTool(),
		manageDatasourcesTool(),
		manageDatasetsTool(),
		manageParametersTool(),
		rebuildTablixTool(),
		tablixSetCellTool(),
		tablixAddRowTool(),
		tablixRemoveRowTool(),
		tablixAddColumnTool(),
		tablixRemoveColumnTool(),
		fixEncodingTool(),
		registerTool(),
		validateTool(),
	)

	return s
}

// Serve starts the MCP server on stdio.
func Serve() error {
	return server.ServeStdio(NewServer())
}

// ── Tool definitions ────────────────────────────────────────────────────────

func cloneTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_clone",
		gomcp.WithDescription("Copy an RDL file with a new ReportID. The target file gets a fresh GUID."),
		gomcp.WithString("source",
			gomcp.Required(),
			gomcp.Description("Path to source RDL file"),
		),
		gomcp.WithString("target",
			gomcp.Required(),
			gomcp.Description("Path to target RDL file"),
		),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleClone}
}

func updateMetadataTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_update_metadata",
		gomcp.WithDescription("Update report metadata: description, title, and/or page orientation. Title requires titleTextbox to be set — use rdl_list_tablixes or rdl_get_metadata to find the textbox name. Orientation swaps A4 page dimensions only."),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
		gomcp.WithString("description",
			gomcp.Description("New description text (pipe-delimited conventions preserved)"),
		),
		gomcp.WithString("title",
			gomcp.Description("New title text. Requires titleTextbox to identify which Textbox to update."),
		),
		gomcp.WithString("titleTextbox",
			gomcp.Description("Name attribute of the Textbox to update with the new title. Required when 'title' is set."),
		),
		gomcp.WithString("orientation",
			gomcp.Description("Portrait or Landscape (swaps A4 page width/height)"),
			gomcp.Enum("Portrait", "Landscape"),
		),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleUpdateMetadata}
}

func swapMacrosTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_swap_macros",
		gomcp.WithDescription("Replace strings in ConnectString elements. Use OLD:NEW format for each pair."),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
		gomcp.WithArray("pairs",
			gomcp.Required(),
			gomcp.Description("Replacement pairs in OLD:NEW format (e.g. ['old_macro:new_macro'])"),
			gomcp.WithStringItems(),
		),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleSwapMacros}
}

func swapFieldsTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_swap_fields",
		gomcp.WithDescription("Replace Fields!X.Value references in Value elements. Use OLD:NEW format for each pair."),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
		gomcp.WithArray("pairs",
			gomcp.Required(),
			gomcp.Description("Replacement pairs in OLD:NEW format (e.g. ['OldField:NewField'])"),
			gomcp.WithStringItems(),
		),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleSwapFields}
}

func manageDatasourcesTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_manage_datasources",
		gomcp.WithDescription("Add, remove, rename DataSources, or replace their ConnectStrings. All operations are idempotent: adding an existing name is a no-op; removing/renaming a missing name reports 'not found' in the summary."),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
		gomcp.WithArray("add",
			gomcp.Description("DataSources to add. Each item: {name, provider, connectString, securityType?}. securityType defaults to 'None'."),
		),
		gomcp.WithArray("remove",
			gomcp.Description("DataSource names to remove"),
			gomcp.WithStringItems(),
		),
		gomcp.WithArray("rename",
			gomcp.Description("Renames; each item: {old, new}"),
		),
		gomcp.WithArray("setConnectString",
			gomcp.Description("Replace ConnectString of a named DataSource. Each item: {name, connectString}"),
		),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleManageDatasources}
}

func manageDatasetsTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_manage_datasets",
		gomcp.WithDescription("Add, remove, rename DataSets, edit their fields, clear fields/filters, or update CommandText. Idempotent: adding an existing DataSet name is a no-op. Renaming a DataSet also updates DataSetName references in Tablixes."),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
		gomcp.WithArray("add",
			gomcp.Description("DataSets to add. Each item: {name, datasource, cmdText, fields:[...]}. cmdText may contain colons."),
		),
		gomcp.WithArray("remove",
			gomcp.Description("DataSet names to remove"),
			gomcp.WithStringItems(),
		),
		gomcp.WithArray("rename",
			gomcp.Description("Renames; each item: {old, new}"),
		),
		gomcp.WithArray("addField",
			gomcp.Description("Fields to add. Each item: {dataset, field}"),
		),
		gomcp.WithArray("clearFields",
			gomcp.Description("DataSet names whose <Fields> section should be emptied (tags preserved)"),
			gomcp.WithStringItems(),
		),
		gomcp.WithArray("clearFilters",
			gomcp.Description("DataSet names whose <Filters> section should be removed entirely"),
			gomcp.WithStringItems(),
		),
		gomcp.WithArray("setCommandText",
			gomcp.Description("Replace CommandText. Each item: {dataset, cmdText}"),
		),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleManageDatasets}
}

func manageParametersTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_manage_parameters",
		gomcp.WithDescription("Add or remove ReportParameters. Removing a parameter also strips any ReportParametersLayout CellDefinition referencing it (prevents orphaned-layout SSRS errors). Idempotent: adding an existing parameter is a no-op."),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
		gomcp.WithArray("add",
			gomcp.Description("Parameters to add. Each item: {name, type, default?, prompt?, hidden?}. prompt defaults to name."),
		),
		gomcp.WithArray("remove",
			gomcp.Description("Parameter names to remove"),
			gomcp.WithStringItems(),
		),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleManageParameters}
}

func rebuildTablixTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_rebuild_tablix",
		gomcp.WithDescription("Rebuild a Tablix in an RDL file from a JSON spec. Spec name targets a specific tablix; if empty, the first tablix is rebuilt and keeps its existing name. Colspan is honoured without placeholder cells (correct SSRS semantics)."),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
		gomcp.WithObject("spec",
			gomcp.Required(),
			gomcp.Description(`Tablix spec: {"name":"X","columns":[3.0,5.0],"dataset":"DS","rows":[{"height":"0.5cm","cells":[{"textbox":"T","value":"v","colspan":2,"format":"N2"}]}]}`),
		),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleRebuildTablix}
}

func tablixSetCellTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_tablix_set_cell",
		gomcp.WithDescription("Set the value of a single cell at (row, col) in a named Tablix. Both indexes are 0-based and address the cell position within the row (colspans are not expanded)."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("tablix", gomcp.Required(), gomcp.Description("Tablix Name attribute")),
		gomcp.WithInteger("row", gomcp.Required(), gomcp.Description("Row index, 0-based")),
		gomcp.WithInteger("col", gomcp.Required(), gomcp.Description("Column (cell) index within the row, 0-based")),
		gomcp.WithString("value", gomcp.Required(), gomcp.Description("New cell value. Use Expression=true for VB expressions like =Fields!X.Value")),
		gomcp.WithString("format", gomcp.Description("Optional format string (e.g. 'N2', 'yyyy-MM-dd')")),
		gomcp.WithBoolean("expression", gomcp.Description("Set true to mark value as a VB expression")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleTablixSetCell}
}

func tablixAddRowTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_tablix_add_row",
		gomcp.WithDescription("Append (or insert) a row into a named Tablix. Adds a matching TablixMember to the row hierarchy. Cells may be empty for a row of empty textboxes."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("tablix", gomcp.Required(), gomcp.Description("Tablix Name")),
		gomcp.WithString("height", gomcp.Description("Row height (e.g. '0.5cm'); defaults to 0.5cm")),
		gomcp.WithArray("cells", gomcp.Description(`Row cells: [{"textbox":"X","value":"Y","colspan":1,"format":"N2"}]`)),
		gomcp.WithInteger("index", gomcp.Description("Insert position; -1 or omit to append")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleTablixAddRow}
}

func tablixRemoveRowTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_tablix_remove_row",
		gomcp.WithDescription("Remove a row from a named Tablix by index. Also removes the corresponding TablixMember from the row hierarchy."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("tablix", gomcp.Required(), gomcp.Description("Tablix Name")),
		gomcp.WithInteger("index", gomcp.Required(), gomcp.Description("Row index, 0-based")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleTablixRemoveRow}
}

func tablixAddColumnTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_tablix_add_column",
		gomcp.WithDescription("Append (or insert) a column to a named Tablix. Adds an empty cell to every row and a TablixMember to the column hierarchy. NB: rows with existing ColSpan cells are NOT adjusted — fix up ColSpan values manually if needed."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("tablix", gomcp.Required(), gomcp.Description("Tablix Name")),
		gomcp.WithString("width", gomcp.Description("Column width (e.g. '2.5cm'); defaults to 2.5cm")),
		gomcp.WithInteger("index", gomcp.Description("Insert position; -1 or omit to append")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleTablixAddColumn}
}

func tablixRemoveColumnTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_tablix_remove_column",
		gomcp.WithDescription("Remove a column from a named Tablix by index. Removes the cell at that index from every row. NB: ColSpan values are not adjusted — fix manually if needed."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("tablix", gomcp.Required(), gomcp.Description("Tablix Name")),
		gomcp.WithInteger("index", gomcp.Required(), gomcp.Description("Column index, 0-based")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleTablixRemoveColumn}
}

func fixEncodingTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_fix_encoding",
		gomcp.WithDescription("Ensure an RDL file has UTF-8 BOM and CRLF line endings."),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
	)
	return server.ServerTool{Tool: tool, Handler: handleFixEncoding}
}

func registerTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_register",
		gomcp.WithDescription("Register an RDL file in a .rptproj project file. Adds the report entry and sorts alphabetically."),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
		gomcp.WithString("project",
			gomcp.Required(),
			gomcp.Description("Path to .rptproj file"),
		),
	)
	return server.ServerTool{Tool: tool, Handler: handleRegister}
}

func validateTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_validate",
		gomcp.WithDescription("Validate an RDL file: XML well-formedness (via load), reference integrity (Fields!X.Value → defined Field, Tablix.DataSetName → DataSet, DataSet.DataSourceName → DataSource), and structural checks (Tablix column/cell/hierarchy counts match). Returns a structured report with severity (error/warning), xpath, and message per issue. The Pass flag is false if any errors exist. Heuristics like 'Language looks like a name' or 'ConnectString has raw &' are intentionally NOT included — call this for real defects, not vibes."),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
	)
	return server.ServerTool{Tool: tool, Handler: handleValidate}
}

// ── Argument structs for BindArguments ──────────────────────────────────────

// Each Ops-aware handler binds to a thin wrapper carrying the file path plus
// the rdl Ops struct. Field tags mirror rdl's struct tags exactly so the
// agent's JSON keys map straight through.

type datasourceCall struct {
	File string `json:"file"`
	rdl.DataSourceOps
}

type datasetCall struct {
	File string `json:"file"`
	rdl.DataSetOps
}

type parameterCall struct {
	File string `json:"file"`
	rdl.ParameterOps
}

// ── Handlers ────────────────────────────────────────────────────────────────

func handleClone(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	source, err := request.RequireString("source")
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	target, err := request.RequireString("target")
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}

	newID, err := rdl.Clone(source, target, dryRunFromRequest(request))
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(fmt.Sprintf("Cloned %s -> %s\nNew ReportID: %s", source, target, newID)), nil
}

func handleUpdateMetadata(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, err := request.RequireString("file")
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	spec := rdl.MetadataUpdate{
		Description:  request.GetString("description", ""),
		Title:        request.GetString("title", ""),
		TitleTextbox: request.GetString("titleTextbox", ""),
		Orientation:  request.GetString("orientation", ""),
	}
	if spec.Description == "" && spec.Title == "" && spec.Orientation == "" {
		return gomcp.NewToolResultError("at least one of description, title, or orientation must be provided"), nil
	}
	if spec.Title != "" && spec.TitleTextbox == "" {
		return gomcp.NewToolResultError("title requires titleTextbox — call rdl_list_tablixes or rdl_get_metadata to find the textbox name"), nil
	}

	count, err := rdl.UpdateMetadata(file, spec, dryRunFromRequest(request))
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(fmt.Sprintf("Updated %d metadata field(s) in %s", count, file)), nil
}

func handleSwapMacros(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, err := request.RequireString("file")
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	rawPairs, err := request.RequireStringSlice("pairs")
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	pairs, err := parsePairs(rawPairs)
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	count, err := rdl.SwapMacros(file, pairs, dryRunFromRequest(request))
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(fmt.Sprintf("Replaced %d occurrence(s) in ConnectString elements of %s", count, file)), nil
}

func handleSwapFields(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, err := request.RequireString("file")
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	rawPairs, err := request.RequireStringSlice("pairs")
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	pairs, err := parsePairs(rawPairs)
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	count, err := rdl.SwapFields(file, pairs, dryRunFromRequest(request))
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(fmt.Sprintf("Replaced %d occurrence(s) in Value elements of %s", count, file)), nil
}

func handleManageDatasources(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	var call datasourceCall
	if err := request.BindArguments(&call); err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	if call.File == "" {
		return gomcp.NewToolResultError("file is required"), nil
	}
	summary, err := rdl.ManageDataSources(call.File, call.DataSourceOps, dryRunFromRequest(request))
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(summary), nil
}

func handleManageDatasets(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	var call datasetCall
	if err := request.BindArguments(&call); err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	if call.File == "" {
		return gomcp.NewToolResultError("file is required"), nil
	}
	summary, err := rdl.ManageDataSets(call.File, call.DataSetOps, dryRunFromRequest(request))
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(summary), nil
}

func handleManageParameters(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	var call parameterCall
	if err := request.BindArguments(&call); err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	if call.File == "" {
		return gomcp.NewToolResultError("file is required"), nil
	}
	summary, err := rdl.ManageParameters(call.File, call.ParameterOps, dryRunFromRequest(request))
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(summary), nil
}

func handleRebuildTablix(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, err := request.RequireString("file")
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	specObj := request.GetArguments()["spec"]
	if specObj == nil {
		return gomcp.NewToolResultError("required argument 'spec' not found"), nil
	}
	// Marshal the inbound spec back to bytes, then unmarshal into the typed struct.
	specBytes, err := json.Marshal(specObj)
	if err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("failed to marshal spec: %v", err)), nil
	}
	var spec rdl.TablixSpec
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("invalid spec: %v", err)), nil
	}

	doc, err := rdl.Load(file)
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	summary, err := doc.RebuildTablix(spec)
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	if dryRunFromRequest(request) {
		return gomcp.NewToolResultText("[DRY RUN] " + summary), nil
	}
	if err := doc.Save(file); err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(summary), nil
}

func handleTablixSetCell(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, _ := request.RequireString("file")
	tablix, _ := request.RequireString("tablix")
	row, _ := request.RequireInt("row")
	col, _ := request.RequireInt("col")
	value, _ := request.RequireString("value")
	cv := rdl.CellValue{
		Value:      value,
		Format:     request.GetString("format", ""),
		Expression: request.GetBool("expression", false),
	}
	summary, err := rdl.TablixSetCell(file, tablix, row, col, cv, dryRunFromRequest(request))
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(summary), nil
}

func handleTablixAddRow(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, _ := request.RequireString("file")
	tablix, _ := request.RequireString("tablix")
	height := request.GetString("height", "")
	index := int(request.GetInt("index", -1))

	row := rdl.RowSpec{Height: height}
	if raw, ok := request.GetArguments()["cells"]; ok && raw != nil {
		b, _ := json.Marshal(raw)
		if err := json.Unmarshal(b, &row.Cells); err != nil {
			return gomcp.NewToolResultError(fmt.Sprintf("invalid cells: %v", err)), nil
		}
	}
	summary, err := rdl.TablixAddRow(file, tablix, row, index, dryRunFromRequest(request))
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(summary), nil
}

func handleTablixRemoveRow(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, _ := request.RequireString("file")
	tablix, _ := request.RequireString("tablix")
	index, _ := request.RequireInt("index")
	summary, err := rdl.TablixRemoveRow(file, tablix, index, dryRunFromRequest(request))
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(summary), nil
}

func handleTablixAddColumn(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, _ := request.RequireString("file")
	tablix, _ := request.RequireString("tablix")
	width := request.GetString("width", "")
	index := int(request.GetInt("index", -1))
	summary, err := rdl.TablixAddColumn(file, tablix, width, index, dryRunFromRequest(request))
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(summary), nil
}

func handleTablixRemoveColumn(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, _ := request.RequireString("file")
	tablix, _ := request.RequireString("tablix")
	index, _ := request.RequireInt("index")
	summary, err := rdl.TablixRemoveColumn(file, tablix, index, dryRunFromRequest(request))
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(summary), nil
}

func handleFixEncoding(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, err := request.RequireString("file")
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}

	summary, err := rdl.FixEncoding(file)
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(summary), nil
}

func handleRegister(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, err := request.RequireString("file")
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	project, err := request.RequireString("project")
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}

	summary, err := rdl.Register(file, project)
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(summary), nil
}

func handleValidate(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, err := request.RequireString("file")
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	report, err := rdl.Validate(file)
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	b, _ := json.MarshalIndent(report, "", "  ")
	return gomcp.NewToolResultText(string(b)), nil
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// dryRunFromRequest returns true when the caller wants to preview the mutation
// without writing. Every mutation handler routes through this.
func dryRunFromRequest(request gomcp.CallToolRequest) bool {
	return request.GetBool("dryRun", false)
}

// withDryRunParam is a small wrapper to add the same boolean parameter to
// every mutation tool definition. Saves repeating the description.
func withDryRunParam() gomcp.ToolOption {
	return gomcp.WithBoolean("dryRun",
		gomcp.Description("If true, compute the mutation in memory and return the summary but do NOT write the file. Summary is prefixed with '[DRY RUN] '."))
}

// inspectTool builds a read-only MCP tool that loads an RDL file and returns
// the result of fn as indented JSON. Reduces 6 identical boilerplate blocks
// to one helper call.
func inspectTool(name, description string, fn func(*rdl.Document) any) server.ServerTool {
	tool := gomcp.NewTool(name,
		gomcp.WithDescription(description),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
	)
	handler := func(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		file, err := request.RequireString("file")
		if err != nil {
			return gomcp.NewToolResultError(err.Error()), nil
		}
		doc, err := rdl.Load(file)
		if err != nil {
			return gomcp.NewToolResultError(err.Error()), nil
		}
		b, err := json.MarshalIndent(fn(doc), "", "  ")
		if err != nil {
			return gomcp.NewToolResultError(fmt.Sprintf("marshalling result: %v", err)), nil
		}
		return gomcp.NewToolResultText(string(b)), nil
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func parsePairs(items []string) ([]rdl.RenamePair, error) {
	pairs := make([]rdl.RenamePair, 0, len(items))
	for _, item := range items {
		parts := strings.SplitN(item, ":", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("invalid pair format %q, expected OLD:NEW", item)
		}
		pairs = append(pairs, rdl.RenamePair{Old: parts[0], New: parts[1]})
	}
	return pairs, nil
}
