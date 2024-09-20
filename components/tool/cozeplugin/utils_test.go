package cozeplugin

import (
	"reflect"
	"testing"

	"code.byted.org/overpass/ocean_cloud_plugin/kitex_gen/ocean/cloud/plugin"

	"code.byted.org/flow/eino/schema"
)

func TestConstructParam(t *testing.T) {
	params := []*plugin.Parameter{
		{
			Name:     "name1",
			Desc:     "description1",
			Type:     "string",
			Required: true,
		},
		{
			Name:     "name2",
			Desc:     "description2",
			Type:     "array",
			Required: true,
			SubParameters: []*plugin.Parameter{
				{
					Name: "name2_1",
					Type: "integer",
				},
			},
		},
		{
			Name:     "name3",
			Desc:     "description3",
			Type:     "object",
			Required: false,
			SubParameters: []*plugin.Parameter{
				{
					Name: "name3_1",
					Desc: "description3_1",
					Type: "string",
				},
				{
					Name: "name3_2",
					Desc: "description3_2",
					Type: "integer",
				},
			},
		},
		{
			Name:     "name4",
			Desc:     "description4",
			Type:     "array",
			Required: true,
		},
	}

	expected := map[string]*schema.ParameterInfo{
		"name1": {
			Type:     schema.String,
			Desc:     "description1",
			Required: true,
		},
		"name2": {
			Type:     schema.Array,
			Desc:     "description2",
			ElemInfo: &schema.ParameterInfo{Type: schema.Object, SubParams: map[string]*schema.ParameterInfo{"name2_1": {Type: schema.Integer}}},
			Required: true,
		},
		"name3": {
			Type: schema.Object,
			Desc: "description3",
			SubParams: map[string]*schema.ParameterInfo{
				"name3_1": {
					Desc: "description3_1",
					Type: schema.String,
				},
				"name3_2": {
					Desc: "description3_2",
					Type: schema.Integer,
				},
			},
			Required: false,
		},
		"name4": {
			Type:     schema.Array,
			Desc:     "description4",
			ElemInfo: &schema.ParameterInfo{Type: "string"},
			Required: true,
		},
	}
	s := constructParams(params)
	if !reflect.DeepEqual(s, expected) {
		t.Fatalf("ConstructParam error, returned %+v, expected %+v", s, expected)
	}
}
