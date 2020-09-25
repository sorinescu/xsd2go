package xsd

import (
	"encoding/xml"
	"fmt"
	"github.com/iancoleman/strcase"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// Schema is the root XSD element
type Schema struct {
	XMLName              xml.Name                `xml:"http://www.w3.org/2001/XMLSchema schema"`
	Xmlns                Xmlns                   `xml:"-"`
	TargetNamespace      string                  `xml:"targetNamespace,attr"`
	Imports              []Import                `xml:"import"`
	Elements             []Element               `xml:"element"`
	Attributes           []Attribute             `xml:"attribute"`
	ComplexTypes         []ComplexType           `xml:"complexType"`
	SimpleTypes          []SimpleType            `xml:"simpleType"`
	ModulesPath          string                  `xml:"-"`
	importedModules      map[string]*Schema      `xml:"-"`
	filePath             string                  `xml:"-"`
	inlinedElements      []Element               `xml:"-"`
	substitutedElements  map[*Element][]*Element `xml:"-"`
	substitutingElements map[*Element]*Element   `xml:"-"`
}

func parseSchema(f io.Reader) (*Schema, error) {
	schema := Schema{
		importedModules:      map[string]*Schema{},
		substitutedElements:  map[*Element][]*Element{},
		substitutingElements: map[*Element]*Element{},
	}
	d := xml.NewDecoder(f)

	if err := d.Decode(&schema); err != nil {
		return nil, fmt.Errorf("Error decoding XSD: %s", err)
	}

	return &schema, nil
}

func (sch *Schema) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	sch.Xmlns = parseXmlns(start)

	type s Schema
	ss := (*s)(sch)
	return d.DecodeElement(ss, &start)
}

func (sch *Schema) compile() {
	for idx, _ := range sch.Elements {
		el := &sch.Elements[idx]
		el.compile(sch, nil)
	}
	for idx, _ := range sch.ComplexTypes {
		ct := &sch.ComplexTypes[idx]
		ct.compile(sch, nil)
	}
	for idx, _ := range sch.SimpleTypes {
		st := &sch.SimpleTypes[idx]
		st.compile(sch, nil)
	}
}

func (sch *Schema) findReferencedAttribute(ref reference) *Attribute {
	innerSchema := sch.findReferencedSchemaByPrefix(ref.NsPrefix())
	if innerSchema == nil {
		panic("Internal error: referenced attribute '" + ref + "' cannot be found.")
	}
	return innerSchema.GetAttribute(ref.Name())
}

func (sch *Schema) findReferencedElement(ref reference) *Element {
	innerSchema := sch.findReferencedSchemaByPrefix(ref.NsPrefix())
	if innerSchema == nil {
		panic("Internal error: referenced element '" + string(ref) + "' cannot be found.")
	}
	if innerSchema != sch {
		sch.registerImportedModule(innerSchema)

	}
	return innerSchema.GetElement(ref.Name())
}

func (sch *Schema) findReferencedType(ref reference) Type {
	innerSchema := sch.findReferencedSchemaByPrefix(ref.NsPrefix())
	if innerSchema == nil {
		xmlnsUri := sch.Xmlns.UriByPrefix(ref.NsPrefix())
		if xmlnsUri == "http://www.w3.org/2001/XMLSchema" {
			return StaticType(ref.Name())
		}
		panic("Internal error: referenced type '" + string(ref) + "' cannot be found.")
	}
	if innerSchema != sch {
		sch.registerImportedModule(innerSchema)
	}
	return innerSchema.GetType(ref.Name())
}

func (sch *Schema) findReferencedSchemaByPrefix(xmlnsPrefix string) *Schema {
	return sch.findReferencedSchemaByXmlns(sch.xmlnsByPrefix(xmlnsPrefix))
}

func (sch *Schema) xmlnsByPrefix(xmlnsPrefix string) string {
	uri := sch.xmlnsByPrefixInternal(xmlnsPrefix)
	if uri == "" {
		panic("Internal error: Unknown xmlns prefix: " + xmlnsPrefix)
	}
	return uri
}

func (sch *Schema) xmlnsByPrefixInternal(xmlnsPrefix string) string {
	switch xmlnsPrefix {
	case "":
		return sch.TargetNamespace
	case "xml":
		return "http://www.w3.org/XML/1998/namespace"
	default:
		uri := sch.Xmlns.UriByPrefix(xmlnsPrefix)
		if uri == "" {
			for _, imported := range sch.importedModules {
				uri = imported.xmlnsByPrefixInternal(xmlnsPrefix)
				if uri != "" {
					return uri
				}
			}
		}
		return uri
	}
	return ""
}

func (sch *Schema) findReferencedSchemaByXmlns(xmlns string) *Schema {
	if sch.TargetNamespace == xmlns {
		return sch
	}
	for _, imp := range sch.Imports {
		if imp.Namespace == xmlns {
			return imp.ImportedSchema
		}
	}
	for _, imp := range sch.importedModules {
		s := imp.findReferencedSchemaByXmlns(xmlns)
		if s != nil {
			return s
		}
	}
	return nil
}

func (sch *Schema) Empty() bool {
	return len(sch.Elements) == 0 && len(sch.ComplexTypes) == 0
}

func (sch *Schema) ExportableElements() []*Element {
	var expElems []*Element
	for i := range sch.Elements {
		el := &sch.Elements[i]
		if el.isExportable() {
			expElems = append(expElems, el)
		}
	}
	for i := range sch.inlinedElements {
		el := &sch.inlinedElements[i]
		if el.isExportable() {
			expElems = append(expElems, el)
		}
	}
	return expElems
}

func (sch *Schema) ExportableComplexTypes() []ComplexType {
	elCache := map[string]bool{}
	for _, el := range sch.Elements {
		elCache[el.GoName()] = true
	}

	var res []ComplexType
	for _, typ := range sch.ComplexTypes {
		_, found := elCache[typ.GoName()]
		if !found {
			res = append(res, typ)
		}
	}
	return res
}

func (sch *Schema) GetAttribute(name string) *Attribute {
	for idx, attr := range sch.Attributes {
		if attr.Name == name {
			return &sch.Attributes[idx]
		}
	}
	return nil
}

func (sch *Schema) GetElement(name string) *Element {
	for idx, elm := range sch.Elements {
		if elm.Name == name {
			return &sch.Elements[idx]
		}
	}
	return nil
}

func (sch *Schema) GetType(name string) Type {
	if name == "string" || name == "base64Binary" {
		return StaticType("string")
	}
	for idx, typ := range sch.ComplexTypes {
		if typ.Name == name {
			return &sch.ComplexTypes[idx]
		}
	}
	for idx, typ := range sch.SimpleTypes {
		if typ.Name == name {
			return &sch.SimpleTypes[idx]
		}
	}
	return nil
}

//func (sch *Schema) GoPackageName() string {
//	xmlnsPrefix := sch.Xmlns.PrefixByUri(sch.TargetNamespace)
//	if xmlnsPrefix == "" {
//		xmlnsPrefix = strings.TrimSuffix(filepath.Base(sch.filePath), ".xsd")
//	}
//	return strings.ReplaceAll(xmlnsPrefix, "-", "_")
//}

func (sch *Schema) GoPackageName() string {
	return filepath.Base(sch.ModulesPath)
}

func (sch *Schema) xmlnsPrefix() string {
	xmlnsPrefix := sch.Xmlns.PrefixByUri(sch.TargetNamespace)
	if xmlnsPrefix == "" {
		xmlnsPrefix = strings.TrimSuffix(filepath.Base(sch.filePath), ".xsd")
	}
	return strings.ReplaceAll(xmlnsPrefix, "-", "_")
}

func (sch *Schema) GoModelsFilePrefix() string {
	return sch.xmlnsPrefix()
}

func (sch *Schema) GoTypePrefix() string {
	return strcase.ToCamel(sch.xmlnsPrefix())
}

func (sch *Schema) GoImportsNeeded() []string {
	imports := []string{"encoding/xml"}
	//for _, importedMod := range sch.importedModules {
	//	imports = append(imports, fmt.Sprintf("%s/%s", sch.ModulesPath, importedMod.GoPackageName()))
	//}
	sort.Strings(imports)
	return imports
}

func (sch *Schema) SubstitutedElements() map[*Element][]*Element {
	return sch.substitutedElements
}

func (sch *Schema) SubstitutingElements() map[*Element]*Element {
	return sch.substitutingElements
}

func (sch *Schema) registerImportedModule(module *Schema) {
	sch.importedModules[module.xmlnsPrefix()] = module
}

// Some elements are not defined at the top-level, rather these are inlined in the complexType definitions
func (sch *Schema) registerInlinedElement(el *Element, parentElement *Element) {
	found := false
	for idx, _ := range sch.Elements {
		e := &sch.Elements[idx]
		if e == el {
			found = true
			break
		}
	}
	if !found {
		if el.Name == "" {
			panic("Not implemented: found inlined xsd:element without @name attribute")
		}
		el.prefixNameWithParent(parentElement)
		sch.inlinedElements = append(sch.inlinedElements, *el)
	}
}

func (sch *Schema) registerElementSubstitution(substGroup reference, el *Element) {
	substSchema := sch.findReferencedSchemaByPrefix(substGroup.NsPrefix())
	if substSchema == nil {
		panic("Internal error: referenced substitution group '" + string(substGroup) + "' schema cannot be found.")
	}
	if substSchema != sch {
		sch.registerImportedModule(substSchema)
	}
	substEl := substSchema.GetElement(substGroup.Name())
	if substEl == nil {
		panic("Internal error: referenced substitution group '" + string(substGroup) + "' cannot be found.")
	}

	sch.substitutingElements[el] = substEl
	substSchema.substitutedElements[substEl] = append(substSchema.substitutedElements[substEl], el)
}

type Import struct {
	XMLName        xml.Name `xml:"http://www.w3.org/2001/XMLSchema import"`
	Namespace      string   `xml:"namespace,attr"`
	SchemaLocation string   `xml:"schemaLocation,attr"`
	ImportedSchema *Schema  `xml:"-"`
}

func (i *Import) load(ws *Workspace, baseDir string) (err error) {
	if i.SchemaLocation != "" {
		i.ImportedSchema, err = ws.loadXsd(filepath.Join(baseDir, i.SchemaLocation))
	}
	return
}
