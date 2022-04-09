package examplefullstack

import (
	"context"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	aurestcapture "github.com/StephanHCB/go-autumn-restclient/implementation/capture"
	auresthttpclient "github.com/StephanHCB/go-autumn-restclient/implementation/httpclient"
	aurestmock "github.com/StephanHCB/go-autumn-restclient/implementation/mock"
	aurestplayback "github.com/StephanHCB/go-autumn-restclient/implementation/playback"
	aurestrecorder "github.com/StephanHCB/go-autumn-restclient/implementation/recorder"
	aurestlogging "github.com/StephanHCB/go-autumn-restclient/implementation/requestlogging"
	aurestretry "github.com/StephanHCB/go-autumn-restclient/implementation/retry"
	"net/http"
	"time"
)

func example() {
	// assumes you set up logging by importing one of the go-autumn-logging-xxx dependencies
	//
	// for this example, let's set up a logger that does nothing, so we don't pull in these dependencies here
	//
	// This of course makes the requestLoggingClient not work.
	aulogging.SetupNoLoggerForTesting()

	// ----- setup (can be done once during application startup) ----

	// 1. set up http client
	var timeout time.Duration = 0
	var customCACert []byte = nil
	var requestManipulator aurestclientapi.RequestManipulatorCallback = nil

	httpClient, _ := auresthttpclient.New(timeout, customCACert, requestManipulator)

	// 2. recording (for use with 1a)
	recorderClient := aurestrecorder.New(httpClient)

	// 3. request logging
	requestLoggingClient := aurestlogging.New(recorderClient)

	// 4. circuit breaker (see https://github.com/StephanHCB/go-autumn-restclient-circuitbreaker)
	// not included here because it has extra dependencies

	// 5. retry
	var repeatCount uint8 = 2 // 0 means only try once (but then why use this at all?)
	var condition aurestclientapi.RetryConditionCallback = func(ctx context.Context, response *aurestclientapi.ParsedResponse, err error) bool {
		return response.Status == http.StatusRequestTimeout
	}
	var beforeRetry aurestclientapi.BeforeRetryCallback = nil

	retryingClient := aurestretry.New(requestLoggingClient, repeatCount, condition, beforeRetry)

	// ----- now make a request -----

	bodyDto := make(map[string]interface{})

	response := aurestclientapi.ParsedResponse{
		Body: &bodyDto,
	}
	err := retryingClient.Perform(context.Background(), http.MethodGet, "https://some.rest.api", nil, &response)
	if err != nil {
		return
	}

	// now bodyDto is filled with the response and response.Status and response.Header are also set.
}

func testingExample() {
	// For testing, set up a do-nothing logger. This of course makes the requestLoggingClient not actually work,
	// but we've got the request capture for that.
	aulogging.SetupNoLoggerForTesting()

	// 1a. alternatively set up playback client (testing)
	playbackClient := aurestplayback.New("../resources/")

	// 1b. alternatively set up mock client (testing)
	mockClient := aurestmock.New(map[string]aurestclientapi.ParsedResponse{
		"GET /health <nil>": {
			Body:   nil,
			Status: 200,
			Header: http.Header{},
		},
	}, map[string]error{})

	// 2a. request capture
	_ = aurestcapture.New(mockClient)
	// or, using playback
	requestCaptureClient := aurestcapture.New(playbackClient)

	// 3. retry
	var repeatCount uint8 = 2 // 0 means only try once (but then why use this at all?)
	var condition aurestclientapi.RetryConditionCallback = func(ctx context.Context, response *aurestclientapi.ParsedResponse, err error) bool {
		return response.Status == http.StatusRequestTimeout
	}
	var beforeRetry aurestclientapi.BeforeRetryCallback = nil

	retryingClient := aurestretry.New(requestCaptureClient, repeatCount, condition, beforeRetry)

	// ----- now make a request -----

	aurestcapture.ResetRecording(requestCaptureClient)

	bodyDto := make(map[string]interface{})

	response := aurestclientapi.ParsedResponse{
		Body: &bodyDto,
	}
	_ = retryingClient.Perform(context.Background(), http.MethodGet, "https://some.rest.api", nil, &response)

	// now bodyDto is filled with the response ... assert on it

	// also assert on the recording what requests were made
	_ = aurestcapture.GetRecording(requestCaptureClient)
}
