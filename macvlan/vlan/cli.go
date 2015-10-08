package vlan

import "github.com/codegangsta/cli"

// the follew option will see in help doc. can set the default value;
var (
	// flat option
	FlagVlanName   = cli.StringFlag{Name: "name", Usage: "the vlan name"}
	FlagVlanSubnet = cli.StringFlag{Name: "subnet", Usage: "the vlan subnet"}
)

// vlan optiion.
var (
	CliVlanName   string
	CliVlanSubnet string
)
