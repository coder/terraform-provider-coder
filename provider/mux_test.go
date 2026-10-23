package provider_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestMuxProviderURL(t *testing.T) {
	// Both provider implementations must accept the SDK's URL configuration.
	t.Setenv("CODER_WORKSPACE_BUILD_ID", "")
	t.Setenv("CODER_TEMPLATE_OUTPUTS", "{}")
	t.Setenv("CODER_TEMPLATE_RESOURCE_REVISION_ID", "")
	for _, tc := range []struct {
		name, envURL, config, wantURL, wantError string
		unset                                    bool
	}{
		{name: "Default", unset: true, wantURL: "https://mydeployment.coder.com"},
		{name: "Environment", envURL: "https://environment.example:8443", wantURL: "https://environment.example:8443"},
		{name: "Explicit", envURL: "https://environment.example", config: `url = "https://configured.example"`, wantURL: "https://configured.example"},
		{name: "EmptyEnvironment", wantURL: "https://mydeployment.coder.com"},
		{name: "EmptyExplicit", envURL: "https://environment.example", config: `url = ""`, wantError: "CODER_AGENT_URL must not be empty"},
		{name: "InvalidExplicit", config: `url = "http://%"`, wantError: "invalid URL escape"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CODER_AGENT_URL", tc.envURL)
			if tc.unset {
				require.NoError(t, os.Unsetenv("CODER_AGENT_URL"))
			}
			step := resource.TestStep{
				Config: fmt.Sprintf(`
provider "coder" { %s }
data "coder_workspace" "me" {}
data "coder_template_outputs" "current" {}
output "empty_outputs" { value = data.coder_template_outputs.current.outputs == {} }
`, tc.config),
			}
			if tc.wantError != "" {
				step.ExpectError = regexp.MustCompile(tc.wantError)
			} else {
				step.Check = resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.coder_workspace.me", "access_url", tc.wantURL),
					resource.TestCheckOutput("empty_outputs", "true"),
				)
			}
			resource.Test(t, resource.TestCase{
				ProtoV5ProviderFactories: coderFactory(),
				IsUnitTest:               true,
				Steps:                    []resource.TestStep{step},
			})
		})
	}
}
