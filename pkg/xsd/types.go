package xsd

import (
	"encoding/xml"

	"github.com/iancoleman/strcase"
)

type Type interface {
	GoName() string
	GoTypeName() string
	GoBaseTypeName() string
	Schema() *Schema
	Attributes() []Attribute
	Elements() []Element
	ContainsText() bool
	compile(*Schema, *Element)
}

type ComplexType struct {
	XMLName          xml.Name        `xml:"http://www.w3.org/2001/XMLSchema complexType"`
	Name             string          `xml:"name,attr"`
	Mixed            bool            `xml:"mixed,attr"`
	AttributesDirect []Attribute     `xml:"attribute"`
	Sequence         *Sequence       `xml:"sequence"`
	schema           *Schema         `xml:"-"`
	SimpleContent    *SimpleContent  `xml:"simpleContent"`
	ComplexContent   *ComplexContent `xml:"complexContent"`
	Choice           *Choice         `xml:"choice"`
	content          GenericContent  `xml:"-"`
}

func (ct *ComplexType) Attributes() []Attribute {
	if ct.content != nil {
		return ct.content.Attributes()
	}
	return ct.AttributesDirect
}

func (ct *ComplexType) Elements() []Element {
	if ct.Sequence != nil {
		return ct.Sequence.Elements()
	} else if ct.content != nil {
		return ct.content.Elements()
	} else if ct.Choice != nil {
		return ct.Choice.Elements
	}
	return []Element{}
}

func (ct *ComplexType) GoBaseTypeName() string {
	if ct.content != nil {
		return ct.content.GoBaseTypeName()
	}
	return ""
}

func (ct *ComplexType) GoName() string {
	return strcase.ToCamel(ct.Name)
}

func (ct *ComplexType) GoTypeName() string {
	if ct.GoName() == "" {
		return ""
	}
	return ct.schema.GoTypePrefix() + ct.GoName()
}

//func (ct *ComplexType) GoForeignModule() string {
//	if ct.schema == nil {
//		return ""
//	}
//	return ct.schema.GoPackageName() + "."
//}

func (ct *ComplexType) ContainsInnerXml() bool {
	return ct.Mixed
}

func (ct *ComplexType) ContainsText() bool {
	return ct.content != nil && ct.content.ContainsText()
}

func (ct *ComplexType) Schema() *Schema {
	return ct.schema
}

func (ct *ComplexType) compile(sch *Schema, parentElement *Element) {
	ct.schema = sch
	if ct.Sequence != nil {
		ct.Sequence.compile(sch, parentElement)
	}

	// Handle improbable name clash. Consider XSD defining two attributes on the element:
	// "id" and "Id", this would create name clash given the camelization we do.
	goNames := map[string]uint{}
	for idx, _ := range ct.Attributes() {
		attribute := &ct.Attributes()[idx]
		attribute.compile(sch)

		count := goNames[attribute.GoName()]
		count += 1
		goNames[attribute.GoName()] = count
		attribute.DuplicateCount = count
		// Second GoName may be different depending on the DuplicateCount
		goNames[attribute.GoName()] = count
	}

	if ct.ComplexContent != nil {
		ct.content = ct.ComplexContent
		if ct.SimpleContent != nil {
			panic("Not implemented: xsd:complexType " + ct.Name + " defines xsd:simpleContent and xsd:complexContent together")
		}
	} else if ct.SimpleContent != nil {
		ct.content = ct.SimpleContent
	}

	if ct.content != nil {
		if len(ct.AttributesDirect) > 1 {
			panic("Not implemented: xsd:complexType " + ct.Name + " defines direct attribute and xsd:*Content")
		}
		if ct.Sequence != nil {
			panic("Not implemented: xsd:complexType " + ct.Name + " defines xsd:sequence and xsd:*Content")
		}
		ct.content.compile(sch, parentElement)
	}

	if ct.Choice != nil {
		if ct.content != nil {
			panic("Not implemented: xsd:complexType " + ct.Name + " defines xsd:choice and xsd:*content")
		}
		if ct.Sequence != nil {
			panic("Not implemented: xsd:complexType " + ct.Name + " defines xsd:choice and xsd:sequence")
		}
		ct.Choice.compile(sch, parentElement)
	}
}

type SimpleType struct {
	XMLName xml.Name `xml:"http://www.w3.org/2001/XMLSchema simpleType"`
	Name    string   `xml:"name,attr"`
	schema  *Schema  `xml:"-"`
}

func (st *SimpleType) GoName() string {
	return strcase.ToCamel(st.Name)
}

func (st *SimpleType) GoTypeName() string {
	return "string"
}

//func (st *SimpleType) GoForeignModule() string {
//	if st.schema == nil {
//		return ""
//	}
//	return st.schema.GoPackageName() + "."
//}
//
func (st *SimpleType) Schema() *Schema {
	return st.schema
}

func (st *SimpleType) compile(sch *Schema, parentElement *Element) {
	st.schema = sch
}

func (st *SimpleType) Attributes() []Attribute {
	return []Attribute{}
}

func (st *SimpleType) Elements() []Element {
	return []Element{}
}

func (st *SimpleType) ExtendedType() Type {
	return nil
}

func (st *SimpleType) GoBaseTypeName() string {
	return ""
}

func (st *SimpleType) ContainsText() bool {
	return true
}

type staticType string

func (st staticType) GoName() string {
	return string(st)
}

func (st staticType) GoTypeName() string {
	return st.GoName()
}

func (st staticType) GoForeignModule() string {
	return ""
}

func (st staticType) Attributes() []Attribute {
	return []Attribute{}
}

func (st staticType) Elements() []Element {
	return []Element{}
}

func (st staticType) ExtendedType() Type {
	return nil
}

func (st staticType) GoBaseTypeName() string {
	return ""
}

func (st staticType) Schema() *Schema {
	return nil
}

func (staticType) ContainsText() bool {
	return true
}

func (st staticType) compile(*Schema, *Element) {
}

func StaticType(name string) staticType {
	if name == "string" || name == "dateTime" || name == "base64Binary" || name == "normalizedString" || name == "token" || name == "NCName" || name == "anySimpleType" {
		return staticType("string")
	} else if name == "int" {
		return "int"
	} else if name == "integer" {
		return "int64"
	} else if name == "decimal" {
		return "float64"
	} else if name == "boolean" {
		return "bool"
	}
	panic("Type xsd:" + name + " not implemented")
	return staticType(name)
}
