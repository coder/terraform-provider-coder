package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/provider"
)

func TestWorkspaceTags(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"coder": provider.New(),
		},
		IsUnitTest: true,
		Steps: []resource.TestStep{{
			Config: `
			data "coder_parameter" "animal" {
				name = "animal"
				type = "string"
				default = "chris"
			}
			data "coder_workspace_tags" "wt" {
				tags = {
					"cat" = "james"
					"dog" = data.coder_parameter.animal.value
				}
			}`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 2)
				resource := state.Modules[0].Resources["data.coder_workspace_tags.wt"]
				require.NotNil(t, resource)

				attribs := resource.Primary.Attributes
				require.Equal(t, "james", attribs["tags.cat"])
				require.Equal(t, "chris", attribs["tags.dog"])
				return nil
			},
		}},
	})
}
