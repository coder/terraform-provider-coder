package provider_test

import (
	"regexp"
	"testing"

	"github.com/coder/terraform-provider-coder/provider"
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
		Name: "FieldsExist",
		Config: `
data "coder_parameter" "region" {
	name = "Region"
	type = "string"
	description = "Some option!"
	mutable = true
	icon = "/icon/region.svg"
	option {
		name = "US Central"
		value = "us-central1-a"
		icon = "/icon/central.svg"
		description = "Select for central!"
	}
	option {
		name = "US East"
		value = "us-east1-a"
		icon = "/icon/east.svg"
		description = "Select for east!"
	}
}
`,
		Check: func(state *terraform.ResourceState) {
			attrs := state.Primary.Attributes
			for key, value := range map[string]interface{}{
				"name":                 "Region",
				"type":                 "string",
				"description":          "Some option!",
				"mutable":              "true",
				"icon":                 "/icon/region.svg",
				"option.0.name":        "US Central",
				"option.0.value":       "us-central1-a",
				"option.0.icon":        "/icon/central.svg",
				"option.0.description": "Select for central!",
				"option.1.name":        "US East",
				"option.1.value":       "us-east1-a",
				"option.1.icon":        "/icon/east.svg",
				"option.1.description": "Select for east!",
			} {
				require.Equal(t, value, attrs[key])
			}
		},
	}, {
		Name: "ValidationWithOptions",
		Config: `
data "coder_parameter" "region" {
	name = "Region"
	type = "number"
	option {
		name = "1"
		value = "1"
	}
	validation {
		regex = "1"
		error = "Not 1"
	}
}
`,
		ExpectError: regexp.MustCompile("conflicts with option"),
	}, {
		Name: "ValidationRegexMissingError",
		Config: `
data "coder_parameter" "region" {
	name = "Region"
	type = "string"
	default = "hello"
	validation {
		regex = "hello"
	}
}
`,
		ExpectError: regexp.MustCompile("an error must be specified"),
	}, {
		Name: "NumberValidation",
		Config: `
data "coder_parameter" "region" {
	name = "Region"
	type = "number"
	default = 2
	validation {
		min = 1
		max = 5
	}
}
`,
	}, {
		Name: "DefaultNotNumber",
		Config: `
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
data "coder_parameter" "region" {
	name = "Region"
	option {
		name = "1"
		value = "1"
	}
}
`,
	}, {
		Name: "DuplicateOptionName",
		Config: `
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
		ExpectError: regexp.MustCompile("cannot have the same name"),
	}, {
		Name: "DuplicateOptionValue",
		Config: `
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

func TestValueValidatesType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		Name,
		Type,
		Value,
		Regex string
		Min,
		Max int
		Error *regexp.Regexp
	}{{
		Name:  "StringWithMin",
		Type:  "string",
		Min:   1,
		Error: regexp.MustCompile("cannot be specified"),
	}, {
		Name:  "StringWithMax",
		Type:  "string",
		Max:   1,
		Error: regexp.MustCompile("cannot be specified"),
	}, {
		Name:  "NonStringWithRegex",
		Type:  "number",
		Regex: "banana",
		Error: regexp.MustCompile("a regex cannot be specified"),
	}, {
		Name:  "Bool",
		Type:  "bool",
		Value: "true",
	}, {
		Name:  "InvalidNumber",
		Type:  "number",
		Value: "hi",
		Error: regexp.MustCompile("parse value hi as int"),
	}, {
		Name:  "NumberBelowMin",
		Type:  "number",
		Value: "0",
		Min:   1,
		Error: regexp.MustCompile("is less than the minimum"),
	}, {
		Name:  "NumberAboveMax",
		Type:  "number",
		Value: "1",
		Max:   0,
		Error: regexp.MustCompile("is more than the maximum"),
	}} {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			v := &provider.Validation{
				Min:   tc.Min,
				Max:   tc.Max,
				Regex: tc.Regex,
			}
			err := v.Valid(tc.Type, tc.Value)
			if tc.Error != nil {
				require.True(t, tc.Error.MatchString(err.Error()))
			}
		})
	}
}
