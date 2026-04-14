package tools

import (
	"context"

	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

const DevDeferredBuiltinSampleCanonicalName = "dev_deferred_builtin_sample"

type DevDeferredBuiltinSampleInput struct {
	Note string `json:"note,omitempty" jsonschema:"description=optional note to echo back for debugging"`
}

type DevDeferredBuiltinSampleOutput struct {
	Tool    string `json:"tool"`
	Note    string `json:"note,omitempty"`
	Message string `json:"message"`
}

func NewDevDeferredBuiltinSampleEntry() (CatalogEntry, error) {
	sampleTool, err := toolutils.InferTool(
		DevDeferredBuiltinSampleCanonicalName,
		"Dev-only experimental deferred builtin sample. Safe and side-effect free; used to exercise deferred surface runtime paths.",
		func(ctx context.Context, input DevDeferredBuiltinSampleInput) (DevDeferredBuiltinSampleOutput, error) {
			return DevDeferredBuiltinSampleOutput{
				Tool:    DevDeferredBuiltinSampleCanonicalName,
				Note:    input.Note,
				Message: "dev deferred builtin sample executed",
			}, nil
		},
	)
	if err != nil {
		return CatalogEntry{}, err
	}
	return CatalogEntry{
		CanonicalName: DevDeferredBuiltinSampleCanonicalName,
		Source:        ToolSourceBuiltin,
		Kind:          ToolKindAction,
		ToolClass:     ToolClassBuiltin,
		DeferReason:   "dev_experimental",
		ShouldDefer:   true,
		AlwaysLoad:    false,
		IsMcp:         false,
		PermissionSpec: PermissionSpec{
			AllowSearch:  true,
			AllowExecute: true,
		},
		Tool: sampleTool,
	}, nil
}
