package server

import (
	"fmt"
	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/http"
)

type Content struct {
	Title       string  `json:"title" jsonschema:"required,description=The title to submit"`
	Description *string `json:"description" jsonschema:"description=The description to submit"`
}
type MyFunctionsArguments struct {
	Submitter string  `json:"submitter" jsonschema:"required,description=The name of the thing calling this tool (openai, google, claude, etc)"`
	Content   Content `json:"content" jsonschema:"required,description=The content of the message"`
}

func (ws *WebServer) StartMcpServer() error {
	transport := http.NewHTTPTransport("/mcp")
	transport.WithAddr(":8082")

	// Create server with the HTTP transport
	server := mcp.NewServer(transport)

	err := server.RegisterTool("search", "Search for a product", func(arguments MyFunctionsArguments) (*mcp.ToolResponse, error) {

		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Hello, %s server!", arguments.Submitter))), nil
	})
	if err != nil {
		return err
	}

	err = server.Serve()
	return err

}
