package main

import (
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-incubator/credhub-cli/credhub"
	"github.com/cloudfoundry-incubator/credhub-cli/credhub/auth"
	"github.com/cloudfoundry-incubator/credhub-cli/util"
	"github.com/cloudfoundry/secure-credentials-broker/broker"
	"github.com/pivotal-cf/brokerapi"
)

func main() {
	brokerLogger := lager.NewLogger("secure-credentials-broker")
	brokerLogger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	brokerLogger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))
	brokerLogger.Info("starting up the secure credentials broker...")

	credHubClient := authenticate()
	serviceBroker := &broker.CredhubServiceBroker{CredHubClient: credHubClient, Logger: brokerLogger}

	brokerCredentials := brokerapi.BrokerCredentials{
		Username: "admin",
		Password: "admin",
	}

	brokerAPI := brokerapi.New(serviceBroker, brokerLogger, brokerCredentials)

	http.Handle("/", brokerAPI)

	var port string
	if port = os.Getenv("PORT"); len(port) == 0 {
		port = "8080"
	}

	brokerLogger.Fatal("http-listen", http.ListenAndServe(":"+port, nil))
}

func authenticate() *credhub.CredHub {

	skipTLSValidation := false
	if skipTLS := os.Getenv("SKIP_TLS_VALIDATION"); skipTLS == "true" {
		skipTLSValidation = true
	}

	ch, err := credhub.New(
		util.AddDefaultSchemeIfNecessary(os.Getenv("CREDHUB_SERVER")),
		credhub.SkipTLSValidation(skipTLSValidation),
		credhub.Auth(auth.UaaClientCredentials(os.Getenv("CREDHUB_CLIENT"), os.Getenv("CREDHUB_SECRET"))),
	)

	if err != nil {
		panic("credhub client configured incorrectly: " + err.Error())
	}

	return ch
}
