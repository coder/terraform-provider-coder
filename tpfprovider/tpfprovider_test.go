package tpfprovider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"coder": providerserver.NewProtocol6WithError(NewFrameworkProvider()()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
}
