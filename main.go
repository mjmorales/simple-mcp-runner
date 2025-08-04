// simple-mcp-runner is a production-ready Model Context Protocol (MCP) server
// that provides Language Learning Models with a safe interface to discover
// and execute system commands on the local machine.
package main

import "github.com/mjmorales/simple-mcp-runner/cmd"

func main() {
	cmd.Execute()
}
