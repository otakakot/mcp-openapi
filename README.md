# MCP OpenAPI Server

A Model Context Protocol (MCP) server that provides access to OpenAPI specification details.

## Features

This MCP server loads an OpenAPI 3.0 specification and provides a tool to retrieve API details by operation ID.

### Tool: `get_api_details`

Retrieves detailed information about a specific API operation from the loaded OpenAPI specification.

**Parameters:**

- `operation_id` (string, required): The operation ID to get details for

### Settings

```json
{
  "servers": {
    "openapi": {
      "type": "stdio",
      "command": "docker",
      "args": ["run", "-i", "--rm", "-v", "/${workspaceFolder}/openapi.yaml:/openapi.yaml", "mcp-openapi"],
    }
  }
}
```
