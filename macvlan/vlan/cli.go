package vlan

import "github.com/codegangsta/cli"

// the follew option will see in help doc. can set the default value;
var (
	// flat option
	FlagVlanName   = cli.StringFlag{Name: "name", Usage: "the vlan name"}
	FlagVlanSubnet = cli.StringFlag{Name: "subnet", Usage: "the vlan subnet"}
	FlagHostIF     = cli.StringFlag{Name: "host-interface", Value: CliVlanHostIF, Usage: "parent of vlan device"}

	FlagAttachName    = cli.StringFlag{Name: "name", Usage: "the vlan name of the container will join"}
	FlagContainerName = cli.StringFlag{Name: "container-name", Usage: "the container name which want to config"}
)

// vlan optiion.
var (
	CliAttachName    string
	CliContainerName string
	CliVlanSubnet    string
	CliVlanHostIF    string = "eth0"
)
