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

	socketPath = "/run/docker/plugins/"
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
	app.Before = initEnv
	app.Action = Run
	app.Run(os.Args)
}

//create unix domain socket file, and set log level
func initEnv(ctx *cli.Context) error {
	socketFile := ctx.String("socket")
	// Default log level is Info
	if ctx.Bool("debug") {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	log.SetOutput(os.Stderr)
	initSock(socketFile)
	return nil
}
