package examplefullstack

import (
	"context"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	auresthttpclient "github.com/StephanHCB/go-autumn-restclient/implementation/httpclient"
	aurestlogging "github.com/StephanHCB/go-autumn-restclient/implementation/requestlogging"
	aurestretry "github.com/StephanHCB/go-autumn-restclient/implementation/retry"
	"net/http"
)

func example() {
	// assumes you set up logging by importing one of the go-autumn-logging-xxx dependencies
	//
	// for this example, let's set up a logger that does nothing, so we don't pull in these dependencies here
	//
	// This of course makes the requestLoggingClient not work.
	aulogging.SetupNoLoggerForTesting()

	// set up
	httpClient, _ := auresthttpclient.New(0, []byte{}, nil)
	requestLoggingClient := aurestlogging.New(httpClient)
	retryingClient := aurestretry.New(requestLoggingClient, 2, func(ctx context.Context, response *aurestclientapi.ParsedResponse, err error) bool {
		return response.Status >= 500 || err != nil
	}, nil)

	// now make a request
	bodyDto := make(map[string]interface{})

	response := aurestclientapi.ParsedResponse{
		Body: &bodyDto,
	}
	_ = retryingClient.Perform(context.Background(), http.MethodGet, "https://some.rest.api", nil, &response)

	// now bodyDto is filled with the response
}