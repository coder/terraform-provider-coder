package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestTemplateOutputsTyped(t *testing.T) {
	// Reads are environment-backed, so these tests cannot run in parallel.
	t.Setenv("CODER_WORKSPACE_BUILD_ID", "")
	t.Setenv("CODER_WORKSPACE_TEMPLATE_NAME", "shared")
	t.Setenv("CODER_WORKSPACE_TEMPLATE_ID", "11111111-1111-4111-8111-111111111111")
	t.Setenv("CODER_TEMPLATE_RESOURCE_REVISION_ID", "22222222-2222-4222-8222-222222222222")
	t.Setenv("CODER_TEMPLATE_OUTPUTS", ` {"namespace":"shared","nested":{"enabled":true,"optional":null},"mixed":[1,"two",false,null,{"x":[]}],"large":9007199254740993,"decimal":0.1234567890123456789,"empty":{}} `)
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: coderFactory(),
		IsUnitTest:               true,
		Steps: []resource.TestStep{{
			Config: `
provider "coder" { url = "https://example.com:8080" }
data "coder_template_outputs" "current" {}
data "coder_workspace" "me" {}
resource "coder_agent" "test" {
  os = "linux"
  arch = "amd64"
}
output "namespace" { value = data.coder_template_outputs.current.outputs.namespace }
output "types_preserved" {
  value = (
    data.coder_template_outputs.current.outputs.nested.enabled == true &&
    data.coder_template_outputs.current.outputs.nested.optional == null &&
    data.coder_template_outputs.current.outputs.mixed == [1, "two", false, null, {x = []}] &&
    data.coder_template_outputs.current.outputs.large == 9007199254740993 &&
    data.coder_template_outputs.current.outputs.decimal == 0.1234567890123456789 &&
    data.coder_template_outputs.current.outputs.empty == {}
  )
}
`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckOutput("namespace", "shared"),
				resource.TestCheckOutput("types_preserved", "true"),
				resource.TestCheckResourceAttr("data.coder_template_outputs.current", "id", "11111111-1111-4111-8111-111111111111"),
				resource.TestCheckResourceAttr("data.coder_template_outputs.current", "revision_id", "22222222-2222-4222-8222-222222222222"),
				resource.TestCheckResourceAttr("data.coder_workspace.me", "access_url", "https://example.com:8080"),
				resource.TestCheckResourceAttr("coder_agent.test", "os", "linux"),
			),
		}},
	})
}
