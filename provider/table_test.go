package provider_test

import (
	"strconv"
	"strings"
	"testing"
)

func TestMDtable(t *testing.T) {
	// Copy and paste the table to https://www.tablesgenerator.com/markdown_tables for easier editing
	table := strings.TrimSpace(`
| Name             | Type          | Input Value | Default   | Options | Validation | -> | Output Value | Optional | Error |
|------------------|---------------|-------------|-----------|---------|------------|----|--------------|----------|-------|
| Empty            | string,number | undefined   | undefined |         | undefined  |    | ""           | false    | -     |
| EmptyWithOptions | number        | undefined   | undefined |         | undefined  |    | ""           | false    | -     |
| DefaultSet       | number        | undefined   | 5         |         | undefined  |    | 5            | true     | -     |
`)

	type row struct {
		Name        string
		Types       []string
		InputValue  string
		Default     string
		Options     string
		Validation  string
		OutputValue string
		Optional    bool
		Error       string
	}

	rows := make([]row, 0)
	lines := strings.Split(table, "\n")
	for _, line := range lines[2:] {
		columns := strings.Split(line, "|")
		columns = columns[1 : len(columns)-2]
		for i := range columns {
			columns[i] = strings.TrimSpace(columns[i])
		}

		optional, err := strconv.ParseBool(columns[8])
		if err != nil {
			t.Fatalf("failed to parse optional column %q: %v", columns[8], err)
		}
		rows = append(rows, row{
			Name:        columns[0],
			Types:       strings.Split(columns[1], ","),
			InputValue:  columns[2],
			Default:     columns[3],
			Options:     columns[4],
			Validation:  columns[5],
			OutputValue: columns[7],
			Optional:    optional,
			Error:       columns[9],
		})
	}
}
