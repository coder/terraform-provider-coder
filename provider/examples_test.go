package provider_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/provider"
)

func TestExamples(t *testing.T) {
	t.Parallel()

	// Resource examples
	for _, testDir := range []string{
		"coder_workspace_tags",
	} {
		t.Run(testDir, func(t *testing.T) {
			testDir := testDir
			t.Parallel()

			resource.Test(t, resource.TestCase{
				Providers: map[string]*schema.Provider{
					"coder": provider.New(),
				},
				IsUnitTest: true,
				Steps: []resource.TestStep{{
					Config: mustReadFile(t, "../examples/resources/"+testDir+"/resource.tf"),
				}},
			})
		})
	}

	// Data source examples
	for _, testDir := range []string{
		"coder_parameter",
	} {
		t.Run(testDir, func(t *testing.T) {
			testDir := testDir
			t.Parallel()

			resource.Test(t, resource.TestCase{
				Providers: map[string]*schema.Provider{
					"coder": provider.New(),
				},
				IsUnitTest: true,
				Steps: []resource.TestStep{{
					Config: mustReadFile(t, "../examples/data-sources/"+testDir+"/data-source.tf"),
				}},
			})
		})
	}
}

func mustReadFile(t *testing.T, path string) string {
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}
