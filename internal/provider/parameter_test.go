package provider_test

import (
	"regexp"
	"testing"

	"github.com/coder/terraform-provider-coder/internal/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestParameter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		Name        string
		Config      string
		ExpectError *regexp.Regexp
		Check       func(state *terraform.ResourceState)
	}{{
		Name: "NumberValidation",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	type = "number"
}
`,
	}, {
		Name: "DefaultNotNumber",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	type = "number"
	default = true
}
`,
		ExpectError: regexp.MustCompile("is not a number"),
	}, {
		Name: "DefaultNotBool",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	type = "bool"
	default = 5
}
`,
		ExpectError: regexp.MustCompile("is not a bool"),
	}, {
		Name: "OptionNotBool",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	type = "bool"
	option {
		value = 1
		name = 1
	}
	option {
		value = 2
		name = 2
	}
}`,
		ExpectError: regexp.MustCompile("\"2\" is not a bool"),
	}, {
		Name: "MultipleOptions",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	type = "string"
	option {
		name = "1"
		value = "1"
		icon = "/icon/code.svg"
		description = "Something!"
	}
	option {
		name = "2"
		value = "2"
	}
}

data "google_compute_regions" "regions" {}

data "coder_parameter" "region" {
	name = "Region"
	type = "string"
	icon = "/icon/asdasd.svg"
	option {
		name = "United States"
		value = "us-central1-a"
		icon = "/icon/usa.svg"
		description = "If you live in America, select this!"
	}
	option {
		name = "Europe"
		value = "2"
	}
}
`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":                 "Region",
				"option.#":             "2",
				"option.0.name":        "1",
				"option.0.value":       "1",
				"option.0.icon":        "/icon/code.svg",
				"option.0.description": "Something!",
			} {
				require.Equal(t, expected, state.Primary.Attributes[key])
			}
		},
	}, {
		Name: "DefaultWithOption",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	default = "hi"
	option {
		name = "1"
		value = "1"
	}
	option {
		name = "2"
		value = "2"
	}
}
`,
		ExpectError: regexp.MustCompile("Invalid combination of arguments"),
	}, {
		Name: "SingleOption",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	option {
		name = "1"
		value = "1"
	}
}
`,
	}, {
		Name: "DuplicateOptionDisplayName",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	type = "string"
	option {
		name = "1"
		value = "1"
	}
	option {
		name = "1"
		value = "2"
	}
}
`,
		ExpectError: regexp.MustCompile("cannot have the same display name"),
	}, {
		Name: "DuplicateOptionValue",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	type = "string"
	option {
		name = "1"
		value = "1"
	}
	option {
		name = "2"
		value = "1"
	}
}
`,
		ExpectError: regexp.MustCompile("cannot have the same value"),
	}} {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			resource.Test(t, resource.TestCase{
				Providers: map[string]*schema.Provider{
					"coder": provider.New(),
				},
				IsUnitTest: true,
				Steps: []resource.TestStep{{
					Config:      tc.Config,
					ExpectError: tc.ExpectError,
					Check: func(state *terraform.State) error {
						require.Len(t, state.Modules, 1)
						require.Len(t, state.Modules[0].Resources, 1)
						param := state.Modules[0].Resources["data.coder_parameter.region"]
						require.NotNil(t, param)
						t.Logf("parameter attributes: %#v", param.Primary.Attributes)
						if tc.Check != nil {
							tc.Check(param)
						}
						return nil
					},
				}},
			})
		})
	}
}
