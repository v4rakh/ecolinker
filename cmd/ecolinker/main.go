package main

import (
	"context"
	"git.myservermanager.com/varakh/ecolinker/internal/app"
	"git.myservermanager.com/varakh/ecolinker/internal/server"
	"git.myservermanager.com/varakh/ecolinker/internal/terminal"
	"github.com/urfave/cli/v3"
	"log"
	"os"
)

func main() {
	cli.VersionFlag = &cli.BoolFlag{
		Name: "version",
	}

	application := &cli.Command{
		Name:                  app.Name,
		Usage:                 "command-line interface for EcoLinker",
		Version:               app.Version,
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			{
				Name: "server",
				Commands: []*cli.Command{
					server.ServeCmd,
				},
			},
			{
				Name: "ecoflow",
				Commands: []*cli.Command{
					terminal.EcoFlowDevicesListCmd,
					terminal.EcoFlowDeviceParametersCmd,
					terminal.EcoFlowDeviceBatteriesCmd,
					terminal.EcoFlowDeviceHistoryCmd,
					terminal.EcoFlowStatusCmd,
				},
			},
			{
				Name: "devices",
				Commands: []*cli.Command{
					terminal.DevicesListCmd,
					terminal.DevicesAddCmd,
					terminal.DevicesRmCmd,
				},
			},
			{
				Name: "subs",
				Commands: []*cli.Command{
					terminal.SubsListCmd,
					terminal.SubsAddCmd,
					terminal.SubsRmCmd,
				},
			},
			{
				Name: "collectors",
				Commands: []*cli.Command{
					terminal.CollectorsListCmd,
					terminal.CollectorsAddCmd,
					terminal.CollectorsRmCmd,
				},
			},
		},
	}

	if err := application.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
