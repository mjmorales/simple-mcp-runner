#!/usr/bin/env python3
"""
Simple test script to demonstrate MCP server functionality.
This simulates how an LLM would interact with the MCP server.
"""

import json
import subprocess
import sys

def send_mcp_request(request):
    """Send a JSON-RPC request to the MCP server via stdin."""
    proc = subprocess.Popen(
        ['./simple-mcp-runner', 'run'],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True
    )
    
    # Send request
    proc.stdin.write(json.dumps(request) + '\n')
    proc.stdin.flush()
    
    # Read response
    response_line = proc.stdout.readline()
    
    # Terminate process
    proc.terminate()
    proc.wait()
    
    return json.loads(response_line)

def main():
    # Example 1: Discover commands
    print("=== Testing Command Discovery ===")
    discover_request = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/call",
        "params": {
            "name": "discover_commands",
            "arguments": {
                "pattern": "echo",
                "max_results": 5
            }
        }
    }
    
    print("Request:", json.dumps(discover_request, indent=2))
    # Note: This is a simplified example. Real MCP interaction requires proper protocol handling
    print("\nThis example demonstrates the request format for MCP tools.")
    
    # Example 2: Execute command
    print("\n\n=== Testing Command Execution ===")
    execute_request = {
        "jsonrpc": "2.0",
        "id": 2,
        "method": "tools/call",
        "params": {
            "name": "execute_command",
            "arguments": {
                "command": "echo",
                "args": ["Hello from MCP!"],
                "timeout": "5s"
            }
        }
    }
    
    print("Request:", json.dumps(execute_request, indent=2))
    print("\nExpected result: stdout containing 'Hello from MCP!'")

if __name__ == "__main__":
    main()