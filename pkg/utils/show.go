package utils

import (
	"github.com/hinfinite/helm/pkg/action"
	"github.com/hinfinite/helm/pkg/cli"
)

func runShow(args []string, client *action.Show, vals map[string]interface{}, settings *cli.EnvSettings) (string, error) {
	debug("Original chart version: %q", client.Version)
	if client.Version == "" && client.Devel {
		debug("setting version to >0.0.0-0")
		client.Version = ">0.0.0-0"
	}

	cp, err := client.ChartPathOptions.LocateChart(args[0], settings)
	if err != nil {
		return "", err
	}
	return client.Run(cp, vals)
}
