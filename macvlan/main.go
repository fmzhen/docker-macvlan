package main

import (
	"os"

	"github.com/codegangsta/cli"
	"github.com/fmzhen/docker-macvlan/macvlan/flat"
)

const (
	appVersion = "0.1"
	appName    = "macvlan"
	appUsage   = "Docker Macvlan Networking"
)

func main() {

	app := cli.NewApp()
	app.Name = appName
	app.Usage = appUsage
	app.Version = appVersion
	app.Commands = []cli.Command{
		{
			Name:    "flat",
			Aliases: []string{"f"},
			Usage:   "add a docker container to host network",
			Flags: []cli.Flag{
				flat.FlaggwIP,
				flat.FlagIP,
				flat.FlagIF,
				flat.FlagMTU,
				flat.FlagContainerName,
			},
			Action: flat.Flat,
		},
	}
	app.Run(os.Args)
}
