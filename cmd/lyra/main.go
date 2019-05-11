package main

import (
	"fmt"
	"github.com/lyraproj/lyra-operator/cmd/manager/controller"
	"github.com/lyraproj/lyra/cmd/lyra/cmd"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func main() {
	if err := cmd.NewRootCmd(controller.NewControllerCmd()).Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}
