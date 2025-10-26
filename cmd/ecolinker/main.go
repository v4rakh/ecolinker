package main

import (
	"context"
	"git.myservermanager.com/varakh/ecolinker/internal/meta"
	"git.myservermanager.com/varakh/ecolinker/internal/server"
	"git.myservermanager.com/varakh/ecolinker/internal/terminal"
	"github.com/urfave/cli/v3"
	golog "log"
	"os"
)

func main() {
	cli.VersionFlag = &cli.BoolFlag{
		Name: "version",
	}

	application := &cli.Command{
		Name:                  meta.Name,
		Usage:                 "command-line interface for EcoLinker",
		Version:               meta.Version,
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			{
				Name: "server",
				Commands: []*cli.Command{
					serveCmd,
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
					terminal.CollectorsInvokeCmd,
				},
			},
		},
	}

	if err := application.Run(context.Background(), os.Args); err != nil {
		golog.Fatal(err)
	}
}

var serveCmd = &cli.Command{
	Name:  "serve",
	Usage: "Starts the server and keeps it running",
	Action: func(ctx context.Context, _ *cli.Command) error {
		server := server.New(&ctx)
		server.Start()
		return nil
	},
}
