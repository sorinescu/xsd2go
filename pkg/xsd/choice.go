package xsd

import (
	"encoding/xml"
)

type Choice struct {
	XMLName   xml.Name   `xml:"http://www.w3.org/2001/XMLSchema choice"`
	MinOccurs string     `xml:"minOccurs,attr"`
	MaxOccurs string     `xml:"maxOccurs,attr"`
	Elements  []Element  `xml:"element"`
	Sequences []Sequence `xml:"sequence"`
	schema    *Schema    `xml:"-"`
}

func (c *Choice) compile(sch *Schema, parentElement *Element) {
	c.schema = sch
	for idx, _ := range c.Elements {
		el := &c.Elements[idx]

		el.compile(sch, parentElement)
		// Propagate array cardinality downwards
		if c.MaxOccurs == "unbounded" {
			el.MaxOccurs = "unbounded"
		}
		if el.MinOccurs == "" {
			el.MinOccurs = "0"
		}
	}

	for idx, _ := range c.Sequences {
		seq := &c.Sequences[idx]
		seq.compile(sch, parentElement)
	}
}
