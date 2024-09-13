package auresthttpclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	aurestnontripping "github.com/StephanHCB/go-autumn-restclient/implementation/errors/nontrippingerror"
	"github.com/go-http-utils/headers"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HttpClientImpl struct {
	HttpClient         *http.Client
	RequestManipulator aurestclientapi.RequestManipulatorCallback
	Timeout            time.Duration

	RequestMetricsCallback  aurestclientapi.MetricsCallbackFunction
	ResponseMetricsCallback aurestclientapi.MetricsCallbackFunction

	// Now is exposed so tests can fixate the time by overwriting this field
	Now func() time.Time
}

// New builds a new http client.
//
// Do not share between different logging/circuit breaker/retry stacks, but you should only build each stack once.
//
// timeout MUST be set to 0 if you use a circuit breaker or anything else that may do a context cancel, or you'll get weird behavior.
//
// If len(customCACert) is 0, the default CA certificates are used, but if you specify it, they are excluded to ensure
// only your certs are accepted.
func New(timeout time.Duration, customCACert []byte, requestManipulator aurestclientapi.RequestManipulatorCallback) (aurestclientapi.Client, error) {
	httpTransport := createHttpTransport(customCACert)

	return &HttpClientImpl{
		HttpClient: &http.Client{
			Transport: httpTransport,
			Timeout:   timeout,
		},
		RequestManipulator:      requestManipulator,
		Now:                     time.Now,
		RequestMetricsCallback:  doNothingMetricsCallback,
		ResponseMetricsCallback: doNothingMetricsCallback,
	}, nil
}

func createHttpTransport(customCACert []byte) http.RoundTripper {
	if len(customCACert) != 0 {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(customCACert)

		return &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		}
	}
	return http.DefaultTransport
}

// Instrument adds instrumentation to a http client.
//
// Either of the callbacks may be nil.
func Instrument(
	client aurestclientapi.Client,
	requestMetricsCallback aurestclientapi.MetricsCallbackFunction,
	responseMetricsCallback aurestclientapi.MetricsCallbackFunction,
) {
	httpClient, ok := client.(*HttpClientImpl)
	if !ok {
		return
	}

	if requestMetricsCallback != nil {
		httpClient.RequestMetricsCallback = requestMetricsCallback
	}
	if responseMetricsCallback != nil {
		httpClient.ResponseMetricsCallback = responseMetricsCallback
	}
}

func doNothingMetricsCallback(_ context.Context, _ string, _ string, _ int, _ error, _ time.Duration, _ int) {

}

func (c *HttpClientImpl) Perform(ctx context.Context, method string, requestUrl string, requestBody interface{}, response *aurestclientapi.ParsedResponse) error {
	requestBodyReader, length, contentType, err := c.requestBodyReader(requestBody)
	if err != nil {
		c.RequestMetricsCallback(ctx, method, requestUrl, 0, err, 0, length)
		return aurestnontripping.New(ctx, err)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestUrl, requestBodyReader)
	if err != nil {
		c.RequestMetricsCallback(ctx, method, requestUrl, 0, err, 0, length)
		return aurestnontripping.New(ctx, err)
	}

	if contentType != "" {
		req.Header.Set(headers.ContentType, contentType)
	}

	if c.RequestManipulator != nil {
		c.RequestManipulator(ctx, req)
	}

	c.RequestMetricsCallback(ctx, method, requestUrl, 0, nil, 0, length)

	response.Time = c.Now()

	responseInternal, err := c.HttpClient.Do(req)
	if err != nil {
		switch err.(type) {
		case *url.Error:
			return fmt.Errorf("url.Error received on http request: %s", err.Error())
		default:
			return fmt.Errorf("unexpected http error received: %s", err.Error())
		}
	}

	response.Header = responseInternal.Header
	response.Status = responseInternal.StatusCode

	responseBody, err := io.ReadAll(responseInternal.Body)
	if err != nil {
		_ = responseInternal.Body.Close()
		c.ResponseMetricsCallback(ctx, method, requestUrl, response.Status, err, c.Now().Sub(response.Time), 0)
		return err
	}

	err = responseInternal.Body.Close()
	if err != nil {
		c.ResponseMetricsCallback(ctx, method, requestUrl, response.Status, err, c.Now().Sub(response.Time), 0)
		return err
	}

	if len(responseBody) > 0 && response.Body != nil {
		switch response.Body.(type) {
		case **[]byte:
			*(response.Body.(**[]byte)) = &responseBody
		default:
			err := json.Unmarshal(responseBody, response.Body)
			if err != nil {
				c.ResponseMetricsCallback(ctx, method, requestUrl, response.Status, err, c.Now().Sub(response.Time), len(responseBody))
				return aurestnontripping.New(ctx, err)
			}
		}
		c.ResponseMetricsCallback(ctx, method, requestUrl, response.Status, nil, c.Now().Sub(response.Time), len(responseBody))
	} else {
		c.ResponseMetricsCallback(ctx, method, requestUrl, response.Status, nil, c.Now().Sub(response.Time), 0)
	}

	return nil
}

func (c *HttpClientImpl) requestBodyReader(requestBody interface{}) (io.Reader, int, string, error) {
	if requestBody == nil {
		return nil, 0, "", nil
	}
	if asCustom, ok := requestBody.(aurestclientapi.CustomRequestBody); ok {
		return asCustom.BodyReader, asCustom.BodyLength, asCustom.ContentType, nil
	}
	if asString, ok := requestBody.(string); ok {
		return strings.NewReader(asString), len(asString), aurestclientapi.ContentTypeApplicationJson, nil
	}
	if asUrlValues, ok := requestBody.(url.Values); ok {
		asString := asUrlValues.Encode()
		return strings.NewReader(asString), len(asString), aurestclientapi.ContentTypeApplicationXWwwFormUrlencoded, nil
	}

	marshalled, err := json.Marshal(requestBody)
	if err != nil {
		return nil, 0, "", err
	}
	return strings.NewReader(string(marshalled)), len(marshalled), aurestclientapi.ContentTypeApplicationJson, nil
}
