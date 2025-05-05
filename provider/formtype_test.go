package provider_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/v2/provider"
)

// formTypeTestCase is the config for a single test case.
type formTypeTestCase struct {
	name        string
	config      formTypeCheck
	assert      paramAssert
	expectError *regexp.Regexp
}

// paramAssert is asserted on the provider's parsed terraform state.
type paramAssert struct {
	FormType provider.ParameterFormType
	Type     provider.OptionType
	Styling  json.RawMessage
}

// formTypeCheck is a struct that helps build the terraform config
type formTypeCheck struct {
	formType   provider.ParameterFormType
	optionType provider.OptionType
	options    bool

	// optional to inform the assert
	customOptions []string
	defValue      string
	styling       json.RawMessage
}

func (c formTypeCheck) String() string {
	return fmt.Sprintf("%s_%s_%t", c.formType, c.optionType, c.options)
}

func TestValidateFormType(t *testing.T) {
	t.Parallel()

	// formTypesChecked keeps track of all checks run. It will be used to
	// ensure all combinations of form_type and option_type are tested.
	// All untested options are assumed to throw an error.
	var formTypesChecked sync.Map

	expectType := func(expected provider.ParameterFormType, opts formTypeCheck) formTypeTestCase {
		ftname := opts.formType
		if ftname == "" {
			ftname = "default"
		}

		if opts.styling == nil {
			// Try passing arbitrary data in, as anything should be accepted
			opts.styling, _ = json.Marshal(map[string]any{
				"foo":      "bar",
				"disabled": true,
				"nested": map[string]any{
					"foo": "bar",
				},
			})
		}

		return formTypeTestCase{
			name: fmt.Sprintf("%s_%s_%t",
				ftname,
				opts.optionType,
				opts.options,
			),
			config: opts,
			assert: paramAssert{
				FormType: expected,
				Type:     opts.optionType,
				Styling:  opts.styling,
			},
			expectError: nil,
		}
	}

	// expectSameFormType just assumes the FormType in the check is the expected
	// FormType. Using `expectType` these fields can differ
	expectSameFormType := func(opts formTypeCheck) formTypeTestCase {
		return expectType(opts.formType, opts)
	}

	cases := []formTypeTestCase{
		{
			// When nothing is specified
			name:   "defaults",
			config: formTypeCheck{},
			assert: paramAssert{
				FormType: provider.ParameterFormTypeInput,
				Type:     provider.OptionTypeString,
				Styling:  []byte("{}"),
			},
		},
		// All default behaviors. Essentially legacy behavior.
		//	String
		expectType(provider.ParameterFormTypeRadio, formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeString,
		}),
		expectType(provider.ParameterFormTypeInput, formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeString,
		}),
		//	Number
		expectType(provider.ParameterFormTypeRadio, formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeNumber,
		}),
		expectType(provider.ParameterFormTypeInput, formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeNumber,
		}),
		//	Boolean
		expectType(provider.ParameterFormTypeRadio, formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeBoolean,
		}),
		expectType(provider.ParameterFormTypeCheckbox, formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeBoolean,
		}),
		//	List(string)
		expectType(provider.ParameterFormTypeRadio, formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeListString,
		}),
		expectType(provider.ParameterFormTypeTagSelect, formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeListString,
		}),

		// ---- New Behavior
		//	String
		expectSameFormType(formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeString,
			formType:   provider.ParameterFormTypeDropdown,
		}),
		expectSameFormType(formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeString,
			formType:   provider.ParameterFormTypeRadio,
		}),
		expectSameFormType(formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeString,
			formType:   provider.ParameterFormTypeInput,
		}),
		expectSameFormType(formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeString,
			formType:   provider.ParameterFormTypeTextArea,
		}),
		//	Number
		expectSameFormType(formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeNumber,
			formType:   provider.ParameterFormTypeDropdown,
		}),
		expectSameFormType(formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeNumber,
			formType:   provider.ParameterFormTypeRadio,
		}),
		expectSameFormType(formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeNumber,
			formType:   provider.ParameterFormTypeInput,
		}),
		expectSameFormType(formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeNumber,
			formType:   provider.ParameterFormTypeSlider,
		}),
		//	Boolean
		expectSameFormType(formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeBoolean,
			formType:   provider.ParameterFormTypeRadio,
		}),
		expectSameFormType(formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeBoolean,
			formType:   provider.ParameterFormTypeSwitch,
		}),
		expectSameFormType(formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeBoolean,
			formType:   provider.ParameterFormTypeCheckbox,
		}),
		//	List(string)
		expectSameFormType(formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeListString,
			formType:   provider.ParameterFormTypeRadio,
		}),
		expectSameFormType(formTypeCheck{
			options:       true,
			optionType:    provider.OptionTypeListString,
			formType:      provider.ParameterFormTypeMultiSelect,
			customOptions: []string{"red", "blue", "green"},
			defValue:      `["red", "blue"]`,
		}),
		expectSameFormType(formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeListString,
			formType:   provider.ParameterFormTypeTagSelect,
		}),

		// Some manual test cases
		{
			name: "list_string_bad_default",
			config: formTypeCheck{
				formType:      provider.ParameterFormTypeMultiSelect,
				optionType:    provider.OptionTypeListString,
				customOptions: []string{"red", "blue", "green"},
				defValue:      `["red", "yellow"]`,
				styling:       nil,
			},
			expectError: regexp.MustCompile("is not a valid option"),
		},
	}

	passed := t.Run("TabledTests", func(t *testing.T) {
		// TabledCases runs through all the manual test cases
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				if _, ok := formTypesChecked.Load(c.config.String()); ok {
					t.Log("Duplicated form type check, delete this extra test case")
					t.Fatalf("form type %q already checked", c.config.String())
				}

				formTypesChecked.Store(c.config.String(), struct{}{})
				formTypeTest(t, c)
			})
		}
	})

	if !passed {
		// Do not run additional tests and pollute the output
		t.Log("Tests failed, will not run the assumed error cases")
		return
	}

	// AssumeErrorCases assumes any uncovered test will return an error. Not covered
	// cases in the truth table are assumed to be invalid. So if the tests above
	// cover all valid cases, this asserts all the invalid cases.
	//
	// This test consequentially ensures all valid cases are covered manually above.
	t.Run("AssumeErrorCases", func(t *testing.T) {
		// requiredChecks loops through all possible form_type and option_type
		// combinations.
		requiredChecks := make([]formTypeCheck, 0)
		for _, ft := range append(provider.ParameterFormTypes(), "") {
			for _, ot := range provider.OptionTypes() {
				requiredChecks = append(requiredChecks, formTypeCheck{
					formType:   ft,
					optionType: ot,
					options:    false,
				})
				requiredChecks = append(requiredChecks, formTypeCheck{
					formType:   ft,
					optionType: ot,
					options:    true,
				})
			}
		}

		for _, check := range requiredChecks {
			if _, alreadyChecked := formTypesChecked.Load(check.String()); alreadyChecked {
				continue
			}

			ftName := check.formType
			if ftName == "" {
				ftName = "default"
			}
			fc := formTypeTestCase{
				name: fmt.Sprintf("%s_%s_%t",
					ftName,
					check.optionType,
					check.options,
				),
				config:      check,
				assert:      paramAssert{},
				expectError: regexp.MustCompile("is not supported"),
			}

			t.Run(fc.name, func(t *testing.T) {
				t.Parallel()

				// This is just helpful log output to give the boilerplate
				// to write the manual test.
				tcText := fmt.Sprintf(`
					expectSameFormType(%s, ezconfigOpts{
						Options:    %t,
						OptionType: %q,
						FormType:   %q,
					}),
				//`, "<expected_form_type>", check.options, check.optionType, check.formType)

				logDebugInfo := formTypeTest(t, fc)
				if !logDebugInfo {
					t.Logf("To construct this test case:\n%s", tcText)
				}
			})

		}
	})
}

// createTF converts a formTypeCheck into a terraform config string.
func createTF(paramName string, cfg formTypeCheck) (defaultValue string, tf string) {
	options := cfg.customOptions
	if cfg.options && len(cfg.customOptions) == 0 {
		switch cfg.optionType {
		case provider.OptionTypeString:
			options = []string{"foo"}
			defaultValue = "foo"
		case provider.OptionTypeBoolean:
			options = []string{"true", "false"}
			defaultValue = "true"
		case provider.OptionTypeNumber:
			options = []string{"1"}
			defaultValue = "1"
		case provider.OptionTypeListString:
			options = []string{`["red", "blue"]`}
			defaultValue = `["red", "blue"]`
		default:
			panic(fmt.Sprintf("unknown option type %q when generating options", cfg.optionType))
		}
	}

	if cfg.defValue == "" {
		cfg.defValue = defaultValue
	}

	var body strings.Builder
	if cfg.defValue != "" {
		body.WriteString(fmt.Sprintf("default = %q\n", cfg.defValue))
	}
	if cfg.formType != "" {
		body.WriteString(fmt.Sprintf("form_type = %q\n", cfg.formType))
	}
	if cfg.optionType != "" {
		body.WriteString(fmt.Sprintf("type = %q\n", cfg.optionType))
	}
	if cfg.styling != nil {
		body.WriteString(fmt.Sprintf("styling = %s\n", strconv.Quote(string(cfg.styling))))
	}

	for i, opt := range options {
		body.WriteString("option {\n")
		body.WriteString(fmt.Sprintf("name = \"val_%d\"\n", i))
		body.WriteString(fmt.Sprintf("value = %q\n", opt))
		body.WriteString("}\n")
	}

	return cfg.defValue, fmt.Sprintf(`
				provider "coder" {
				}
				data "coder_parameter" "%s" {
					name = "%s"
					%s
				}
		`, paramName, paramName, body.String())
}

func formTypeTest(t *testing.T, c formTypeTestCase) bool {
	t.Helper()
	const paramName = "test_param"
	// logDebugInfo is just a guess used for logging. It's not important. It cannot
	// determine for sure if the test passed because the terraform test runner is a
	// black box. It does not indicate if the test passed or failed. Since this is
	// just used for logging, this is good enough.
	logDebugInfo := true

	def, tf := createTF(paramName, c.config)
	checkFn := func(state *terraform.State) error {
		require.Len(t, state.Modules, 1)
		require.Len(t, state.Modules[0].Resources, 1)

		key := strings.Join([]string{"data", "coder_parameter", paramName}, ".")
		param := state.Modules[0].Resources[key]

		logDebugInfo = logDebugInfo && assert.Equal(t, def, param.Primary.Attributes["default"], "default value")
		logDebugInfo = logDebugInfo && assert.Equal(t, string(c.assert.FormType), param.Primary.Attributes["form_type"], "form_type")
		logDebugInfo = logDebugInfo && assert.Equal(t, string(c.assert.Type), param.Primary.Attributes["type"], "type")
		logDebugInfo = logDebugInfo && assert.JSONEq(t, string(c.assert.Styling), param.Primary.Attributes["styling"], "styling")

		return nil
	}
	if c.expectError != nil {
		checkFn = nil
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: coderFactory(),
		Steps: []resource.TestStep{
			{
				Config:      tf,
				Check:       checkFn,
				ExpectError: c.expectError,
			},
		},
	})

	if !logDebugInfo {
		t.Logf("Terraform config:\n%s", tf)
	}
	return logDebugInfo
}
