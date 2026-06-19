package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
		cloneTool(),
		updateMetadataTool(),
		swapMacrosTool(),
		swapFieldsTool(),
		manageDatasourcesTool(),
		manageDatasetsTool(),
		manageParametersTool(),
		rebuildTablixTool(),
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
	)
	return server.ServerTool{Tool: tool, Handler: handleClone}
}

func updateMetadataTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_update_metadata",
		gomcp.WithDescription("Update report metadata: description, title, and/or page orientation."),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
		gomcp.WithString("description",
			gomcp.Description("Pipe-delimited description (e.g. 'Section|Subsection|Detail')"),
		),
		gomcp.WithString("title",
			gomcp.Description("Report title"),
		),
		gomcp.WithString("orientation",
			gomcp.Description("Portrait or Landscape"),
			gomcp.Enum("Portrait", "Landscape"),
		),
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
	)
	return server.ServerTool{Tool: tool, Handler: handleSwapFields}
}

func manageDatasourcesTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_manage_datasources",
		gomcp.WithDescription("Add, remove, or rename DataSources in an RDL file."),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
		gomcp.WithArray("add",
			gomcp.Description("DataSources to add, each with name, provider, and connectString"),
		),
		gomcp.WithArray("remove",
			gomcp.Description("DataSource names to remove"),
			gomcp.WithStringItems(),
		),
		gomcp.WithArray("rename",
			gomcp.Description("Renames, each with old and new name"),
		),
	)
	return server.ServerTool{Tool: tool, Handler: handleManageDatasources}
}

func manageDatasetsTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_manage_datasets",
		gomcp.WithDescription("Add, remove, or rename DataSets, or add fields to existing DataSets."),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
		gomcp.WithArray("add",
			gomcp.Description("DataSets to add, each with name, datasource, cmdText, and fields (comma-separated string)"),
		),
		gomcp.WithArray("remove",
			gomcp.Description("DataSet names to remove"),
			gomcp.WithStringItems(),
		),
		gomcp.WithArray("rename",
			gomcp.Description("Renames, each with old and new name"),
		),
		gomcp.WithArray("add_field",
			gomcp.Description("Fields to add, each with dataset (name) and field (field name)"),
		),
	)
	return server.ServerTool{Tool: tool, Handler: handleManageDatasets}
}

func manageParametersTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_manage_parameters",
		gomcp.WithDescription("Add or remove ReportParameters in an RDL file."),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
		gomcp.WithArray("add",
			gomcp.Description("Parameters to add, each with name, type, default, and hidden"),
		),
		gomcp.WithArray("remove",
			gomcp.Description("Parameter names to remove"),
			gomcp.WithStringItems(),
		),
	)
	return server.ServerTool{Tool: tool, Handler: handleManageParameters}
}

func rebuildTablixTool() server.ServerTool {
	tool := gomcp.NewTool("rdl_rebuild_tablix",
		gomcp.WithDescription("Rebuild the first Tablix in an RDL file from a JSON spec. The spec defines columns, rows, cells, and styles."),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
		gomcp.WithObject("spec",
			gomcp.Required(),
			gomcp.Description("Tablix specification object with columns, rows, and cells"),
		),
	)
	return server.ServerTool{Tool: tool, Handler: handleRebuildTablix}
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
		gomcp.WithDescription("Validate RDL structure: tablix columns match cell count, field sources exist in datasets, row hierarchy is consistent."),
		gomcp.WithString("file",
			gomcp.Required(),
			gomcp.Description("Path to RDL file"),
		),
	)
	return server.ServerTool{Tool: tool, Handler: handleValidate}
}

// ── Argument structs for BindArguments ──────────────────────────────────────

type addDataSourceArgs struct {
	Name          string `json:"name"`
	Provider      string `json:"provider"`
	ConnectString string `json:"connectString"`
}

type renameArgs struct {
	Old string `json:"old"`
	New string `json:"new"`
}

type addDataSetArgs struct {
	Name       string `json:"name"`
	Datasource string `json:"datasource"`
	CmdText    string `json:"cmdText"`
	Fields     string `json:"fields"` // comma-separated
}

type addFieldArgs struct {
	Dataset string `json:"dataset"`
	Field   string `json:"field"`
}

type addParamArgs struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default string `json:"default"`
	Hidden  string `json:"hidden"`
}

type datasourcesArgs struct {
	File   string             `json:"file"`
	Add    []addDataSourceArgs `json:"add"`
	Remove []string           `json:"remove"`
	Rename []renameArgs       `json:"rename"`
}

type datasetsArgs struct {
	File     string          `json:"file"`
	Add      []addDataSetArgs `json:"add"`
	Remove   []string        `json:"remove"`
	Rename   []renameArgs    `json:"rename"`
	AddField []addFieldArgs  `json:"add_field"`
}

type parametersArgs struct {
	File   string         `json:"file"`
	Add    []addParamArgs `json:"add"`
	Remove []string       `json:"remove"`
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

	newID, err := rdl.Clone(source, target)
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
	description := request.GetString("description", "")
	title := request.GetString("title", "")
	orientation := request.GetString("orientation", "")

	if description == "" && title == "" && orientation == "" {
		return gomcp.NewToolResultError("at least one of description, title, or orientation must be provided"), nil
	}

	count, err := rdl.UpdateMetadata(file, description, title, orientation)
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

	count, err := rdl.SwapMacros(file, pairs)
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

	count, err := rdl.SwapFields(file, pairs)
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(fmt.Sprintf("Replaced %d occurrence(s) in Value elements of %s", count, file)), nil
}

func handleManageDatasources(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	var args datasourcesArgs
	if err := request.BindArguments(&args); err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	// Convert to rdl types: adds as [][3]string, renames as [][2]string
	adds := make([][3]string, len(args.Add))
	for i, a := range args.Add {
		adds[i] = [3]string{a.Name, a.Provider, a.ConnectString}
	}
	renames := make([][2]string, len(args.Rename))
	for i, r := range args.Rename {
		renames[i] = [2]string{r.Old, r.New}
	}

	summary, err := rdl.ManageDataSources(args.File, adds, args.Remove, renames)
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(summary), nil
}

func handleManageDatasets(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	var args datasetsArgs
	if err := request.BindArguments(&args); err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	// Convert to rdl types
	adds := make([]rdl.DatasetAddInfo, len(args.Add))
	for i, a := range args.Add {
		var fields []string
		if a.Fields != "" {
			fields = strings.Split(a.Fields, ",")
		}
		adds[i] = rdl.DatasetAddInfo{
			Name:    a.Name,
			DS:      a.Datasource,
			CmdText: a.CmdText,
			Fields:  fields,
		}
	}
	renames := make([][2]string, len(args.Rename))
	for i, r := range args.Rename {
		renames[i] = [2]string{r.Old, r.New}
	}
	addFields := make([][2]string, len(args.AddField))
	for i, f := range args.AddField {
		addFields[i] = [2]string{f.Dataset, f.Field}
	}

	summary, err := rdl.ManageDataSets(args.File, adds, args.Remove, renames, addFields)
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(summary), nil
}

func handleManageParameters(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	var args parametersArgs
	if err := request.BindArguments(&args); err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	adds := make([]rdl.ParamAddInfo, len(args.Add))
	for i, a := range args.Add {
		adds[i] = rdl.ParamAddInfo{
			Name:    a.Name,
			Type:    a.Type,
			Default: a.Default,
			Hidden:  a.Hidden,
		}
	}

	summary, err := rdl.ManageParameters(args.File, adds, args.Remove)
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

	// The spec is passed as a JSON object. We need to write it to a temp file
	// because rdl.RebuildTablix expects a file path for the spec.
	specObj := request.GetArguments()["spec"]
	if specObj == nil {
		return gomcp.NewToolResultError("required argument 'spec' not found"), nil
	}

	// Marshal the spec back to JSON bytes
	specBytes, err := json.Marshal(specObj)
	if err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("failed to marshal spec: %v", err)), nil
	}

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "rdl-tablix-spec-*.json")
	if err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("failed to create temp file: %v", err)), nil
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(specBytes); err != nil {
		tmpFile.Close()
		return gomcp.NewToolResultError(fmt.Sprintf("failed to write temp file: %v", err)), nil
	}
	tmpFile.Close()

	summary, err := rdl.RebuildTablix(file, tmpFile.Name())
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

	summary, err := rdl.Validate(file)
	if err != nil {
		return gomcp.NewToolResultError(err.Error()), nil
	}
	return gomcp.NewToolResultText(summary), nil
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func parsePairs(items []string) ([][2]string, error) {
	var pairs [][2]string
	for _, item := range items {
		parts := strings.SplitN(item, ":", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("invalid pair format %q, expected OLD:NEW", item)
		}
		pairs = append(pairs, [2]string{parts[0], parts[1]})
	}
	return pairs, nil
}
