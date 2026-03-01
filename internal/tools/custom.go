package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

// NewCustomTool is a helper for creating custom tools using Go generics.
// It wraps utils.InferTool to provide a simple API for registering tools
// with automatically inferred JSON schema from the input struct's tags.
//
// Usage:
//
//	type MyInput struct {
//	    Query string `json:"query" jsonschema:"description=the search query"`
//	}
//	type MyOutput struct {
//	    Result string `json:"result"`
//	}
//	myTool, err := tools.NewCustomTool[MyInput, MyOutput](
//	    "my_tool",
//	    "Description of my tool",
//	    func(ctx context.Context, input MyInput) (MyOutput, error) {
//	        return MyOutput{Result: "hello"}, nil
//	    },
//	)
func NewCustomTool[I any, O any](name, description string,
	fn func(context.Context, I) (O, error)) (tool.BaseTool, error) {

	return toolutils.InferTool(name, description, fn)
}
