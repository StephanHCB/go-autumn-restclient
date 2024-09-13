package auresthttpclient

import (
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	"net/http"
	"time"
)

type AuRestHttpClient struct {
	*http.Client

	// Now is exposed so tests can fixate the time by overwriting this field
	Now func() time.Time
}

func NewHttpClient(timeout time.Duration, customCACert []byte,
	requestManipulator aurestclientapi.RequestManipulatorCallback, customHttpTransport *http.RoundTripper) (*AuRestHttpClient, error) {

	var httpTransport http.RoundTripper
	if customHttpTransport == nil {
		httpTransport = &HttpClientRoundTripper{
			wrapped:                 createHttpTransport(customCACert),
			RequestManipulator:      requestManipulator,
			RequestMetricsCallback:  doNothingMetricsCallback,
			ResponseMetricsCallback: doNothingMetricsCallback,
		}
	} else {
		httpTransport = *customHttpTransport
	}

	return &AuRestHttpClient{
		Client: &http.Client{
			Transport: httpTransport,
			Timeout:   timeout,
		},
		Now: time.Now,
	}, nil
}

type HttpClientRoundTripper struct {
	wrapped http.RoundTripper

	RequestManipulator      aurestclientapi.RequestManipulatorCallback
	RequestMetricsCallback  aurestclientapi.MetricsCallbackFunction
	ResponseMetricsCallback aurestclientapi.MetricsCallbackFunction
}

func (c *HttpClientRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if c.RequestManipulator != nil {
		c.RequestManipulator(req.Context(), req)
	}

	c.RequestMetricsCallback(req.Context(), req.Method, req.URL.String(), 0, nil, 0, int(req.ContentLength))

	response, err := c.wrapped.RoundTrip(req)

	c.ResponseMetricsCallback(req.Context(), req.Method, req.URL.String(), 0, nil, 0, int(req.ContentLength))

	return response, err
}
