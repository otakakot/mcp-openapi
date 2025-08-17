package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type OpenAPIServer struct {
	doc *openapi3.T
}

func NewOpenAPIServer() (*OpenAPIServer, error) {
	openAPIPath, err := resolveOpenAPIPath(".")
	if err != nil {
		return nil, fmt.Errorf("failed to find openapi.yaml in current directory: %v", err)
	}

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(openAPIPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI specification from %s: %v", openAPIPath, err)
	}

	return &OpenAPIServer{
		doc: doc,
	}, nil
}

type GetAPIDetailsParams struct {
	OperationID string `json:"operation_id" jsonschema:"the operationId to get details for"`
}

type APIDetails struct {
	OperationID string              `json:"operation_id"`
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	Summary     string              `json:"summary,omitempty"`
	Description string              `json:"description,omitempty"`
	Parameters  []ParameterDetails  `json:"parameters,omitempty"`
	RequestBody *RequestBodyDetails `json:"request_body,omitempty"`
	Responses   map[string]any      `json:"responses,omitempty"`
}

type ParameterDetails struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
	Schema      any    `json:"schema,omitempty"`
}

type RequestBodyDetails struct {
	Description string                 `json:"description,omitempty"`
	Required    bool                   `json:"required"`
	Content     map[string]interface{} `json:"content,omitempty"`
}

func resolveOpenAPIPath(path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("path does not exist: %v", err)
	}

	if info.IsDir() {
		openAPIFile := filepath.Join(path, "openapi.yaml")
		if _, err := os.Stat(openAPIFile); err != nil {
			openAPIFile = filepath.Join(path, "openapi.yml")
			if _, err := os.Stat(openAPIFile); err != nil {
				return "", fmt.Errorf("openapi.yaml or openapi.yml not found in directory: %s", path)
			}
		}
		return openAPIFile, nil
	}

	return path, nil
}

func (s *OpenAPIServer) GetAPIDetails(
	ctx context.Context,
	cc *mcp.ServerSession,
	params *mcp.CallToolParamsFor[GetAPIDetailsParams],
) (*mcp.CallToolResultFor[any], error) {
	var apiDetails *APIDetails

	for path, pathItem := range s.doc.Paths.Map() {
		operations := map[string]*openapi3.Operation{
			"GET":     pathItem.Get,
			"POST":    pathItem.Post,
			"PUT":     pathItem.Put,
			"DELETE":  pathItem.Delete,
			"PATCH":   pathItem.Patch,
			"HEAD":    pathItem.Head,
			"OPTIONS": pathItem.Options,
			"TRACE":   pathItem.Trace,
		}

		for method, operation := range operations {
			if operation != nil && operation.OperationID == params.Arguments.OperationID {
				apiDetails = &APIDetails{
					OperationID: operation.OperationID,
					Method:      method,
					Path:        path,
					Summary:     operation.Summary,
					Description: operation.Description,
				}

				if operation.Parameters != nil {
					for _, paramRef := range operation.Parameters {
						if paramRef.Value != nil {
							param := paramRef.Value
							paramDetails := ParameterDetails{
								Name:        param.Name,
								In:          param.In,
								Required:    param.Required,
								Description: param.Description,
							}
							if param.Schema != nil && param.Schema.Value != nil {
								paramDetails.Schema = param.Schema.Value
							}
							apiDetails.Parameters = append(apiDetails.Parameters, paramDetails)
						}
					}
				}

				if operation.RequestBody != nil && operation.RequestBody.Value != nil {
					reqBody := operation.RequestBody.Value
					apiDetails.RequestBody = &RequestBodyDetails{
						Description: reqBody.Description,
						Required:    reqBody.Required,
						Content:     make(map[string]interface{}),
					}
					for mediaType, mediaTypeObj := range reqBody.Content {
						apiDetails.RequestBody.Content[mediaType] = mediaTypeObj
					}
				}

				if operation.Responses != nil {
					apiDetails.Responses = make(map[string]interface{})
					for status, responseRef := range operation.Responses.Map() {
						if responseRef.Value != nil {
							apiDetails.Responses[status] = responseRef.Value
						}
					}
				}

				break
			}
		}
		if apiDetails != nil {
			break
		}
	}

	if apiDetails == nil {
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Operation with ID '%s' not found in the OpenAPI specification", params.Arguments.OperationID),
				},
			},
		}, nil
	}

	jsonData, err := json.MarshalIndent(apiDetails, "", "  ")
	if err != nil {
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Failed to marshal API details: %v", err),
				},
			},
		}, nil
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonData),
			},
		},
	}, nil
}

func main() {
	openAPIServer, err := NewOpenAPIServer()
	if err != nil {
		log.Fatalf("Failed to initialize OpenAPI server: %v", err)
	}

	server := mcp.NewServer(
		&mcp.Implementation{Name: "openapi-reader", Version: "v1.0.0"},
		nil,
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "get_api_details",
			Description: "Get API details by operationId from OpenAPI specification loaded at startup",
		},
		openAPIServer.GetAPIDetails,
	)

	if err := server.Run(context.Background(), mcp.NewStdioTransport()); err != nil {
		log.Fatal(err)
	}
}
