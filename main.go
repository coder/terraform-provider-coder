package main

import (
	"flag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/coder/terraform-provider-coder/v2/provider"
)

// Run the docs generation tool, check its repository for more information on how it works and how docs
// can be customized.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

func main() {
	debug := flag.Bool("debug", false, "Enable debug mode for the provider")
	flag.Parse()

	opts := &plugin.ServeOpts{
		Debug:        *debug,
		ProviderAddr: "registry.terraform.io/coder/coder",
		ProviderFunc: provider.New,
	}

	servePprof()
	plugin.Serve(opts)
}
