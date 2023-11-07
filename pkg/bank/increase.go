package bank

import (
	"os"

	"github.com/increase/increase-go"
	"github.com/increase/increase-go/option"
)

// TODO: absract this behind an interface
var IncreaseClient *increase.Client

func init() {
	_, ok := os.LookupEnv("INCREASE_SANDBOX")
	if ok {
		client := increase.NewClient(
			// defaults to os.LookupEnv("INCREASE_API_KEY")
			option.WithEnvironmentSandbox(), // defaults to option.WithEnvironmentProduction()
		)
		IncreaseClient = client
	} else {
		// defaults to os.LookupEnv("INCREASE_API_KEY")
		client := increase.NewClient()
		IncreaseClient = client
	}
}
