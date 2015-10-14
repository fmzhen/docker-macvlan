package service

import (
	"github.com/codegangsta/cli"
	"github.com/fmzhen/docker-macvlan/macvlan/daemon"
)

var (
	FlagContainerName = cli.StringFlag{Name: "container-name", Usage: "the container name which want to config"}
)

var (
	CliContainerName string
)

func Service(ctx *cli.Context) {
	CliContainerName := ctx.String("name")

	//TODO: check the container name is run ?

	//TODO: the value is just tmp
	value := "0.0.0.0/0"
	daemon.AddService(CliContainerName, value)
}
