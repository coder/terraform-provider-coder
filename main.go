package main

import (
	"context"
	"flag"
	"log"
	// Embed timezone data for use in environments that may not have the
	// timezone database available (e.g. scratch Docker images).
	_ "time/tzdata"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/coder/terraform-provider-coder/v2/provider"
)

// Run the docs generation tool, check its repository for more information on how it works and how docs
// can be customized.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

func main() {
	debug := flag.Bool("debug", false, "Enable debug mode for the provider")
	flag.Parse()

	server, err := provider.NewMux(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	opts := &plugin.ServeOpts{
		Debug:            *debug,
		ProviderAddr:     "registry.terraform.io/coder/coder",
		GRPCProviderFunc: func() tfprotov5.ProviderServer { return server },
	}

	servePprof()
	plugin.Serve(opts)
}
