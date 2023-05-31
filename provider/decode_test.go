package provider_test

import (
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/provider"
)

func TestDecode(t *testing.T) {
	const (
		legacyVariable     = "Legacy Variable"
		legacyVariableName = "Legacy Variable Name"

		displayName = "Display Name"
	)

	aMap := map[string]interface{}{
		"name":                 "Parameter Name",
		"type":                 "number",
		"display_name":         displayName,
		"legacy_variable":      legacyVariable,
		"legacy_variable_name": legacyVariableName,
		"min":                  nil,
		"validation": []map[string]interface{}{
			{
				"min": nil,
				"max": 5,
			},
		},
	}

	var param provider.Parameter
	err := mapstructure.Decode(aMap, &param)
	require.NoError(t, err)
	assert.Equal(t, displayName, param.DisplayName)
	assert.Equal(t, legacyVariable, param.LegacyVariable)
	assert.Equal(t, legacyVariableName, param.LegacyVariableName)
	assert.Equal(t, 5, param.Validation[0].Max)
	assert.Equal(t, 0, param.Validation[0].Min)
}
