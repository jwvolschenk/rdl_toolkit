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
rdl-tool update-metadata report.rdl --title "New Title" --orientation Landscape

# Replace macros in ConnectString elements
rdl-tool swap-macros report.rdl "old_macro:new_macro"

# Replace field references
rdl-tool swap-fields report.rdl "OldField:NewField"

# Manage DataSources
rdl-tool manage-datasources report.rdl --add "NewDS:SQL:Server=db;Database=test" --remove OldDS

# Manage DataSets
rdl-tool manage-datasets report.rdl --add "NewDS:NewDS:SELECT *:Col1,Col2" --remove OldDS

# Manage Parameters
rdl-tool manage-parameters report.rdl --add "Param1:String::true" --remove OldParam

# Rebuild Tablix from JSON spec
rdl-tool rebuild-tablix report.rdl spec.json

# Fix encoding
rdl-tool fix-encoding report.rdl

# Register in project file
rdl-tool register report.rdl project.rptproj

# Validate structure
rdl-tool validate report.rdl
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

| Tool | Description |
|------|-------------|
| `rdl_clone` | Copy an RDL file with a new ReportID |
| `rdl_update_metadata` | Update report metadata (description, title, orientation) |
| `rdl_swap_macros` | Replace strings in ConnectString elements |
| `rdl_swap_fields` | Replace Fields!X.Value references in Value elements |
| `rdl_manage_datasources` | Add, remove, or rename DataSources |
| `rdl_manage_datasets` | Add, remove, or rename DataSets |
| `rdl_manage_parameters` | Add or remove ReportParameters |
| `rdl_rebuild_tablix` | Rebuild the first Tablix from a JSON spec |
| `rdl_fix_encoding` | Fix UTF-8 BOM and CRLF line endings |
| `rdl_register` | Register an RDL in a .rptproj file |
| `rdl_validate` | Validate RDL structure |

## Cross-Platform Build

```bash
make build-all
```

Produces binaries for Linux, Windows, and macOS (amd64 + arm64).
