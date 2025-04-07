package provider_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/v2/provider"
)

type formTypeTestCase struct {
	name        string
	config      string
	assert      paramAssert
	expectError *regexp.Regexp
}

type paramAssert struct {
	Default  string
	FormType string
	Type     string
	Styling  string
}

type formTypeCheck struct {
	formType     provider.ParameterFormType
	optionType   provider.OptionType
	optionsExist bool
}

func TestValidateFormType(t *testing.T) {
	t.Parallel()

	//formTypesChecked := make(map[provider.ParameterFormType]map[provider.OptionType]map[bool]struct{})
	formTypesChecked := make(map[formTypeCheck]struct{})

	const paramName = "test_me"

	cases := []formTypeTestCase{
		{
			// When nothing is specified
			name:   "defaults",
			config: ezconfig(paramName, ezconfigOpts{}),
			assert: paramAssert{
				Default:  "",
				FormType: "input",
				Type:     "string",
				Styling:  "",
			},
		},
		{
			name:   "string radio",
			config: ezconfig(paramName, ezconfigOpts{Options: []string{"foo"}}),
			assert: paramAssert{
				Default:  "",
				FormType: "radio",
				Type:     "string",
				Styling:  "",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.assert.Styling == "" {
				c.assert.Styling = "{}"
			}

			formTypesChecked[formTypeTest(t, paramName, c)] = struct{}{}
		})
	}

	// TODO: assume uncovered cases should fail
	t.Run("AssumeErrorCases", func(t *testing.T) {
		t.Skip()
		// requiredChecks loops through all possible form_type and option_type
		// combinations.
		requiredChecks := make([]formTypeCheck, 0)
		//requiredChecks := make(map[provider.ParameterFormType][]provider.OptionType)
		for _, ft := range provider.ParameterFormTypes() {
			//requiredChecks[ft] = make([]provider.OptionType, 0)
			for _, ot := range provider.OptionTypes() {
				requiredChecks = append(requiredChecks, formTypeCheck{
					formType:     ft,
					optionType:   ot,
					optionsExist: false,
				})
				requiredChecks = append(requiredChecks, formTypeCheck{
					formType:     ft,
					optionType:   ot,
					optionsExist: true,
				})
				//requiredChecks[ft] = append(requiredChecks[ft], ot)
			}
		}

		for _, check := range requiredChecks {
			_, alreadyChecked := formTypesChecked[check]
			if alreadyChecked {
				continue
			}

			// Assume it will fail

		}

		//checkedFormTypes := make([]provider.ParameterFormType, 0)
		//for ft, ot := range formTypesChecked {
		//	checkedFormTypes = append(checkedFormTypes, ft)
		//	var _ = ot
		//}
		//
		//// Fist check all form types have at least 1 test.
		//require.ElementsMatch(t, provider.ParameterFormTypes(), checkedFormTypes, "some form types are missing tests")
		//
		//// Then check each form type has tried with each option type
		//for expectedFt, expectedOptionTypes := range requiredChecks {
		//	actual := formTypesChecked[expectedFt]
		//
		//	assert.Equalf(t, expectedOptionTypes, maps.Keys(actual), "some option types are missing for form type %s", expectedFt)
		//
		//	// Also check for a true/false
		//	for _, expectedOptionType := range expectedOptionTypes {
		//		assert.Equalf(t, []bool{true, false}, maps.Keys(actual[expectedOptionType]), "options should be both present and absent for form type %q, option type %q", expectedFt, expectedOptionType)
		//	}
		//}
	})
}

type ezconfigOpts struct {
	Options    []string
	FormType   string
	OptionType string
	Default    string
}

func ezconfig(paramName string, cfg ezconfigOpts) string {
	var body strings.Builder
	if cfg.Default != "" {
		body.WriteString(fmt.Sprintf("default = %q\n", cfg.Default))
	}
	if cfg.FormType != "" {
		body.WriteString(fmt.Sprintf("form_type = %q\n", cfg.FormType))
	}
	if cfg.OptionType != "" {
		body.WriteString(fmt.Sprintf("type = %q\n", cfg.OptionType))
	}

	for i, opt := range cfg.Options {
		body.WriteString("option {\n")
		body.WriteString(fmt.Sprintf("name = \"val_%d\"\n", i))
		body.WriteString(fmt.Sprintf("value = %q\n", opt))
		body.WriteString("}\n")
	}

	return coderParamHCL(paramName, body.String())
}

func coderParamHCL(paramName string, body string) string {
	return fmt.Sprintf(`
				provider "coder" {
				}
				data "coder_parameter" "%s" {
					name = "%s"
					%s
				}
		`, paramName, paramName, body)
}

func formTypeTest(t *testing.T, paramName string, c formTypeTestCase) formTypeCheck {
	var check formTypeCheck
	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: coderFactory(),
		Steps: []resource.TestStep{
			{
				Config: c.config,
				Check: func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					require.Len(t, state.Modules[0].Resources, 1)

					key := strings.Join([]string{"data", "coder_parameter", paramName}, ".")
					param := state.Modules[0].Resources[key]

					assert.Equal(t, c.assert.Default, param.Primary.Attributes["default"], "default value")
					assert.Equal(t, c.assert.FormType, param.Primary.Attributes["form_type"], "form_type")
					assert.Equal(t, c.assert.Type, param.Primary.Attributes["type"], "type")
					assert.JSONEq(t, c.assert.Styling, param.Primary.Attributes["styling"], "styling")

					ft := provider.ParameterFormType(param.Primary.Attributes["form_type"])
					ot := provider.OptionType(param.Primary.Attributes["type"])

					// Option blocks are not stored in a very friendly format
					// here.
					optionsExist := param.Primary.Attributes["option.0.name"] != ""
					check = formTypeCheck{
						formType:     ft,
						optionType:   ot,
						optionsExist: optionsExist,
					}

					return nil
				},
			},
		},
	})
	return check
}
