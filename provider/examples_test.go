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

	t.Run("coder_agent", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			Providers: map[string]*schema.Provider{
				"coder": provider.New(),
			},
			IsUnitTest: true,
			Steps: []resource.TestStep{{
				Config: mustReadFile(t, "../examples/resources/coder_agent/resource.tf"),
			}},
		})
	})

	t.Run("coder_parameter", func(t *testing.T) {
		t.Parallel()

		resource.Test(t, resource.TestCase{
			Providers: map[string]*schema.Provider{
				"coder": provider.New(),
			},
			IsUnitTest: true,
			Steps: []resource.TestStep{{
				Config: mustReadFile(t, "../examples/resources/coder_parameter/resource.tf"),
			}},
		})
	})
}

func mustReadFile(t *testing.T, path string) string {
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}
