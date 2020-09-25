package template

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"text/template"

	"github.com/markbates/pkger"
	"github.com/sorinescu/xsd2go/pkg/xsd"
)

func GenerateTypes(schema *xsd.Schema, outputDir string) error {
	t, err := newTypesTemplate()
	if err != nil {
		return err
	}

	err = os.MkdirAll(outputDir, os.FileMode(0722))
	if err != nil {
		return err
	}
	goFile := fmt.Sprintf("%s/%s_models.go", outputDir, schema.GoModelsFilePrefix())
	fmt.Printf("\tGenerating '%s'\n", goFile)
	f, err := os.Create(goFile)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, schema); err != nil {
		return err
	}

	p, err := format.Source(buf.Bytes())
	if err != nil {
		return errors.New(err.Error() + " in following file:\n" + string(buf.Bytes()))
	}

	_, err = f.Write(p)
	if err != nil {
		return err
	}

	return nil
}

func GenerateGlobals(ws *xsd.Workspace, outputDir string) error {
	t, err := newGlobalsTemplate()
	if err != nil {
		return err
	}

	//packageName := ws.GoModule
	//dir := filepath.Join(outputDir, packageName)
	dir := outputDir
	err = os.MkdirAll(dir, os.FileMode(0722))
	if err != nil {
		return err
	}
	goFile := fmt.Sprintf("%s/globals.go", dir)
	fmt.Printf("\tGenerating '%s'\n", goFile)
	f, err := os.Create(goFile)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, ws); err != nil {
		return err
	}

	p, err := format.Source(buf.Bytes())
	if err != nil {
		return errors.New(err.Error() + " in following file:\n" + string(buf.Bytes()))
	}

	_, err = f.Write(p)
	if err != nil {
		return err
	}

	return nil
}

func newTypesTemplate() (*template.Template, error) {
	in, err := pkger.Open("/pkg/template/types.tmpl")
	if err != nil {
		return nil, err
	}
	defer in.Close()

	tempText, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}

	return template.New("types.tmpl").Funcs(template.FuncMap{}).Parse(string(tempText))
}

func newGlobalsTemplate() (*template.Template, error) {
	in, err := pkger.Open("/pkg/template/globals.tmpl")
	if err != nil {
		return nil, err
	}
	defer in.Close()

	tempText, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}

	return template.New("globals.tmpl").Funcs(template.FuncMap{}).Parse(string(tempText))
}
