// fornaxplugin implements the plugin on fornax as eino tool interface.
// more information please refer to https://bytedance.larkoffice.com/wiki/XyDewau0ViBRvJkEKWIcsNcZnEd.
// e.g.
//
//	tool, err := fornaxplugin.NewPluginTool(ctx, &fornaxplugin.PluginToolConfig{
//		ToolID: 321123,
//	})
//	if err != nil {...}
package fornaxplugin

import (
	"context"
	"fmt"

	"code.byted.org/flow/eino/components/tool"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flowdevops/fornax_sdk"
	"code.byted.org/gopkg/logs/v2"
	"code.byted.org/overpass/flow_devops_plugin/kitex_gen/flow/devops/plugin/domain/definition"
)

// ToolOutputConverter converts the output of the tool to the expected format.
// in eino, tool node output is always string.
type ToolOutputConverter func(ctx context.Context, toolOut string, toolOutDef *definition.Definition) (content string, err error)

// PluginToolConfig is the configuration for the plugin tool.
type PluginToolConfig struct {
	ToolID int64 `json:"tool_id"`

	Converter ToolOutputConverter `json:"-"`
}

// NewPluginTool creates a new plugin tool.
func NewPluginTool(ctx context.Context, conf *PluginToolConfig) (tool.InvokableTool, error) {

	toolID := conf.ToolID
	_, err := getToolInfo(ctx, toolID)
	if err != nil {
		return nil, err
	}
	return &invokablePlugin{
		toolID:    toolID,
		converter: conf.Converter,
	}, nil
}

type invokablePlugin struct {
	toolID int64

	converter ToolOutputConverter
}

func (i *invokablePlugin) Info(ctx context.Context) (*schema.ToolInfo, error) {
	pi, err := getToolInfo(ctx, i.toolID)
	if err != nil {
		logs.Errorf("[Fornax Plugin] get tool info failed, toolID: %d, err: %v", i.toolID, err)
		return nil, fmt.Errorf("[Fornax Plugin]get tool info failed: %w", err)
	}
	return pi, nil
}

func (i *invokablePlugin) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	plgOpt := tool.GetImplSpecificOptions(&pluginOptions{}, opts...)

	if nCtx, err := fornax_sdk.InjectTrace(ctx); err == nil {
		ctx = nCtx
	}
	content, def, err := executeTool(ctx, i.toolID, argumentsInJSON, plgOpt.callOpts...)
	if err != nil {
		return "", fmt.Errorf("[Fornax Plugin]execute tool failed: %w", err)
	}
	if i.converter != nil {
		content, err = i.converter(ctx, content, def)
		if err != nil {
			return "", fmt.Errorf("[Fornax Plugin]convert tool output failed: %w", err)
		}
	}

	return content, nil
}
