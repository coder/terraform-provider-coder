package provider_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/provider"
)

func TestExamples_CoderParameter(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"coder": provider.New(),
		},
		IsUnitTest: true,
		Steps: []resource.TestStep{{
			Config: mustReadFile(t, "../examples/resources/coder_workspace_tags/resource.tf"),
		}},
	})
}

func TestExamples_CoderWorkspaceTags(t *testing.T) {
	// no parallel as the test calls t.Setenv()
	workDir := "../examples/resources/coder_workspace_tags"
	t.Setenv(provider.TerraformWorkDirEnv, workDir)

	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"coder": provider.New(),
		},
		IsUnitTest: true,
		Steps: []resource.TestStep{{
			Config: mustReadFile(t, filepath.Join(workDir, "/resource.tf")),
		}},
	})
}

func mustReadFile(t *testing.T, path string) string {
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}
