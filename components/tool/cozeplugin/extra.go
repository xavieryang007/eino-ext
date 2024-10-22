package cozeplugin

import (
	"code.byted.org/flow/eino/components/tool"
	"code.byted.org/overpass/ocean_cloud_plugin/kitex_gen/ocean/cloud/plugin"
	"code.byted.org/overpass/ocean_cloud_plugin/kitex_gen/ocean/cloud/plugin_interrupt"
)

const (
	typ                       = "CozePlugin"
	callbackExtraInputDetail  = "coze_plugin_input_detail"  // *InputExtraDetail
	callbackExtraOutputDetail = "coze_plugin_output_detail" // *OutputExtraDetail
)

type InputExtraDetail struct {
	API      *plugin.API       `json:"API,omitempty"`
	UserID   int64             `json:"UserID,omitempty"`
	DeviceID *int64            `json:"DeviceID,omitempty"`
	Ext      map[string]string `json:"Ext,omitempty"`
}

type OutputExtraDetail struct {
	Tokens              *int64                                `json:"Tokens,omitempty"`
	Cost                *string                               `json:"Cost,omitempty"`
	MockHitStatus       *string                               `json:"MockHitStatus,omitempty"`
	PluginInterruptData *plugin_interrupt.PluginInterruptData `json:"PluginInterruptData,omitempty"`
	SSEEventID          string                                `json:"SSEEventID,omitempty"` // only available in stream
}

func GetCallbackInputExtraDetail(ci *tool.CallbackInput) (*InputExtraDetail, bool) {
	if ci == nil {
		return nil, false
	}

	detail, ok := ci.Extra[callbackExtraInputDetail].(*InputExtraDetail)

	return detail, ok
}

func GetCallbackOutputExtraDetail(co *tool.CallbackOutput) (*OutputExtraDetail, bool) {
	if co == nil {
		return nil, false
	}

	detail, ok := co.Extra[callbackExtraOutputDetail].(*OutputExtraDetail)

	return detail, ok
}
