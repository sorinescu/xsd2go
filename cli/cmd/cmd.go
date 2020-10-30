package cmd

import (
	"github.com/sorinescu/xsd2go/pkg/xsd2go"
	"github.com/urfave/cli"
	"os"
)

// Execute ...
func Execute() error {
	app := cli.NewApp()
	app.Name = "GoComply XSD2Go"
	app.Usage = "Automatically generate golang xml parser based on XSD"
	app.Commands = []cli.Command{
		convert,
	}

	return app.Run(os.Args)
}

var convert = cli.Command{
	Name:      "convert",
	Usage:     "convert XSD to golang code to parse xml files generated by given xsd",
	ArgsUsage: "XSD-FILE GO-MODULE-IMPORT OUTPUT-DIR",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "ignoreNS",
			Usage: "Ignore XML namespaces (don't import and don't generate types)",
		},
		&cli.StringSliceFlag{
			Name:  "ignoreSubst",
			Usage: "Ignore XML element substitution; name is like 'http://oval.mitre.org/XMLSchema/oval-common-5 notes'",
		},
	},
	Before: func(c *cli.Context) error {
		if c.NArg() != 3 {
			return cli.NewExitError("Exactly 3 arguments are required", 1)
		}
		return nil
	},
	Action: func(c *cli.Context) error {
		xsdFile, goModule, outputDir := c.Args()[0], c.Args()[1], c.Args()[2]
		ignoredNamespaces := c.StringSlice("ignoreNS")
		ignoredSubsts := c.StringSlice("ignoreSubst")
		err := xsd2go.Convert(xsdFile, goModule, outputDir, ignoredNamespaces, ignoredSubsts)
		if err != nil {
			return cli.NewExitError(err, 1)
		}
		return nil
	},
}
