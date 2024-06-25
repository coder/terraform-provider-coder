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

type ResourceTestData struct {
	Name         string
	ResourceType string
}

func TestExamples(t *testing.T) {
	t.Parallel()

	for _, resourceTestData := range []ResourceTestData{
		{"coder_parameter", "data-source"},
		{"coder_workspace_tags", "data-source"},
		{"coder_app", "resource"}
	} {
		t.Run(resourceTestData.Name, func(t *testing.T) {
			resourceTestData := resourceTestData
			t.Parallel()

			resourceTest(t, resourceTestData)
		})
	}
}

func resourceTest(t *testing.T, testData ResourceTestData) {
	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"coder": provider.New(),
		},
		IsUnitTest: true,
		Steps: []resource.TestStep{{
			Config: mustReadFile(t, fmt.Sprintf("../examples/%ss/%s/%s.tf", testData.ResourceType, testData.Name, testData.ResourceType)),
		}},
	})
}

func mustReadFile(t *testing.T, path string) string {
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}
