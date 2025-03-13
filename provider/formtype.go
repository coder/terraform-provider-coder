package provider

import (
	"slices"

	"golang.org/x/xerrors"
)

type ParameterFormType string

const (
	ParameterFormTypeDefault     ParameterFormType = ""
	ParameterFormTypeRadio       ParameterFormType = "radio"
	ParameterFormTypeInput       ParameterFormType = "input"
	ParameterFormTypeDropdown    ParameterFormType = "dropdown"
	ParameterFormTypeCheckbox    ParameterFormType = "checkbox"
	ParameterFormTypeSwitch      ParameterFormType = "switch"
	ParameterFormTypeMultiSelect ParameterFormType = "multi-select"
	ParameterFormTypeTagInput    ParameterFormType = "tag-input"
	//ParameterFormTypeTextArea    ParameterFormType = "textarea"
	ParameterFormTypeError ParameterFormType = "error"
)

func ParameterFormTypes() []ParameterFormType {
	return []ParameterFormType{
		ParameterFormTypeDefault,
		ParameterFormTypeRadio,
		ParameterFormTypeInput,
		ParameterFormTypeDropdown,
		ParameterFormTypeCheckbox,
		ParameterFormTypeSwitch,
		ParameterFormTypeMultiSelect,
		ParameterFormTypeTagInput,
		//ParameterFormTypeTextArea,
		ParameterFormTypeError,
	}
}

// formTypeTruthTable is a map of [`type`][`optionCount` > 0] to `form_type`.
// The first value in the slice is the default value assuming `form_type` is
// not specified.
// | Type              | Options | Specified Form Type | form_type      | Notes                          |
// |-------------------|---------|---------------------|----------------|--------------------------------|
// | `string` `number` | Y       |                     | `radio`        |                                |
// | `string` `number` | Y       | `dropdown`          | `dropdown`     |                                |
// | `string` `number` | N       |                     | `input`        |                                |
// | `bool`            | Y       |                     | `radio`        |                                |
// | `bool`            | N       |                     | `checkbox`     |                                |
// | `bool`            | N       | `switch`            | `switch`       |                                |
// | `list(string)`    | Y       |                     | `radio`        |                                |
// | `list(string)`    | N       |                     | `tag-select`   |                                |
// | `list(string)`    | Y       | `multi-select`      | `multi-select` | Option values will be `string` |
var formTypeTruthTable = map[string]map[bool][]ParameterFormType{
	"string": {
		true:  {ParameterFormTypeRadio, ParameterFormTypeDropdown},
		false: {ParameterFormTypeInput},
	},
	"number": {
		true:  {ParameterFormTypeRadio, ParameterFormTypeDropdown},
		false: {ParameterFormTypeInput},
	},
	"bool": {
		true:  {ParameterFormTypeRadio},
		false: {ParameterFormTypeCheckbox, ParameterFormTypeSwitch},
	},
	"list(string)": {
		true:  {ParameterFormTypeRadio, ParameterFormTypeMultiSelect},
		false: {ParameterFormTypeTagInput},
	},
}

// ValidateFormType handles the truth table for the valid set of `type` and
// `form_type` options.
// | Type              | Options | Specified Form Type | form_type      | Notes                          |
// |-------------------|---------|---------------------|----------------|--------------------------------|
// | `string` `number` | Y       |                     | `radio`        |                                |
// | `string` `number` | Y       | `dropdown`          | `dropdown`     |                                |
// | `string` `number` | N       |                     | `input`        |                                |
// | `bool`            | Y       |                     | `radio`        |                                |
// | `bool`            | N       |                     | `checkbox`     |                                |
// | `bool`            | N       | `switch`            | `switch`       |                                |
// | `list(string)`    | Y       |                     | `radio`        |                                |
// | `list(string)`    | N       |                     | `tag-select`   |                                |
// | `list(string)`    | Y       | `multi-select`      | `multi-select` | Option values will be `string` |
func ValidateFormType(paramType string, optionCount int, specifiedFormType ParameterFormType) (string, ParameterFormType, error) {
	allowed, ok := formTypeTruthTable[paramType][optionCount > 0]
	if !ok || len(allowed) == 0 {
		return paramType, specifiedFormType, xerrors.Errorf("value type %q is not supported for 'form_types'", paramType)
	}

	if specifiedFormType == ParameterFormTypeDefault {
		// handle the default case
		specifiedFormType = allowed[0]
	}

	if !slices.Contains(allowed, specifiedFormType) {
		return paramType, specifiedFormType, xerrors.Errorf("value type %q is not supported for 'form_types'", paramType)
	}

	// Special case
	if paramType == "list(string)" && specifiedFormType == ParameterFormTypeMultiSelect {
		return "string", ParameterFormTypeMultiSelect, nil
	}

	return paramType, specifiedFormType, nil
}

func toStrings[A ~string](l []A) []string {
	var r []string
	for _, v := range l {
		r = append(r, string(v))
	}
	return r
}
