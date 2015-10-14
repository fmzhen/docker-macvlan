package vlan

import (
	"github.com/codegangsta/cli"
	"github.com/fmzhen/docker-macvlan/macvlan/daemon"

	log "github.com/Sirupsen/logrus"
)

func CreateVlan(ctx *cli.Context) {
	vlanname := ctx.String("name")
	vlansubnet := ctx.String("subnet")
	CliVlanHostIF = ctx.String("host-interface")

	if err := daemon.CreateVlanNetwork(vlanname, vlansubnet, CliVlanHostIF); err != nil {
		log.Warnf("create subnet error: %v", err)
	}
}

func Vlan(ctx *cli.Context) {
	CliAttachName = ctx.String("name")
	CliContainerName = ctx.String("container-name")

	res, ok := daemon.CheckVlanName(CliAttachName)
	if !ok {
		log.Fatalf("the attach vlan name doesn't exist, please create fitst \n")
	}

	//TODO: verify whether the container exist

	daemon.AddVlannetwork(res, CliAttachName, CliContainerName)
}
