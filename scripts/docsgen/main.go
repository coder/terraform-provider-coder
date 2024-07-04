package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/coder/terraform-provider-coder/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/xerrors"
)

// This script patches Markdown docs generated by `terraform-plugin-docs` to expose the original deprecation message.

const docsDir = "docs" // FIXME expose as flag?

var reDeprecatedProperty = regexp.MustCompile("`([^`]+)` \\(([^,\\)]+), Deprecated\\) ([^\n]+)")

func main() {
	p := provider.New()
	err := exposeDeprecationMessage(p)
	if err != nil {
		log.Fatal(err)
	}
}

func exposeDeprecationMessage(p *schema.Provider) error {
	// Patch data-sources
	for dataSourceName, dataSource := range p.DataSourcesMap {
		docFile := filepath.Join(docsDir, "data-sources", strings.Replace(dataSourceName, "coder_", "", 1)+".md")

		err := adjustDocFile(docFile, dataSource.Schema)
		if err != nil {
			return xerrors.Errorf("unable to adjust data-source doc file (data-source: %s): %w", dataSourceName, err)
		}
	}

	// Patch resources
	for resourceName, resource := range p.ResourcesMap {
		docFile := filepath.Join(docsDir, "resources", strings.Replace(resourceName, "coder_", "", 1)+".md")

		err := adjustDocFile(docFile, resource.Schema)
		if err != nil {
			return xerrors.Errorf("unable to adjust resource doc file (resource: %s): %w", resourceName, err)
		}
	}

	// Patch index
	docFile := filepath.Join(docsDir, "index.md")
	err := adjustDocFile(docFile, p.Schema)
	if err != nil {
		return xerrors.Errorf("unable to adjust index doc file: %w", err)
	}
	return nil
}

func adjustDocFile(docPath string, schemas map[string]*schema.Schema) error {
	doc, err := os.ReadFile(docPath)
	if err != nil {
		return xerrors.Errorf("can't read the source doc file: %w", err)
	}

	result := writeDeprecationMessage(doc, schemas)

	err = os.WriteFile(docPath, result, 0644)
	if err != nil {
		return xerrors.Errorf("can't write modified doc file: %w", err)
	}
	return nil
}

func writeDeprecationMessage(doc []byte, schemas map[string]*schema.Schema) []byte {
	return reDeprecatedProperty.ReplaceAllFunc(doc, func(m []byte) []byte {
		matches := reDeprecatedProperty.FindSubmatch(m)
		propertyName := matches[1]
		description := matches[3]

		sch := schemas[string(propertyName)]
		if string(description) != sch.Description {
			log.Printf("warn: same property name `%s` but description does not match, most likely a different property", propertyName)
			return m
		}
		return bytes.Replace(m, []byte("Deprecated"), []byte(fmt.Sprintf("**Deprecated**: %s", sch.Deprecated)), 1)
	})
}
