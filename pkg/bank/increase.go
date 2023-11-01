package bank

import (
	"github.com/increase/increase-go"
	"github.com/increase/increase-go/option"
)

// TODO: absract this behind an interface
var IncreaseClient *increase.Client

func init() {
	client := increase.NewClient(
		// defaults to os.LookupEnv("INCREASE_API_KEY")
		option.WithEnvironmentSandbox(), // defaults to option.WithEnvironmentProduction()
	)
	IncreaseClient = client
}
