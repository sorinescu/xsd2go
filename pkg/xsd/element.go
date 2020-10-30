package xsd

import (
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/iancoleman/strcase"
)

// Element defines single XML element
type Element struct {
	XMLName           xml.Name     `xml:"http://www.w3.org/2001/XMLSchema element"`
	Name              string       `xml:"name,attr"`
	nameOverride      string       `xml:"-"`
	FieldOverride     bool         `xml:"-"`
	Type              reference    `xml:"type,attr"`
	Ref               reference    `xml:"ref,attr"`
	MinOccurs         string       `xml:"minOccurs,attr"`
	MaxOccurs         string       `xml:"maxOccurs,attr"`
	SubstitutionGroup reference    `xml:"substitutionGroup,attr"`
	refElm            *Element     `xml:"-"`
	ComplexType       *ComplexType `xml:"complexType"`
	SimpleType        *SimpleType  `xml:"simpleType"`
	schema            *Schema      `xml:"-"`
	typ               Type         `xml:"-"`
}

func (e *Element) Attributes() []Attribute {
	// Only for anonymous types, the rest "inherit" the type struct
	if e.typ != nil && e.typ.GoName() == "" {
		return e.typ.Attributes()
	}
	return []Attribute{}
}

func (e *Element) Elements() []Element {
	// Only for anonymous types, the rest "inherit" the type struct
	if e.typ != nil && e.typ.GoName() == "" {
		return e.typ.Elements()
	}
	return []Element{}
}

func (e *Element) GoBaseTypeName() string {
	//if e.typ != nil {
	//	return e.typ.GoTypeName()
	//}
	if e.Type != "" {
		return e.typ.GoTypeName()
	} else if e.isPlainString() {
		return "string"
	}
	return ""
}

func (e *Element) GoFieldName() string {
	name := e.Name
	if name == "" {
		return e.refElm.GoName()
	}
	if e.FieldOverride {
		name += "Elm"
	}
	return strcase.ToCamel(name)
}

func (e *Element) GoName() string {
	if e.nameOverride != "" {
		return strcase.ToCamel(e.nameOverride)
	}
	return e.GoFieldName()
}

func (e *Element) GoMemLayout() string {
	if e.isArray() {
		return "[]"
	}
	if (e.MaxOccurs == "1" || e.MaxOccurs == "") && e.MinOccurs == "0" && e.GoBaseTypeName() != "string" {
		return "*"
	}
	return ""
}

func (e *Element) GoTypeName() string {
	if e.isInlinedElement() {
		return e.schema.GoTypePrefix() + e.GoName()
	}
	if e.typ != nil && e.typ.GoTypeName() != "" {
		return e.typ.GoTypeName()
	}

	if e.Name == "" {
		return e.refElm.GoTypeName()
	}
	return e.schema.GoTypePrefix() + strcase.ToCamel(e.Name) + "Elem"
}

func (e *Element) isExportable() bool {
	return e.typ == nil || e.typ.GoTypeName() == "" || e.isInlinedElement()
}

func (e *Element) SubstitutingElements() []*Element {
	if e.Name == "" {
		return e.refElm.SubstitutingElements()
	}
	return e.schema.SubstitutedElements()[e]
}

func (e *Element) SubstitutedElement() *Element {
	if e.Name == "" {
		return e.refElm.SubstitutedElement()
	}
	return e.schema.SubstitutingElements()[e]
}

//func (e *Element) GoForeignModule() string {
//	foreignSchema := (*Schema)(nil)
//	if e.refElm != nil {
//		foreignSchema = e.refElm.schema
//	} else if e.typ != nil {
//		foreignSchema = e.typ.Schema()
//	}
//
//	if foreignSchema != nil && foreignSchema != e.schema {
//		return foreignSchema.GoPackageName() + "."
//	}
//	return ""
//}

func (e *Element) XmlName() string {
	name := e.Name
	if name == "" {
		return e.refElm.XmlName()
	}
	return name
}

func (e *Element) ContainsText() bool {
	return e.typ != nil && e.typ.ContainsText()
}

func (e *Element) isPlainString() bool {
	return e.SimpleType != nil || (e.Type == "" && e.Ref == "" && e.ComplexType == nil)
}

func (e *Element) isArray() bool {
	if e.MaxOccurs == "unbounded" {
		return true
	}
	occurs, err := strconv.Atoi(e.MaxOccurs)
	return err == nil && occurs > 1
}

func (e *Element) compile(s *Schema, parentElement *Element) {
	if e.schema != nil {
		return	// already compiled
	}

	e.schema = s
	if e.ComplexType != nil {
		e.typ = e.ComplexType
		if e.SimpleType != nil {
			panic("Not implemented: xsd:element " + e.Name + " defines ./xsd:simpleType and ./xsd:complexType together")
		} else if e.Type != "" {
			panic("Not implemented: xsd:element " + e.Name + " defines ./@type= and ./xsd:complexType together")
		}
		e.typ.compile(s, e)
	} else if e.SimpleType != nil {
		e.typ = e.SimpleType
		if e.Type != "" {
			panic("Not implemented: xsd:element " + e.Name + " defines ./@type= and ./xsd:simpleType together")
		}
		e.typ.compile(s, e)
	} else if e.Type != "" {
		e.typ = e.schema.findReferencedType(e.Type)
		if e.typ == nil {
			panic("Cannot resolve type reference: " + string(e.Type))
		}
	}

	if e.Ref != "" {
		e.refElm = e.schema.findReferencedElement(e.Ref)
		if e.refElm == nil {
			panic("Cannot resolve element reference: " + e.Ref)
		}
	}

	if e.isInlinedElement() {
		e.schema.registerInlinedElement(e, parentElement)
	}

	if e.SubstitutionGroup != "" {
		e.schema.registerElementSubstitution(e.SubstitutionGroup, e)
	}
}

func (e *Element) isInlinedElement() bool {
	return e.Ref == "" && e.Type == "" && !e.isPlainString()
}

func (e *Element) prefixNameWithParent(parentElement *Element) {
	// In case there are inlined xsd:elements within another xsd:elements, it may happen that two top-level xsd:elements
	// define child xsd:element of a same name. In such case, we need to override children name to avoid name clashes.
	if parentElement != nil {
		e.nameOverride = fmt.Sprintf("%s-%s", parentElement.GoName(), e.GoName())
	}
}
