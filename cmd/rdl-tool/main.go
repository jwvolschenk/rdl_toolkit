package main

import (
	"fmt"
	"os"

	"github.com/rdl-toolkit/internal/cli"
	rdlmcp "github.com/rdl-toolkit/internal/mcp"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--mcp" {
		if err := rdlmcp.Serve(); err != nil {
			fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
