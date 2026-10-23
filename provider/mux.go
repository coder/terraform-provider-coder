package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	framework "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
)

// NewMux returns the protocol 5 provider serving SDK resources alongside
// Framework data sources that require dynamic attribute types.
func NewMux(ctx context.Context) (tfprotov5.ProviderServer, error) {
	mux, err := tf5muxserver.NewMuxServer(ctx,
		New().GRPCProvider,
		providerserver.NewProtocol5(&templateOutputsProvider{}),
	)
	if err != nil {
		return nil, err
	}
	return mux.ProviderServer(), nil
}

type templateOutputsProvider struct{}

func (*templateOutputsProvider) Metadata(_ context.Context, _ framework.MetadataRequest, resp *framework.MetadataResponse) {
	resp.TypeName = "coder"
}

func (*templateOutputsProvider) Schema(_ context.Context, _ framework.SchemaRequest, resp *framework.SchemaResponse) {
	// Mux requires identical provider schemas. The SDK side retains ownership
	// of URL validation, environment defaults, and resource configuration.
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Optional:    true,
				Description: "The URL to access Coder.",
			},
		},
	}
}

func (*templateOutputsProvider) Configure(context.Context, framework.ConfigureRequest, *framework.ConfigureResponse) {
}

func (*templateOutputsProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource { return &templateOutputsDataSource{} },
	}
}

func (*templateOutputsProvider) Resources(context.Context) []func() resource.Resource {
	return nil
}
