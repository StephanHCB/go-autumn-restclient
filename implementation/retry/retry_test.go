package aurestretry

import (
	"context"
	"errors"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	aurestcapture "github.com/StephanHCB/go-autumn-restclient/implementation/capture"
	aurestmock "github.com/StephanHCB/go-autumn-restclient/implementation/mock"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/require"
	"testing"
)

func tstMock() aurestclientapi.Client {
	mockClient := aurestmock.New(
		map[string]aurestclientapi.ParsedResponse{
			"GET http://ok <nil>": {
				Body:   nil,
				Status: 200,
				Header: map[string][]string{
					headers.ContentType: []string{aurestclientapi.ContentTypeApplicationJson},
				},
			},
			"GET http://500 <nil>": {
				Body:   nil,
				Status: 500,
				Header: map[string][]string{
					headers.ContentType: []string{aurestclientapi.ContentTypeApplicationJson},
				},
			},
		},
		map[string]error{
			"GET http://err <nil>": errors.New("some transport error"),
		},
	)
	recordingMockClient := aurestcapture.New(mockClient)
	return recordingMockClient
}

func TestSuccessNoRetry(t *testing.T) {
	aulogging.SetupNoLoggerForTesting()

	mock := tstMock()
	cut := New(mock, 0,
		func(ctx context.Context, response *aurestclientapi.ParsedResponse, err error) bool {
			return true
		},
		nil)

	response := &aurestclientapi.ParsedResponse{}
	err := cut.Perform(context.Background(), "GET", "http://ok", nil, response)
	require.Nil(t, err)
	require.Equal(t, []string{"GET http://ok <nil>"}, aurestcapture.GetRecording(mock))
}

func TestSuccessWithRetry(t *testing.T) {
	aulogging.SetupNoLoggerForTesting()

	mock := tstMock()
	cut := New(mock, 2,
		func(ctx context.Context, response *aurestclientapi.ParsedResponse, err error) bool {
			return true
		},
		nil)

	response := &aurestclientapi.ParsedResponse{}
	err := cut.Perform(context.Background(), "GET", "http://ok", nil, response)
	require.Nil(t, err)
	r := "GET http://ok <nil>"
	require.Equal(t, []string{r, r, r}, aurestcapture.GetRecording(mock))
}

func TestSuccessWithNotNeededRetry(t *testing.T) {
	aulogging.SetupNoLoggerForTesting()

	mock := tstMock()
	cut := New(mock, 2,
		func(ctx context.Context, response *aurestclientapi.ParsedResponse, err error) bool {
			return false
		},
		nil)

	response := &aurestclientapi.ParsedResponse{}
	err := cut.Perform(context.Background(), "GET", "http://ok", nil, response)
	require.Nil(t, err)
	r := "GET http://ok <nil>"
	require.Equal(t, []string{r}, aurestcapture.GetRecording(mock))
}

func TestFailNoRetry(t *testing.T) {
	aulogging.SetupNoLoggerForTesting()

	mock := tstMock()
	cut := New(mock, 0,
		func(ctx context.Context, response *aurestclientapi.ParsedResponse, err error) bool {
			return true
		},
		nil)

	response := &aurestclientapi.ParsedResponse{}
	err := cut.Perform(context.Background(), "GET", "http://err", nil, response)
	require.Equal(t, "some transport error", err.Error())
	require.Equal(t, []string{"GET http://err <nil>"}, aurestcapture.GetRecording(mock))
}

func TestFailWithRetry(t *testing.T) {
	aulogging.SetupNoLoggerForTesting()

	retryCount := 0

	mock := tstMock()
	cut := New(mock, 3,
		func(ctx context.Context, response *aurestclientapi.ParsedResponse, err error) bool {
			return true
		},
		func(ctx context.Context, originalResponse *aurestclientapi.ParsedResponse, originalError error) error {
			retryCount++
			return nil
		})

	response := &aurestclientapi.ParsedResponse{}
	err := cut.Perform(context.Background(), "GET", "http://err", nil, response)
	require.NotNil(t, err)
	require.Equal(t, "some transport error", err.Error())
	r := "GET http://err <nil>"
	require.Equal(t, []string{r, r, r, r}, aurestcapture.GetRecording(mock))
}

func TestAbortRetry(t *testing.T) {
	aulogging.SetupNoLoggerForTesting()

	mock := tstMock()
	cut := New(mock, 2,
		func(ctx context.Context, response *aurestclientapi.ParsedResponse, err error) bool {
			return true
		},
		func(ctx context.Context, originalResponse *aurestclientapi.ParsedResponse, originalError error) error {
			return originalError
		})

	response := &aurestclientapi.ParsedResponse{}
	err := cut.Perform(context.Background(), "GET", "http://err", nil, response)
	require.NotNil(t, err)
	r := "GET http://err <nil>"
	require.Equal(t, []string{r}, aurestcapture.GetRecording(mock))
}
