package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestAITask_Enabled(t *testing.T) {
	t.Run("EnabledWhenTask", func(t *testing.T) {
		t.Setenv("CODER_TASK_ID", "7d8d4c2e-fb57-44f9-a183-22509819c2e7")

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
				provider "coder" {
				}
				resource "coder_ai_task" "test" {
					app_id = "9a3ff7b4-4b3f-48c6-8d3a-a8118ac921fc"
				}
				`,
				Check: func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					resource := state.Modules[0].Resources["coder_ai_task.test"]
					require.NotNil(t, resource)

					require.Equal(t, "true", resource.Primary.Attributes["enabled"])

					return nil
				},
			}},
		})
	})

	t.Run("DisabledWhenWorkspace", func(t *testing.T) {
		t.Setenv("CODER_TASK_ID", "")

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
				provider "coder" {
				}
				resource "coder_ai_task" "test" {
					app_id = "9a3ff7b4-4b3f-48c6-8d3a-a8118ac921fc"
				}
				`,
				Check: func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					resource := state.Modules[0].Resources["coder_ai_task.test"]
					require.NotNil(t, resource)

					require.Equal(t, "false", resource.Primary.Attributes["enabled"])

					return nil
				},
			}},
		})
	})
}

func TestAITask(t *testing.T) {
	t.Setenv("CODER_TASK_ID", "7d8d4c2e-fb57-44f9-a183-22509819c2e7")
	t.Setenv("CODER_TASK_PROMPT", "some task prompt")

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
				provider "coder" {
				}
				resource "coder_ai_task" "test" {
					app_id = "9a3ff7b4-4b3f-48c6-8d3a-a8118ac921fc"
				}
				`,
				Check: func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					resource := state.Modules[0].Resources["coder_ai_task.test"]
					require.NotNil(t, resource)
					for _, key := range []string{
						"id",
						"prompt",
						"app_id",
						"enabled",
					} {
						value := resource.Primary.Attributes[key]
						require.NotNil(t, value)
						require.Greater(t, len(value), 0)
					}

					taskID := resource.Primary.Attributes["id"]
					require.Equal(t, "7d8d4c2e-fb57-44f9-a183-22509819c2e7", taskID)

					taskPrompt := resource.Primary.Attributes["prompt"]
					require.Equal(t, "some task prompt", taskPrompt)

					return nil
				},
			}},
		})
	})

	t.Run("InvalidAppID", func(t *testing.T) {
		t.Parallel()

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
				provider "coder" {
				}
				resource "coder_ai_task" "test" {
					app_id = "not-a-uuid"
				}
				`,
				ExpectError: regexp.MustCompile(`expected "app_id" to be a valid UUID`),
			}},
		})
	})

	t.Run("DeprecatedSidebarApp", func(t *testing.T) {
		t.Parallel()

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
				provider "coder" {
				}
				resource "coder_ai_task" "test" {
					sidebar_app {
					    id = "9a3ff7b4-4b3f-48c6-8d3a-a8118ac921fc"
					}
				}
				`,
				Check: func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					resource := state.Modules[0].Resources["coder_ai_task.test"]
					require.NotNil(t, resource)

					for _, key := range []string{
						"id",
						"prompt",
						"app_id",
						"enabled",
					} {
						value := resource.Primary.Attributes[key]
						require.NotNil(t, value)
						require.Greater(t, len(value), 0)
					}

					require.Equal(t, "1", resource.Primary.Attributes["sidebar_app.#"])
					sidebarAppID := resource.Primary.Attributes["sidebar_app.0.id"]
					require.Equal(t, "9a3ff7b4-4b3f-48c6-8d3a-a8118ac921fc", sidebarAppID)

					actualAppID := resource.Primary.Attributes["app_id"]
					require.Equal(t, "9a3ff7b4-4b3f-48c6-8d3a-a8118ac921fc", actualAppID)

					return nil
				},
			}},
		})
	})

	t.Run("ConflictingFields", func(t *testing.T) {
		t.Parallel()

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
				provider "coder" {
				}
				resource "coder_ai_task" "test" {
					app_id = "9a3ff7b4-4b3f-48c6-8d3a-a8118ac921fc"
					sidebar_app {
					    id = "9a3ff7b4-4b3f-48c6-8d3a-a8118ac921fc"
					}
				}
				`,
				ExpectError: regexp.MustCompile(`"app_id": conflicts with sidebar_app`),
			}},
		})
	})

	t.Run("NoAppID", func(t *testing.T) {
		t.Parallel()

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
				provider "coder" {
				}
				resource "coder_ai_task" "test" {
				}
				`,
				ExpectError: regexp.MustCompile(`'app_id' must be set`),
			}},
		})
	})
}
