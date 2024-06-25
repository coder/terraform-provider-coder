package provider_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/provider"
)

func TestExamples(t *testing.T) {
	t.Parallel()

	for _, testDir := range []string{
		"coder_parameter",
		"coder_workspace_tags",
	} {
		t.Run(testDir, func(t *testing.T) {
			testDir := testDir
			t.Parallel()

			resourceTest(t, testDir)
		})
	}
}

func resourceTest(t *testing.T, testDir string) {
	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"coder": provider.New(),
		},
		IsUnitTest: true,
		Steps: []resource.TestStep{{
			Config: mustReadFile(t, fmt.Sprintf("../examples/data-sources/%s/data-source.tf", testDir)),
		}},
	})
}

func mustReadFile(t *testing.T, path string) string {
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}
