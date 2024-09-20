package cozeplugin

import (
	"code.byted.org/overpass/ocean_cloud_plugin/kitex_gen/ocean/cloud/plugin"

	"code.byted.org/flow/eino/schema"
)

func constructParams(params []*plugin.Parameter) map[string]*schema.ParameterInfo {
	if len(params) == 0 {
		return map[string]*schema.ParameterInfo{}
	}

	ret := make(map[string]*schema.ParameterInfo, len(params))

	for _, p := range params {
		if p == nil {
			continue
		}

		pi := &schema.ParameterInfo{
			Type:     schema.DataType(p.Type),
			Desc:     p.Desc,
			Required: p.Required,
		}

		if p.Type == "array" && len(p.SubParameters) == 0 {
			pi.ElemInfo = &schema.ParameterInfo{Type: schema.String}
		}

		if len(p.SubParameters) > 0 {
			subParameters := constructParams(p.SubParameters)
			if p.Type == "array" {
				pi.ElemInfo = &schema.ParameterInfo{
					Type:      schema.Object,
					SubParams: subParameters,
				}
			} else {
				pi.SubParams = subParameters
			}
		}

		ret[p.Name] = pi
	}

	return ret
}
