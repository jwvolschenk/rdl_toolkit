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

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/jwvolschenk/rdl_toolkit/master/scripts/setup-rdl-tool.sh | bash
```

The installer downloads the correct binary for your platform, verifies the checksum, and prompts you to select your AI agent for MCP configuration.

To skip the prompt and specify an agent directly:

```bash
bash scripts/setup-rdl-tool.sh --agent Copilot
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/jwvolschenk/rdl_toolkit/master/scripts/setup-rdl-tool.ps1 | iex
```

Or download and run manually:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/setup-rdl-tool.ps1
powershell -ExecutionPolicy Bypass -File scripts/setup-rdl-tool.ps1 -Agent Copilot
```

### Build from source

```bash
bash scripts/build-rdl-tool.sh
# or
go build -o bin/rdl-tool ./cmd/rdl-tool
```

### Cross-platform build

```bash
make build-all
```

Produces binaries for Linux (amd64 + arm64), Windows (amd64), and macOS (amd64 + arm64).

## MCP Server Setup

After installing, the setup script will prompt you to choose your AI agent and display the exact configuration to add. The instructions below are also available for reference.

### Copilot (VS Code)

Add to `~/.copilot/mcp-config.json` (user-wide) or `.vscode/mcp.json` (per-workspace):

```json
{
  "mcpServers": {
    "rdl-toolkit": {
      "command": "rdl-tool",
      "args": ["--mcp"]
    }
  }
}
```

### Hermes

Add to `~/.hermes/config.yaml`:

```yaml
mcp_servers:
  rdl-toolkit:
    command: rdl-tool
    args:
      --mcp
    enabled: true
```

### Claude Desktop

Add to `~/.claude.json`:

```json
{
  "mcpServers": {
    "rdl-toolkit": {
      "command": "rdl-tool",
      "args": ["--mcp"]
    }
  }
}
```

### Codex

Add to `~/.codex/config.toml`:

```toml
[mcp_servers.rdl-toolkit]
command = "rdl-tool"
args = ["--mcp"]
startup_timeout_sec = 30
```

### Gemini

Add to `~/.gemini/settings.json`:

```json
{
  "mcpServers": {
    "rdl-toolkit": {
      "command": "rdl-tool",
      "args": ["--mcp"]
    }
  }
}
```

### Cursor

Add to `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "rdl-toolkit": {
      "command": "rdl-tool",
      "args": ["--mcp"]
    }
  }
}
```

### OpenCode

Add to `~/.config/opencode/opencode.jsonc` (mcp section):

```json
"rdl-toolkit": {
  "command": "rdl-tool",
  "args": ["--mcp"],
  "enabled": true
}
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

## Available MCP Tools (v2.0.0)

All tools return **structured JSON** with `ok`, `tool`, `file`, and optional `data` / `summary`. Errors return JSON with `code`, `message`, `hint`, and `context`.

**Recommended workflow:** `rdl_inspect` -> `rdl_list_*` -> mutate with `dryRun: true` -> `rdl_validate` -> mutate with `dryRun: false`.

**Inspection (read-only -- call these first):**

| Tool | Description |
|------|-------------|
| `rdl_inspect` | Top-level summary: ReportID, language, page size/orientation, counts |
| `rdl_list_datasources` | Each datasource with provider, connect string, security type, ID |
| `rdl_list_datasets` | Each dataset with bound datasource, command text, fields, filter count |
| `rdl_list_parameters` | Each parameter with type, flags, prompt, default |
| `rdl_list_tablixes` | Each tablix with name, dataset, columns, per-cell textbox + value |
| `rdl_get_metadata` | Report metadata: description, language, author, page size, margins |

**Mutations (atomic -- one operation per call):**

| Tool | Description |
|------|-------------|
| `rdl_clone` | Copy an RDL file with a new ReportID |
| `rdl_update_metadata` | Update description, title (by textbox), or orientation |
| `rdl_swap_macros` | Replace one string in ConnectString elements (`old`, `new`) |
| `rdl_swap_fields` | Replace one Fields!X.Value reference (`old`, `new` field names) |
| `rdl_add_datasource` | Add one DataSource |
| `rdl_remove_datasource` | Remove one DataSource |
| `rdl_rename_datasource` | Rename one DataSource |
| `rdl_set_datasource_connect_string` | Set ConnectString on one DataSource |
| `rdl_add_dataset` | Add one DataSet |
| `rdl_remove_dataset` | Remove one DataSet |
| `rdl_rename_dataset` | Rename one DataSet |
| `rdl_add_dataset_field` | Add one field to a DataSet |
| `rdl_clear_dataset_fields` | Clear all fields from a DataSet |
| `rdl_clear_dataset_filters` | Remove Filters from a DataSet |
| `rdl_set_dataset_command_text` | Update CommandText on one DataSet |
| `rdl_add_parameter` | Add one ReportParameter |
| `rdl_remove_parameter` | Remove one ReportParameter |
| `rdl_rebuild_tablix` | Rebuild a Tablix from a JSON spec |
| `rdl_tablix_set_cell` | Set one cell value at (row, col) |
| `rdl_tablix_add_row` | Append or insert one row |
| `rdl_tablix_remove_row` | Remove one row by index |
| `rdl_tablix_add_column` | Append or insert one column |
| `rdl_tablix_remove_column` | Remove one column by index |
| `rdl_fix_encoding` | Fix UTF-8 BOM and CRLF line endings |
| `rdl_register` | Register an RDL in a .rptproj file |
| `rdl_validate` | Validate RDL structure; check `data.pass` |

**Breaking change from v1.x:** `rdl_manage_datasources`, `rdl_manage_datasets`, and `rdl_manage_parameters` were removed. Use the atomic `rdl_add_*`, `rdl_remove_*`, `rdl_rename_*`, and `rdl_set_*` tools instead.
