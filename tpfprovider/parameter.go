package tpfprovider

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"strings"

	"github.com/coder/terraform-provider-coder/v2/provider"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type parameterDataSourceModel struct {
	Name  types.String  `tfsdk:"name"`
	Type  types.Dynamic `tfsdk:"type"`
	Value types.Dynamic `tfsdk:"value"`
}

type parameterDataSource struct{}

func NewParameterDataSource() datasource.DataSource {
	return &parameterDataSource{}
}

func (m *parameterDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "coder_parameter"
}

func (m *parameterDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"type": schema.DynamicAttribute{
				Required: true,
			},
			"value": schema.DynamicAttribute{
				Computed: true,
			},
		},
	}
}

func (m *parameterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data parameterDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Name.IsNull() {
		resp.Diagnostics.AddError("name is required", "name")
		return
	}

	ds := data.Name.ValueString()
	parameterEnv := provider.ParameterEnvironmentVariable(ds)
	rawValue, ok := os.LookupEnv(parameterEnv)
	if !ok {
		resp.Diagnostics.AddError("parameter not found", "name")
		return
	}

	switch data.Type.UnderlyingValue().(type) {
	case types.String:
		data.Value = types.DynamicValue(types.StringValue(rawValue))
	case types.Number:
		// convert the raw value to a number
		var floatVal float64
		if err := json.NewDecoder(strings.NewReader(rawValue)).Decode(&floatVal); err != nil {
			resp.Diagnostics.AddError("failed to parse value as number", "value")
			return
		}

		data.Value = types.DynamicValue(types.NumberValue(big.NewFloat(floatVal)))
	case types.Bool:
		// convert the raw value to a bool
		var boolVal bool
		if err := json.NewDecoder(strings.NewReader(rawValue)).Decode(&boolVal); err != nil {
			resp.Diagnostics.AddError("failed to parse value as bool", "value")
			return
		}
		data.Value = types.DynamicValue(types.BoolValue(boolVal))
	case types.List:
		// TODO: handle list
		resp.Diagnostics.AddError("TODO: list type not supported", "type")
		return
	case types.Map:
		// TODO: handle map
		resp.Diagnostics.AddError("TODO: map type not supported", "type")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
