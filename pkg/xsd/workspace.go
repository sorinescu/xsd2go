package xsd

import (
	"fmt"
	"os"
	"path/filepath"
)

type Workspace struct {
	Cache             map[string]*Schema
	IgnoredNamespaces map[string]struct{}
	GoModulesPath     string
}

func NewWorkspace(goModule, outputDir, xsdPath string, ignoredNamesapces []string) (*Workspace, error) {
	ws := Workspace{
		Cache:         map[string]*Schema{},
		IgnoredNamespaces: map[string]struct{}{},
		GoModulesPath: fmt.Sprintf("%s/%s", goModule, outputDir),
	}
	for _, ns := range ignoredNamesapces {
		ws.IgnoredNamespaces[ns] = struct{}{}
	}
	var err error
	_, err = ws.loadXsd(xsdPath)
	return &ws, err
}

func (ws *Workspace) GoModule() string {
	return filepath.Base(ws.GoModulesPath)
}

func (ws *Workspace) loadXsd(xsdPath string) (*Schema, error) {
	cached, found := ws.Cache[xsdPath]
	if found {
		return cached, nil
	}
	fmt.Println("\tParsing:", xsdPath)

	f, err := os.Open(xsdPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	schema, err := parseSchema(f)
	if err != nil {
		return nil, err
	}
	schema.ModulesPath = ws.GoModulesPath
	schema.filePath = xsdPath
	ws.Cache[xsdPath] = schema

	if len(ws.IgnoredNamespaces) != 0 {
		imports := make([]Import, 0, len(schema.Imports))
		for i := range schema.Imports {
			if _, found := ws.IgnoredNamespaces[schema.Imports[i].Namespace]; !found {
				imports = append(imports, schema.Imports[i])
			} else {
				fmt.Printf("\t\tIgnoring XML namespace %q\n", schema.Imports[i].Namespace)
			}
		}
		schema.Imports = imports
	}

	dir := filepath.Dir(xsdPath)
	for i := range schema.Imports {
		if err := schema.Imports[i].load(ws, dir); err != nil {
			return nil, err
		}
	}
	schema.compile()
	return schema, nil
}
