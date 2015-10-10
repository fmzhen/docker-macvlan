package vlan

import (
	"github.com/codegangsta/cli"
	"github.com/fmzhen/docker-macvlan/macvlan/daemon"

	log "github.com/Sirupsen/logrus"
)

func CreateVlan(ctx *cli.Context) {
	vlanname := ctx.String("name")
	vlansubnet := ctx.String("subnet")
	vlanhostif := ctx.String("host-interface")

	if err := daemon.CreateVlanNetwork(vlanname, vlansubnet, vlanhostif); err != nil {
		log.Fatalf("create subnet error: %v", err)
	}
}

func Vlan(ctx *cli.Context) {

}
