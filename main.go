package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/coder/terraform-provider-coder/v2/provider"
)

// Run the docs generation tool, check its repository for more information on how it works and how docs
// can be customized.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

func main() {
	servePprof()
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: provider.New,
	})
}
