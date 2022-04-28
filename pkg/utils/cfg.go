package utils

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/hinfinite/helm/pkg/action"
	"github.com/hinfinite/helm/pkg/cli"
)

func debug(format string, v ...interface{}) {
	debug, _ := strconv.ParseBool(os.Getenv("HELM_DEBUG"))

	if debug {
		format = fmt.Sprintf("[debug] %s\n", format)
		log.Output(2, fmt.Sprintf(format, v...))
	}
}

// getCfg get helm config
func getCfg(namespace string) (*action.Configuration, *cli.EnvSettings) {
	settings := cli.New()
	settings.SetNamespace(namespace)
	actionConfig := &action.Configuration{}

	helmDriver := os.Getenv("HELM_DRIVER")
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), helmDriver, debug); err != nil {
		log.Fatal(err)
	}
	return actionConfig, settings
}
