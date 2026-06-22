# RDL Toolkit

A Go CLI tool and MCP server for manipulating SSRS RDL (Report Definition Language) XML files.

## Features

- **Clone** RDL files with new ReportIDs
- **Swap macros** and field references in bulk
- **Manage DataSources, DataSets, and Parameters**
- **Rebuild Tablix** from JSON specifications
- **Fix encoding** (UTF-8 BOM + CRLF)
- **Register** RDL files in .rptproj projects
- **Validate** RDL structure
- **MCP server** mode for AI agent integration

## Installation

```bash
go build -o bin/rdl-tool ./cmd/rdl-tool
# or
go install ./cmd/rdl-tool
```

## CLI Usage

```bash
# Clone a report
rdl-tool clone --source report.rdl --target new-report.rdl

# Update metadata
rdl-tool update-metadata report.rdl --description "New Title" --orientation Landscape

# Replace macros in ConnectString elements
rdl-tool swap-macros report.rdl "old_macro:new_macro"

# Replace field references
rdl-tool swap-fields report.rdl "OldField:NewField"

# Manage DataSources (--add takes JSON; connection strings with colons survive)
rdl-tool manage-datasources report.rdl --add '{"name":"NewDS","provider":"SQL","connectString":"Data Source=server;Initial Catalog=test","securityType":"Integrated"}' --remove OldDS

# Manage DataSets (--add takes JSON; SQL with colons in cmdText survives)
rdl-tool manage-datasets report.rdl --add '{"name":"NewDS","datasource":"NewDS","cmdText":"SELECT a:b FROM t","fields":["a","b"]}' --remove OldDS

# Manage Parameters
rdl-tool manage-parameters report.rdl --add '{"name":"P1","type":"String","prompt":"P1"}' --remove OldParam

# Per-tablix cell edits (target by name)
rdl-tool tablix-set-cell report.rdl --tablix SalesTable --row 0 --col 0 --value "New Header"
rdl-tool tablix-add-row report.rdl --tablix SalesTable --cells '[{"textbox":"X","value":"Y"}]'
rdl-tool tablix-add-column report.rdl --tablix SalesTable --width "4cm"

# Rebuild Tablix from JSON spec
rdl-tool rebuild-tablix report.rdl spec.json

# Fix encoding
rdl-tool fix-encoding report.rdl

# Register in project file
rdl-tool register report.rdl project.rptproj

# Validate structure (use --json for machine-readable output)
rdl-tool validate report.rdl

# Any mutation can be previewed with --dry-run (file is not written)
rdl-tool --dry-run manage-datasources report.rdl --add '{"name":"X","provider":"SQL","connectString":"..."}'
```

## MCP Server Usage

Start as an MCP server for AI agent integration (VS Code, Claude, Hermes, etc.):

```bash
rdl-tool --mcp
```

### Copilot CLI Configuration

Add to `.github/mcp.json` in your project:

```json
{
  "servers": {
    "rdl-toolkit": {
      "type": "stdio",
      "command": "SSRS/tools/rdl-tool",
      "args": ["--mcp"]
    }
  }
}
```

### VS Code Configuration

Add to `.vscode/mcp.json` in your project:

```json
{
  "servers": {
    "rdl-toolkit": {
      "type": "stdio",
      "command": "rdl-tool",
      "args": ["--mcp"]
    }
  }
}
```

### Available MCP Tools

**Inspection (read-only — call these first to understand a report):**

| Tool | Description |
|------|-------------|
| `rdl_inspect` | Top-level summary: ReportID, language, page size/orientation, counts of datasources/datasets/parameters/tablixes |
| `rdl_list_datasources` | Each datasource with provider, connect string, security type, ID |
| `rdl_list_datasets` | Each dataset with bound datasource, command text, fields, filter count |
| `rdl_list_parameters` | Each parameter with type, flags (nullable/hidden/multivalue), prompt, default |
| `rdl_list_tablixes` | Each tablix with name, bound dataset, columns, and per-cell textbox + value |
| `rdl_get_metadata` | Report metadata: description, language, author, page size, margins |

**Mutations:**

| Tool | Description |
|------|-------------|
| `rdl_clone` | Copy an RDL file with a new ReportID |
| `rdl_update_metadata` | Update report metadata (description, title by textbox, orientation) |
| `rdl_swap_macros` | Replace strings in ConnectString elements |
| `rdl_swap_fields` | Replace Fields!X.Value references in Value elements |
| `rdl_manage_datasources` | Add, remove, rename DataSources, or set their ConnectStrings (idempotent) |
| `rdl_manage_datasets` | Add, remove, rename DataSets, edit fields, update CommandText (idempotent) |
| `rdl_manage_parameters` | Add or remove ReportParameters (auto-cleans orphaned layout cells) |
| `rdl_rebuild_tablix` | Rebuild a Tablix from a JSON spec (targeted by name; correct colspan semantics) |
| `rdl_tablix_set_cell` | Set the value of a single cell at (row, col) |
| `rdl_tablix_add_row` | Append or insert a row |
| `rdl_tablix_remove_row` | Remove a row by index |
| `rdl_tablix_add_column` | Append or insert a column |
| `rdl_tablix_remove_column` | Remove a column by index |
| `rdl_fix_encoding` | Fix UTF-8 BOM and CRLF line endings |
| `rdl_register` | Register an RDL in a .rptproj file |
| `rdl_validate` | Validate RDL structure |

## Cross-Platform Build

```bash
make build-all
```

Produces binaries for Linux, Windows, and macOS (amd64 + arm64).
