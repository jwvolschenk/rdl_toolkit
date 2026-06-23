package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rdl-toolkit/internal/rdl"
)

// NewServer creates and returns the RDL Toolkit MCP server with all tools registered.
func NewServer() *server.MCPServer {
	s := server.NewMCPServer(
		"rdl-toolkit",
		"2.0.0",
		server.WithToolCapabilities(false),
	)

	s.AddTools(
		inspectTool("rdl_inspect",
			serverInstructions+" Top-level summary: ReportID, language, page size/orientation, counts. Call first.",
			func(doc *rdl.Document) any { return doc.Overview() }),
		inspectTool("rdl_list_datasources",
			"List every DataSource with provider, connect string, security type, and ID.",
			func(doc *rdl.Document) any { return doc.ListDataSources() }),
		inspectTool("rdl_list_datasets",
			"List every DataSet with bound datasource, command text, fields, and filter count.",
			func(doc *rdl.Document) any { return doc.ListDataSets() }),
		inspectTool("rdl_list_parameters",
			"List every ReportParameter with type, flags, prompt, and default value.",
			func(doc *rdl.Document) any { return doc.ListParameters() }),
		inspectTool("rdl_list_tablixes",
			"List every Tablix with name, dataset, columns, and per-cell textbox + value. Call before tablix edits.",
			func(doc *rdl.Document) any { return doc.ListTablixes() }),
		inspectTool("rdl_get_metadata",
			"Report metadata: description, language, author, page size, margins.",
			func(doc *rdl.Document) any { return doc.GetMetadata() }),

		createTool(),
		cloneTool(),
		updateMetadataTool(),
		swapMacrosTool(),
		swapFieldsTool(),

		addDataSourceTool(),
		removeDataSourceTool(),
		renameDataSourceTool(),
		setDataSourceConnectStringTool(),

		addDataSetTool(),
		removeDataSetTool(),
		renameDataSetTool(),
		addDataSetFieldTool(),
		clearDataSetFieldsTool(),
		clearDataSetFiltersTool(),
		setDataSetCommandTextTool(),

		addParameterTool(),
		removeParameterTool(),

		rebuildTablixTool(),
		tablixSetCellTool(),
		tablixAddRowTool(),
		tablixRemoveRowTool(),
		tablixAddColumnTool(),
		tablixRemoveColumnTool(),
		applyThemeTool(),
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

func createTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_create",
		gomcp.WithDescription("Create a new RDL file from scratch with minimal skeleton. No inherited baggage. Use rdl_add_datasource, rdl_add_dataset, rdl_rebuild_tablix to build up the report."),
		gomcp.WithString("target", gomcp.Required(), gomcp.Description("Path to new RDL file")),
		gomcp.WithString("title", gomcp.Required(), gomcp.Description("Report title")),
		gomcp.WithString("orientation", gomcp.Description("Portrait (default) or Landscape"), gomcp.Enum("Portrait", "Landscape")),
		gomcp.WithString("description", gomcp.Description("Pipe-delimited metadata (Section|Portfolio|Group|...)")),
		gomcp.WithString("author", gomcp.Description("Author (default: Credo)")),
		gomcp.WithString("fontFamily", gomcp.Description("Default font family (default: Segoe UI)")),
		gomcp.WithString("pageWidth", gomcp.Description("Page width (default: auto from orientation)")),
		gomcp.WithString("pageHeight", gomcp.Description("Page height (default: auto from orientation)")),
		gomcp.WithString("leftMargin", gomcp.Description("Left margin (default: 1cm)")),
		gomcp.WithString("rightMargin", gomcp.Description("Right margin (default: 1cm)")),
		gomcp.WithString("topMargin", gomcp.Description("Top margin (default: 1cm)")),
		gomcp.WithString("bottomMargin", gomcp.Description("Bottom margin (default: 1cm)")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleCreate}
}

func cloneTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_clone",
		gomcp.WithDescription("Copy an RDL file with a new ReportID. Target gets a fresh GUID."),
		gomcp.WithString("source", gomcp.Required(), gomcp.Description("Path to source RDL file")),
		gomcp.WithString("target", gomcp.Required(), gomcp.Description("Path to target RDL file")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleClone}
}

func updateMetadataTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_update_metadata",
		gomcp.WithDescription("Update report metadata: description, title, and/or page orientation. Title requires titleTextbox."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("description", gomcp.Description("New description text")),
		gomcp.WithString("title", gomcp.Description("New title text (requires titleTextbox)")),
		gomcp.WithString("titleTextbox", gomcp.Description("Textbox Name to update when setting title")),
		gomcp.WithString("orientation", gomcp.Description("Portrait or Landscape"), gomcp.Enum("Portrait", "Landscape")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleUpdateMetadata}
}

func swapMacrosTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_swap_macros",
		gomcp.WithDescription("Replace one string in all ConnectString elements. One replacement per call."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("old", gomcp.Required(), gomcp.Description("String to find")),
		gomcp.WithString("new", gomcp.Required(), gomcp.Description("Replacement string")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleSwapMacros}
}

func swapFieldsTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_swap_fields",
		gomcp.WithDescription("Replace one Fields!X.Value reference in Value elements. One field rename per call."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("old", gomcp.Required(), gomcp.Description("Old field name (without Fields! prefix)")),
		gomcp.WithString("new", gomcp.Required(), gomcp.Description("New field name")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleSwapFields}
}

func addDataSourceTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_add_datasource",
		gomcp.WithDescription("Add one DataSource. Idempotent: returns skipped if name already exists."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("name", gomcp.Required(), gomcp.Description("DataSource name")),
		gomcp.WithString("provider", gomcp.Required(), gomcp.Description("Data provider (e.g. SQL)")),
		gomcp.WithString("connectString", gomcp.Required(), gomcp.Description("Connection string")),
		gomcp.WithString("securityType", gomcp.Description("Security type; defaults to None")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleAddDataSource}
}

func removeDataSourceTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_remove_datasource",
		gomcp.WithDescription("Remove one DataSource by name."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("name", gomcp.Required(), gomcp.Description("DataSource name to remove")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleRemoveDataSource}
}

func renameDataSourceTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_rename_datasource",
		gomcp.WithDescription("Rename one DataSource and update DataSourceName references in DataSets."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("old", gomcp.Required(), gomcp.Description("Current name")),
		gomcp.WithString("new", gomcp.Required(), gomcp.Description("New name")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleRenameDataSource}
}

func setDataSourceConnectStringTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_set_datasource_connect_string",
		gomcp.WithDescription("Replace ConnectString of one named DataSource."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("name", gomcp.Required(), gomcp.Description("DataSource name")),
		gomcp.WithString("connectString", gomcp.Required(), gomcp.Description("New connection string")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleSetDataSourceConnectString}
}

func addDataSetTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_add_dataset",
		gomcp.WithDescription("Add one DataSet. Idempotent: returns skipped if name already exists."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("name", gomcp.Required(), gomcp.Description("DataSet name")),
		gomcp.WithString("datasource", gomcp.Required(), gomcp.Description("Bound DataSource name")),
		gomcp.WithString("cmdText", gomcp.Required(), gomcp.Description("SQL command text")),
		gomcp.WithArray("fields", gomcp.Description("Field names to add"), gomcp.WithStringItems()),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleAddDataSet}
}

func removeDataSetTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_remove_dataset",
		gomcp.WithDescription("Remove one DataSet by name."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("name", gomcp.Required(), gomcp.Description("DataSet name")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleRemoveDataSet}
}

func renameDataSetTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_rename_dataset",
		gomcp.WithDescription("Rename one DataSet and update DataSetName references in Tablixes."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("old", gomcp.Required(), gomcp.Description("Current name")),
		gomcp.WithString("new", gomcp.Required(), gomcp.Description("New name")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleRenameDataSet}
}

func addDataSetFieldTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_add_dataset_field",
		gomcp.WithDescription("Add one Field to a DataSet."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("dataset", gomcp.Required(), gomcp.Description("DataSet name")),
		gomcp.WithString("field", gomcp.Required(), gomcp.Description("Field name")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleAddDataSetField}
}

func clearDataSetFieldsTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_clear_dataset_fields",
		gomcp.WithDescription("Remove all Field elements from a DataSet (container preserved)."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("name", gomcp.Required(), gomcp.Description("DataSet name")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleClearDataSetFields}
}

func clearDataSetFiltersTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_clear_dataset_filters",
		gomcp.WithDescription("Remove Filters section from a DataSet."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("name", gomcp.Required(), gomcp.Description("DataSet name")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleClearDataSetFilters}
}

func setDataSetCommandTextTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_set_dataset_command_text",
		gomcp.WithDescription("Replace CommandText of one DataSet."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("dataset", gomcp.Required(), gomcp.Description("DataSet name")),
		gomcp.WithString("cmdText", gomcp.Required(), gomcp.Description("New SQL command text")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleSetDataSetCommandText}
}

func addParameterTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_add_parameter",
		gomcp.WithDescription("Add one ReportParameter. Idempotent: returns skipped if name already exists."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("name", gomcp.Required(), gomcp.Description("Parameter name")),
		gomcp.WithString("type", gomcp.Required(), gomcp.Description("Data type"),
			gomcp.Enum("String", "Integer", "Float", "Boolean", "DateTime")),
		gomcp.WithString("default", gomcp.Description("Default value")),
		gomcp.WithString("prompt", gomcp.Description("Prompt text; defaults to name")),
		gomcp.WithBoolean("hidden", gomcp.Description("Hide parameter in UI")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleAddParameter}
}

func removeParameterTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_remove_parameter",
		gomcp.WithDescription("Remove one ReportParameter and clean orphaned layout cells."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("name", gomcp.Required(), gomcp.Description("Parameter name")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleRemoveParameter}
}

func rebuildTablixTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_rebuild_tablix",
		gomcp.WithDescription("Rebuild a Tablix from a JSON spec. Colspan without placeholder cells."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithObject("spec", gomcp.Required(),
			gomcp.Description(`{"name":"X","columns":[3.0,5.0],"dataset":"DS","rows":[{"height":"0.5cm","cells":[{"textbox":"T","value":"v","colspan":2}]}]}`)),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleRebuildTablix}
}

func tablixSetCellTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_tablix_set_cell",
		gomcp.WithDescription("Set one cell value at (row, col). Indexes are 0-based cell positions (colspan not expanded)."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("tablix", gomcp.Required(), gomcp.Description("Tablix Name")),
		gomcp.WithInteger("row", gomcp.Required(), gomcp.Description("Row index")),
		gomcp.WithInteger("col", gomcp.Required(), gomcp.Description("Cell index in row")),
		gomcp.WithString("value", gomcp.Required(), gomcp.Description("Cell value; use expression:true for VB expressions")),
		gomcp.WithString("format", gomcp.Description("Format string (e.g. N2)")),
		gomcp.WithBoolean("expression", gomcp.Description("Prepend '=' if missing")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleTablixSetCell}
}

func tablixAddRowTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_tablix_add_row",
		gomcp.WithDescription("Append or insert one row into a Tablix."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("tablix", gomcp.Required(), gomcp.Description("Tablix Name")),
		gomcp.WithString("height", gomcp.Description("Row height (default 0.5cm)")),
		gomcp.WithArray("cells", gomcp.Description(`[{"textbox":"X","value":"Y","colspan":1}]`)),
		gomcp.WithInteger("index", gomcp.Description("Insert position; -1 to append")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleTablixAddRow}
}

func tablixRemoveRowTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_tablix_remove_row",
		gomcp.WithDescription("Remove one row from a Tablix by index."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("tablix", gomcp.Required(), gomcp.Description("Tablix Name")),
		gomcp.WithInteger("index", gomcp.Required(), gomcp.Description("Row index")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleTablixRemoveRow}
}

func tablixAddColumnTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_tablix_add_column",
		gomcp.WithDescription("Append or insert one column. ColSpan values are NOT adjusted on existing rows."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("tablix", gomcp.Required(), gomcp.Description("Tablix Name")),
		gomcp.WithString("width", gomcp.Description("Column width (default 2.5cm)")),
		gomcp.WithInteger("index", gomcp.Description("Insert position; -1 to append")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleTablixAddColumn}
}

func tablixRemoveColumnTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_tablix_remove_column",
		gomcp.WithDescription("Remove one column by index. ColSpan values are NOT adjusted."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("tablix", gomcp.Required(), gomcp.Description("Tablix Name")),
		gomcp.WithInteger("index", gomcp.Required(), gomcp.Description("Column index")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleTablixRemoveColumn}
}

func fixEncodingTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_fix_encoding",
		gomcp.WithDescription("Ensure UTF-8 BOM and CRLF line endings."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
	)
	return server.ServerTool{Tool: tool, Handler: handleFixEncoding}
}

func registerTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_register",
		gomcp.WithDescription("Register an RDL file in a .rptproj project file."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
		gomcp.WithString("project", gomcp.Required(), gomcp.Description("Path to .rptproj file")),
	)
	return server.ServerTool{Tool: tool, Handler: handleRegister}
}

func validateTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_validate",
		gomcp.WithDescription("Validate RDL structure and references. Check data.pass in the result."),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
	)
	return server.ServerTool{Tool: tool, Handler: handleValidate}
}

func applyThemeTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_apply_theme",
		gomcp.WithDescription("Copy visual theming from a source report to a target. Copies HeaderTheme shared dataset, PageHeader, PageFooter, margins, and page dimensions. Adds pvc_Theme and vc_ReportPack parameters if missing. Does NOT touch data sources, datasets (except HeaderTheme), parameters (except theme params), or tablix."),
		gomcp.WithString("source", gomcp.Required(), gomcp.Description("Path to source RDL file (the report to copy theme from)")),
		gomcp.WithString("target", gomcp.Required(), gomcp.Description("Path to target RDL file (the report to apply theme to)")),
		withDryRunParam(),
	)
	return server.ServerTool{Tool: tool, Handler: handleApplyTheme}
}

// ── Handlers ────────────────────────────────────────────────────────────────

func handleCreate(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	target, res, _ := requireString(request, "target")
	if res != nil {
		return res, nil
	}
	title, res, _ := requireString(request, "title")
	if res != nil {
		return res, nil
	}
	spec := rdl.CreateSpec{
		Target:      target,
		Title:       title,
		Orientation: request.GetString("orientation", ""),
		Description: request.GetString("description", ""),
		Author:      request.GetString("author", ""),
		FontFamily:  request.GetString("fontFamily", ""),
		PageWidth:   request.GetString("pageWidth", ""),
		PageHeight:  request.GetString("pageHeight", ""),
		LeftMargin:  request.GetString("leftMargin", ""),
		RightMargin: request.GetString("rightMargin", ""),
		TopMargin:   request.GetString("topMargin", ""),
		BottomMargin: request.GetString("bottomMargin", ""),
	}
	dry := dryRunFromRequest(request)
	newID, err := rdl.Create(spec, dry)
	if err != nil {
		return mapError(err)
	}
	data := map[string]any{"target": target, "reportId": newID, "title": title, "orientation": spec.Orientation}
	summary := fmt.Sprintf("Created %s (ReportID: %s, %s)", target, newID, spec.Orientation)
	return successResult("rdl_create", target, dry, data, summary)
}

func handleClone(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	source, res, _ := requireString(request, "source")
	if res != nil {
		return res, nil
	}
	target, res, _ := requireString(request, "target")
	if res != nil {
		return res, nil
	}
	dry := dryRunFromRequest(request)
	newID, err := rdl.Clone(source, target, dry)
	if err != nil {
		return mapError(err)
	}
	data := map[string]any{"source": source, "target": target, "reportId": newID}
	summary := fmt.Sprintf("Cloned %s -> %s (ReportID: %s)", source, target, newID)
	return successResult("rdl_clone", target, dry, data, summary)
}

func handleUpdateMetadata(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	spec := rdl.MetadataUpdate{
		Description:  request.GetString("description", ""),
		Title:        request.GetString("title", ""),
		TitleTextbox: request.GetString("titleTextbox", ""),
		Orientation:  request.GetString("orientation", ""),
	}
	if spec.Description == "" && spec.Title == "" && spec.Orientation == "" {
		res, _ := errorResult("PRECONDITION", "at least one of description, title, or orientation must be provided", "", nil)
		return res, nil
	}
	if spec.Title != "" && spec.TitleTextbox == "" {
		res, _ := errorResult("PRECONDITION", "title requires titleTextbox",
			"call rdl_list_tablixes to find the textbox name", nil)
		return res, nil
	}
	dry := dryRunFromRequest(request)
	count, err := rdl.UpdateMetadata(file, spec, dry)
	if err != nil {
		return mapError(err)
	}
	data := map[string]any{"fieldsUpdated": count}
	return successResult("rdl_update_metadata", file, dry, data,
		fmt.Sprintf("Updated %d metadata field(s)", count))
}

func handleSwapMacros(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	old, res, _ := requireString(request, "old")
	if res != nil {
		return res, nil
	}
	newVal, res, _ := requireString(request, "new")
	if res != nil {
		return res, nil
	}
	dry := dryRunFromRequest(request)
	count, err := rdl.SwapMacro(file, old, newVal, dry)
	if err != nil {
		return mapError(err)
	}
	data := map[string]any{"replacements": count, "old": old, "new": newVal}
	return successResult("rdl_swap_macros", file, dry, data,
		fmt.Sprintf("Replaced %d occurrence(s) in ConnectString elements", count))
}

func handleSwapFields(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	old, res, _ := requireString(request, "old")
	if res != nil {
		return res, nil
	}
	newVal, res, _ := requireString(request, "new")
	if res != nil {
		return res, nil
	}
	dry := dryRunFromRequest(request)
	count, err := rdl.SwapField(file, old, newVal, dry)
	if err != nil {
		return mapError(err)
	}
	data := map[string]any{"replacements": count, "old": old, "new": newVal}
	return successResult("rdl_swap_fields", file, dry, data,
		fmt.Sprintf("Replaced %d occurrence(s) in Value elements", count))
}

func handleAddDataSource(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	name, res, _ := requireString(request, "name")
	if res != nil {
		return res, nil
	}
	provider, res, _ := requireString(request, "provider")
	if res != nil {
		return res, nil
	}
	connectString, res, _ := requireString(request, "connectString")
	if res != nil {
		return res, nil
	}
	spec := rdl.DataSourceAdd{
		Name:          name,
		Provider:      provider,
		ConnectString: connectString,
		SecurityType:  request.GetString("securityType", ""),
	}
	out, err := rdl.AddDataSource(file, spec, dryRunFromRequest(request))
	if err != nil {
		return mapError(err)
	}
	return successFromOutcome("rdl_add_datasource", file, dryRunFromRequest(request), out)
}

func handleRemoveDataSource(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	name, res, _ := requireString(request, "name")
	if res != nil {
		return res, nil
	}
	out, err := rdl.RemoveDataSource(file, name, dryRunFromRequest(request))
	if err != nil {
		return mapError(err)
	}
	return successFromOutcome("rdl_remove_datasource", file, dryRunFromRequest(request), out)
}

func handleRenameDataSource(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	old, res, _ := requireString(request, "old")
	if res != nil {
		return res, nil
	}
	newName, res, _ := requireString(request, "new")
	if res != nil {
		return res, nil
	}
	out, err := rdl.RenameDataSource(file, old, newName, dryRunFromRequest(request))
	if err != nil {
		return mapError(err)
	}
	return successFromOutcome("rdl_rename_datasource", file, dryRunFromRequest(request), out)
}

func handleSetDataSourceConnectString(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	name, res, _ := requireString(request, "name")
	if res != nil {
		return res, nil
	}
	connectString, res, _ := requireString(request, "connectString")
	if res != nil {
		return res, nil
	}
	out, err := rdl.SetDataSourceConnectString(file, name, connectString, dryRunFromRequest(request))
	if err != nil {
		return mapError(err)
	}
	return successFromOutcome("rdl_set_datasource_connect_string", file, dryRunFromRequest(request), out)
}

func handleAddDataSet(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	name, res, _ := requireString(request, "name")
	if res != nil {
		return res, nil
	}
	datasource, res, _ := requireString(request, "datasource")
	if res != nil {
		return res, nil
	}
	cmdText, res, _ := requireString(request, "cmdText")
	if res != nil {
		return res, nil
	}
	fields, _ := request.RequireStringSlice("fields")
	spec := rdl.DataSetAdd{Name: name, DataSource: datasource, CmdText: cmdText, Fields: fields}
	out, err := rdl.AddDataSet(file, spec, dryRunFromRequest(request))
	if err != nil {
		return mapError(err)
	}
	return successFromOutcome("rdl_add_dataset", file, dryRunFromRequest(request), out)
}

func handleRemoveDataSet(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	name, res, _ := requireString(request, "name")
	if res != nil {
		return res, nil
	}
	out, err := rdl.RemoveDataSet(file, name, dryRunFromRequest(request))
	if err != nil {
		return mapError(err)
	}
	return successFromOutcome("rdl_remove_dataset", file, dryRunFromRequest(request), out)
}

func handleRenameDataSet(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	old, res, _ := requireString(request, "old")
	if res != nil {
		return res, nil
	}
	newName, res, _ := requireString(request, "new")
	if res != nil {
		return res, nil
	}
	out, err := rdl.RenameDataSet(file, old, newName, dryRunFromRequest(request))
	if err != nil {
		return mapError(err)
	}
	return successFromOutcome("rdl_rename_dataset", file, dryRunFromRequest(request), out)
}

func handleAddDataSetField(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	dataset, res, _ := requireString(request, "dataset")
	if res != nil {
		return res, nil
	}
	field, res, _ := requireString(request, "field")
	if res != nil {
		return res, nil
	}
	out, err := rdl.AddDataSetField(file, dataset, field, dryRunFromRequest(request))
	if err != nil {
		return mapError(err)
	}
	return successFromOutcome("rdl_add_dataset_field", file, dryRunFromRequest(request), out)
}

func handleClearDataSetFields(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	name, res, _ := requireString(request, "name")
	if res != nil {
		return res, nil
	}
	out, err := rdl.ClearDataSetFields(file, name, dryRunFromRequest(request))
	if err != nil {
		return mapError(err)
	}
	return successFromOutcome("rdl_clear_dataset_fields", file, dryRunFromRequest(request), out)
}

func handleClearDataSetFilters(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	name, res, _ := requireString(request, "name")
	if res != nil {
		return res, nil
	}
	out, err := rdl.ClearDataSetFilters(file, name, dryRunFromRequest(request))
	if err != nil {
		return mapError(err)
	}
	return successFromOutcome("rdl_clear_dataset_filters", file, dryRunFromRequest(request), out)
}

func handleSetDataSetCommandText(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	dataset, res, _ := requireString(request, "dataset")
	if res != nil {
		return res, nil
	}
	cmdText, res, _ := requireString(request, "cmdText")
	if res != nil {
		return res, nil
	}
	out, err := rdl.SetDataSetCommandText(file, dataset, cmdText, dryRunFromRequest(request))
	if err != nil {
		return mapError(err)
	}
	return successFromOutcome("rdl_set_dataset_command_text", file, dryRunFromRequest(request), out)
}

func handleAddParameter(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	name, res, _ := requireString(request, "name")
	if res != nil {
		return res, nil
	}
	typ, res, _ := requireString(request, "type")
	if res != nil {
		return res, nil
	}
	spec := rdl.ParameterAdd{
		Name:    name,
		Type:    typ,
		Default: request.GetString("default", ""),
		Prompt:  request.GetString("prompt", ""),
		Hidden:  request.GetBool("hidden", false),
	}
	out, err := rdl.AddParameter(file, spec, dryRunFromRequest(request))
	if err != nil {
		return mapError(err)
	}
	return successFromOutcome("rdl_add_parameter", file, dryRunFromRequest(request), out)
}

func handleRemoveParameter(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	name, res, _ := requireString(request, "name")
	if res != nil {
		return res, nil
	}
	out, err := rdl.RemoveParameter(file, name, dryRunFromRequest(request))
	if err != nil {
		return mapError(err)
	}
	return successFromOutcome("rdl_remove_parameter", file, dryRunFromRequest(request), out)
}

func handleRebuildTablix(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	specObj := request.GetArguments()["spec"]
	if specObj == nil {
		res, _ := errorResult("ARG_MISSING", "spec is required", "", map[string]any{"param": "spec"})
		return res, nil
	}
	specBytes, err := json.Marshal(specObj)
	if err != nil {
		res, _ := errorResult("ARG_INVALID", fmt.Sprintf("failed to marshal spec: %v", err), "", map[string]any{"param": "spec"})
		return res, nil
	}
	var spec rdl.TablixSpec
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		res, _ := errorResult("ARG_INVALID", fmt.Sprintf("invalid spec: %v", err), "", map[string]any{"param": "spec"})
		return res, nil
	}
	dry := dryRunFromRequest(request)
	summary, err := rdl.RebuildTablixFile(file, spec, dry)
	if err != nil {
		return mapError(rdl.MapLoadError(err, file))
	}
	data := map[string]any{"tablix": spec.Name, "summary": summary}
	return successResult("rdl_rebuild_tablix", file, dry, data, summary)
}

func handleTablixSetCell(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	tablix, res, _ := requireString(request, "tablix")
	if res != nil {
		return res, nil
	}
	row, res, _ := requireInt(request, "row")
	if res != nil {
		return res, nil
	}
	col, res, _ := requireInt(request, "col")
	if res != nil {
		return res, nil
	}
	value, res, _ := requireString(request, "value")
	if res != nil {
		return res, nil
	}
	cv := rdl.CellValue{
		Value:      value,
		Format:     request.GetString("format", ""),
		Expression: request.GetBool("expression", false),
	}
	dry := dryRunFromRequest(request)
	summary, err := rdl.TablixSetCell(file, tablix, row, col, cv, dry)
	if err != nil {
		return mapError(err)
	}
	data := map[string]any{"tablix": tablix, "row": row, "col": col, "value": value}
	return successResult("rdl_tablix_set_cell", file, dry, data, summary)
}

func handleTablixAddRow(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	tablix, res, _ := requireString(request, "tablix")
	if res != nil {
		return res, nil
	}
	height := request.GetString("height", "")
	index := int(request.GetInt("index", -1))
	row := rdl.RowSpec{Height: height}
	if raw, ok := request.GetArguments()["cells"]; ok && raw != nil {
		b, _ := json.Marshal(raw)
		if err := json.Unmarshal(b, &row.Cells); err != nil {
			res, _ := errorResult("ARG_INVALID", fmt.Sprintf("invalid cells: %v", err), "", map[string]any{"param": "cells"})
			return res, nil
		}
	}
	dry := dryRunFromRequest(request)
	summary, err := rdl.TablixAddRow(file, tablix, row, index, dry)
	if err != nil {
		return mapError(err)
	}
	return successResult("rdl_tablix_add_row", file, dry, map[string]any{"tablix": tablix, "index": index}, summary)
}

func handleTablixRemoveRow(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	tablix, res, _ := requireString(request, "tablix")
	if res != nil {
		return res, nil
	}
	index, res, _ := requireInt(request, "index")
	if res != nil {
		return res, nil
	}
	dry := dryRunFromRequest(request)
	summary, err := rdl.TablixRemoveRow(file, tablix, index, dry)
	if err != nil {
		return mapError(err)
	}
	return successResult("rdl_tablix_remove_row", file, dry, map[string]any{"tablix": tablix, "index": index}, summary)
}

func handleTablixAddColumn(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	tablix, res, _ := requireString(request, "tablix")
	if res != nil {
		return res, nil
	}
	width := request.GetString("width", "")
	index := int(request.GetInt("index", -1))
	dry := dryRunFromRequest(request)
	summary, err := rdl.TablixAddColumn(file, tablix, width, index, dry)
	if err != nil {
		return mapError(err)
	}
	return successResult("rdl_tablix_add_column", file, dry, map[string]any{"tablix": tablix, "index": index}, summary)
}

func handleTablixRemoveColumn(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	tablix, res, _ := requireString(request, "tablix")
	if res != nil {
		return res, nil
	}
	index, res, _ := requireInt(request, "index")
	if res != nil {
		return res, nil
	}
	dry := dryRunFromRequest(request)
	summary, err := rdl.TablixRemoveColumn(file, tablix, index, dry)
	if err != nil {
		return mapError(err)
	}
	return successResult("rdl_tablix_remove_column", file, dry, map[string]any{"tablix": tablix, "index": index}, summary)
}

func handleFixEncoding(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	summary, err := rdl.FixEncoding(file)
	if err != nil {
		return mapError(err)
	}
	return successResult("rdl_fix_encoding", file, false, nil, summary)
}

func handleRegister(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	project, res, _ := requireString(request, "project")
	if res != nil {
		return res, nil
	}
	summary, err := rdl.Register(file, project)
	if err != nil {
		return mapError(err)
	}
	data := map[string]any{"file": file, "project": project}
	return successResult("rdl_register", file, false, data, summary)
}

func handleValidate(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, res, _ := requireString(request, "file")
	if res != nil {
		return res, nil
	}
	report, err := rdl.Validate(file)
	if err != nil {
		return mapError(rdl.MapLoadError(err, file))
	}
	summary := "validation passed"
	if !report.Pass {
		summary = fmt.Sprintf("validation failed with %d issue(s)", len(report.Issues))
	}
	return successResult("rdl_validate", file, false, report, summary)
}

func handleApplyTheme(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	source, res, _ := requireString(request, "source")
	if res != nil {
		return res, nil
	}
	target, res, _ := requireString(request, "target")
	if res != nil {
		return res, nil
	}
	summary, err := rdl.ApplyTheme(source, target, dryRunFromRequest(request))
	if err != nil {
		return mapError(err)
	}
	data := map[string]any{"source": source, "target": target}
	return successResult("rdl_apply_theme", target, dryRunFromRequest(request), data, summary)
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func dryRunFromRequest(request gomcp.CallToolRequest) bool {
	return request.GetBool("dryRun", false)
}

func withDryRunParam() gomcp.ToolOption {
	return gomcp.WithBoolean("dryRun",
		gomcp.Description("If true, preview the mutation without writing. Result JSON includes dryRun:true."))
}

func inspectTool(name, description string, fn func(*rdl.Document) any) server.ServerTool {
	tool := gomcp.NewTool(name,
		gomcp.WithDescription(description),
		gomcp.WithString("file", gomcp.Required(), gomcp.Description("Path to RDL file")),
	)
	handler := func(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		file, res, _ := requireString(request, "file")
		if res != nil {
			return res, nil
		}
		doc, err := rdl.Load(file)
		if err != nil {
			return mapError(rdl.MapLoadError(err, file))
		}
		return successResult(name, file, false, fn(doc), "ok")
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}
