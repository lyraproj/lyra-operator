package main

import (
	"flag"
	"fmt"
	"runtime"

	"github.com/lyraproj/lyra-operator/cmd/manager/controller"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("operator-sdk Version: %v", sdkVersion.Version))
}

type mockApplicator struct{}

func (*mockApplicator) ApplyWorkflowWithHieraData(workflowName string, data map[string]string) {
	// Mock Applicator is not really applying the workflow ...
}

func (*mockApplicator) DeleteWorkflowWithHieraData(workflowName string, data map[string]string) {
	// Mock Applicator is not really doing anything ...
}

func main() {
	flag.Parse()

	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(logf.ZapLogger(false))

	printVersion()

	err := controller.Start("default", &mockApplicator{})
	if err != nil {
		log.Error(err, "controller failed to run")
	}

}
