package cozeplugin

import (
	"context"
	"fmt"
	"io"
	"runtime/debug"

	"github.com/cloudwego/kitex/client/callopt/streamcall"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/kite/kitex/client/callopt"
	"code.byted.org/kite/kitutil"
	"code.byted.org/overpass/ocean_cloud_plugin/kitex_gen/ocean/cloud/plugin"
	"code.byted.org/overpass/ocean_cloud_plugin/kitex_gen/ocean/cloud/plugin/pluginservice"

	"code.byted.org/flow/eino/components/tool"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flow/eino/utils/safe"
)

var kitexClient pluginservice.Client
var streamClient pluginservice.StreamClient

const cozePluginPSM = "ocean.cloud.plugin"

func init() {
	cli, err := pluginservice.NewClient(cozePluginPSM)
	if err != nil {
		panic(fmt.Sprintf("init coze plugin client error: %v", err.Error()))
	}
	kitexClient = cli

	streamCli, err := pluginservice.NewStreamClient(cozePluginPSM)
	if err != nil {
		panic(fmt.Sprintf("init coze plugin streamclient error: %v", err.Error()))
	}
	streamClient = streamCli
}

var defaultENV string
var defaultCallOpts []callopt.Option
var defaultStreamCallOpts []streamcall.Option

// InitENV all coze plugin request will use the env you set here, if K_ENV haven't been set to ctx.
func InitENV(e string) {
	defaultENV = e
}

// InitCallOpts all coze plugin request will use the options you set here.
func InitCallOpts(opts ...callopt.Option) {
	defaultCallOpts = opts
}

// InitStreamCallOpts all stream coze plugin request will use the options you set here.
func InitStreamCallOpts(opts ...streamcall.Option) {
	defaultStreamCallOpts = opts
}

type Config struct {
	// If API isn't nil, sdk will use API directly instead of using PluginID and APIID to query.
	API *plugin.API

	PluginID int64
	APIID    int64
}

// NewTool you can get a InvokableTool or StreamableTool from this function, and the decision of which type of tool to get depends entirely on the tool information obtained from coze.
func NewTool(ctx context.Context, config *Config) (tool.BaseTool, error) {
	var err error
	api := config.API
	if api == nil {
		api, err = getCozePluginInfo(ctx, config.PluginID, config.APIID)
		if err != nil {
			return nil, fmt.Errorf("get coze plugin info error: %w", err)
		}
	}

	if api.RunMode == plugin.RunMode_Streaming {
		return &streamCozePlugin{
			API: config.API,

			PluginName: api.PluginName,
			PluginID:   api.PluginId,
			APIName:    api.Name,
			APIID:      api.ApiID,
		}, nil
	}
	return &cozePlugin{
		API: config.API,

		PluginName: api.PluginName,
		PluginID:   api.PluginId,
		APIName:    api.Name,
		APIID:      api.ApiID,
	}, nil
}

func getCozePluginInfo(ctx context.Context, pluginID int64, apiID int64) (*plugin.API, error) {
	trueValue := true
	req := &plugin.GetPluginListRequest{
		PluginIds:    []int64{pluginID},
		NeedAPIs:     &trueValue,
		NeedWorkflow: &trueValue,
		NeedAPIID:    &trueValue,
	}
	if _, ok := kitutil.GetCtxEnv(ctx); !ok && defaultENV != "" {
		ctx = kitutil.NewCtxWithEnv(ctx, defaultENV)
	}
	resp, err := kitexClient.GetPluginList(ctx, req, defaultCallOpts...)
	if err != nil {
		return nil, err
	}

	return apiFilter(resp.GetPluginListData.PluginInfos, pluginID, apiID)
}

func apiFilter(pluginInfos []*plugin.PluginInfo, pluginID, apiID int64) (*plugin.API, error) {
	for _, pInfo := range pluginInfos {
		if pInfo.Id != pluginID {
			continue
		}
		for _, apiInfo := range pInfo.APIs {
			if apiInfo.ApiID == apiID {
				return apiInfo, nil
			}
		}
	}
	return nil, fmt.Errorf("can't find coze plugin[%d] api[%d]", pluginID, apiID)
}

type cozePlugin struct {
	API *plugin.API

	PluginName string
	PluginID   int64
	APIName    string
	APIID      int64
}

func (c *cozePlugin) Info(ctx context.Context) (*schema.ToolInfo, error) {
	var err error
	api := c.API
	if api == nil {
		api, err = getCozePluginInfo(ctx, c.PluginID, c.APIID)
		if err != nil {
			return nil, fmt.Errorf("get coze plugin info error: %w", err)
		}
	}

	param := constructParams(api.Parameters)

	return &schema.ToolInfo{
		Name:        api.Name,
		Desc:        api.Desc,
		ParamsOneOf: schema.NewParamsOneOfByParams(param),
	}, nil
}

func (c *cozePlugin) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (output string, err error) {
	plgOpt := tool.GetImplSpecificOptions(&pluginOptions{}, opts...)

	cbm, cbmOK := callbacks.ManagerFromCtx(ctx)

	defer func() {
		if err != nil && cbmOK {
			cbm.OnError(ctx, err)
		}
	}()

	if plgOpt.InputModifier != nil {
		argumentsInJSON, err = plgOpt.InputModifier(ctx, argumentsInJSON)
		if err != nil {
			return "", fmt.Errorf("input modifier execute fail, plugin name:%s, api name: %s, error: %w", c.PluginName, c.APIName, err)
		}
	}

	req := &plugin.DoActionRequest{
		PluginID:   c.PluginID,
		PluginName: c.PluginName,
		APIName:    c.APIName,
		Parameters: argumentsInJSON,
		UserID:     plgOpt.UserID,
		DeviceID:   plgOpt.DeviceID,
		Ext:        plgOpt.Extra,
	}

	if _, ok := kitutil.GetCtxEnv(ctx); !ok && defaultENV != "" {
		ctx = kitutil.NewCtxWithEnv(ctx, defaultENV)
	}

	if cbmOK {
		ctx = cbm.OnStart(ctx, &tool.CallbackInput{
			ArgumentsInJSON: argumentsInJSON,
			Extra: map[string]any{
				callbackExtraInputDetail: &InputExtraDetail{
					API:      c.API,
					UserID:   req.UserID,
					DeviceID: req.DeviceID,
					Ext:      req.Ext,
				},
			},
		})
	}

	resp, err := kitexClient.DoAction(ctx, req, append(defaultCallOpts, plgOpt.callOpts...)...)
	if err != nil {
		return "", fmt.Errorf("request to execute coze plugin fail: %w", err)
	}

	if !resp.Success {
		return "", fmt.Errorf("execute coze plugin fail, plugin:%+v, input:%s, code:%d, message:%s", *c, argumentsInJSON, resp.BaseResp.StatusCode, resp.BaseResp.StatusMessage)
	}

	if cbmOK {
		ctx = cbm.OnEnd(ctx, &tool.CallbackOutput{
			Response: resp.Resp,
			Extra: map[string]any{
				callbackExtraOutputDetail: &OutputExtraDetail{
					Tokens:              resp.Tokens,
					Cost:                resp.Cost,
					MockHitStatus:       resp.MockHitStatus,
					PluginInterruptData: resp.PluginInterruptData,
					SSEEventID:          "",
				},
			},
		})
	}

	return resp.Resp, nil
}

func (c *cozePlugin) GetType() string {
	return typ
}

type streamCozePlugin struct {
	API *plugin.API

	PluginName string
	PluginID   int64
	APIName    string
	APIID      int64
}

func (s *streamCozePlugin) Info(ctx context.Context) (*schema.ToolInfo, error) {
	var err error
	api := s.API
	if api == nil {
		api, err = getCozePluginInfo(ctx, s.PluginID, s.APIID)
		if err != nil {
			return nil, fmt.Errorf("get coze plugin info error: %w", err)
		}
	}

	param := constructParams(api.Parameters)

	return &schema.ToolInfo{
		Name:        api.Name,
		Desc:        api.Desc,
		ParamsOneOf: schema.NewParamsOneOfByParams(param),
	}, nil
}

func (s *streamCozePlugin) StreamableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (sr *schema.StreamReader[string], err error) {
	cbm, cbmOK := callbacks.ManagerFromCtx(ctx)

	defer func() {
		if err != nil && cbmOK {
			cbm.OnError(ctx, err)
		}
	}()

	plgOpt := tool.GetImplSpecificOptions(&pluginOptions{}, opts...)

	if plgOpt.InputModifier != nil {
		argumentsInJSON, err = plgOpt.InputModifier(ctx, argumentsInJSON)
		if err != nil {
			return nil, fmt.Errorf("input modifier execute fail, plugin name:%s, api name: %s, error: %w", s.PluginName, s.APIName, err)
		}
	}

	req := &plugin.StreamDoActionRequest{
		PluginID:   s.PluginID,
		PluginName: s.PluginName,
		APIName:    s.APIName,
		Parameters: argumentsInJSON,
		UserID:     plgOpt.UserID,
		DeviceID:   plgOpt.DeviceID,
		Ext:        plgOpt.Extra,
	}

	if _, ok := kitutil.GetCtxEnv(ctx); !ok && defaultENV != "" {
		ctx = kitutil.NewCtxWithEnv(ctx, defaultENV)
	}

	if cbmOK {
		ctx = cbm.OnStart(ctx, &tool.CallbackInput{
			ArgumentsInJSON: argumentsInJSON,
			Extra: map[string]any{
				callbackExtraInputDetail: &InputExtraDetail{
					API:      s.API,
					UserID:   req.UserID,
					DeviceID: req.DeviceID,
					Ext:      req.Ext,
				},
			},
		})
	}

	resp, err := streamClient.StreamDoAction(ctx, req, append(defaultStreamCallOpts, plgOpt.streamCallOpts...)...)
	if err != nil {
		return nil, fmt.Errorf("request to stream execute coze plugin fail: %w", err)
	}

	rawSr, sw := schema.Pipe[*plugin.StreamDoActionResponse](1)
	go func() {
		defer func() {
			panicErr := recover()
			if panicErr != nil {
				sw.Send(nil, safe.NewPanicErr(panicErr, debug.Stack()))
			}
			sw.Close()
			_ = resp.Close()
		}()

		for {
			chunk, err := resp.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				sw.Send(nil, fmt.Errorf("execute coze plugin stream response error: %w", err))
				break
			}
			if !chunk.Success {
				sw.Send(nil, fmt.Errorf("stream execute coze plugin fail"))
				break
			}
			if chunk.Resp == nil {
				sw.Send(nil, fmt.Errorf("stream execute coze plugin success, but Resp is empty"))
				break
			}
			if chunk.Resp.SSEData == nil {
				sw.Send(nil, fmt.Errorf("stream execute coze plugin success, but SSEData is empty"))
				break
			}
			if chunk.IsFinish {
				break
			}
		}
	}()

	rawStreamArr := make([]*schema.StreamReader[*plugin.StreamDoActionResponse], 2)
	if cbmOK {
		rawStreamArr = rawSr.Copy(2)
	} else {
		rawStreamArr[0] = rawSr
	}

	sr = schema.StreamReaderWithConvert(rawStreamArr[0], func(src *plugin.StreamDoActionResponse) (string, error) {
		return src.Resp.GetSSEData(), nil
	})

	if cbmOK {
		cbStream := schema.StreamReaderWithConvert(rawStreamArr[1], func(src *plugin.StreamDoActionResponse) (callbacks.CallbackOutput, error) {
			return &tool.CallbackOutput{
				Response: src.Resp.GetSSEData(),
				Extra: map[string]any{
					callbackExtraOutputDetail: &OutputExtraDetail{
						Tokens:              src.Tokens,
						Cost:                src.Cost,
						MockHitStatus:       nil,
						PluginInterruptData: src.PluginInterruptData,
						SSEEventID:          src.SSEEventId,
					},
				},
			}, nil
		})

		_ = cbm.OnEndWithStreamOutput(ctx, cbStream)
	}

	return sr, nil
}

func (s *streamCozePlugin) GetType() string {
	return typ
}
