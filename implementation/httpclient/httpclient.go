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
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HttpClientImpl struct {
	HttpClient         *http.Client
	RequestManipulator aurestclientapi.RequestManipulatorCallback
	Timeout            time.Duration
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
	if len(customCACert) != 0 {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(customCACert)

		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		}

		return &HttpClientImpl{
			HttpClient: &http.Client{
				Transport: transport,
				Timeout:   timeout,
			},
			RequestManipulator: requestManipulator,
		}, nil
	} else {
		return &HttpClientImpl{
			HttpClient: &http.Client{
				Timeout: timeout,
			},
			RequestManipulator: requestManipulator,
		}, nil
	}
}

func (c *HttpClientImpl) Perform(ctx context.Context, method string, requestUrl string, requestBody interface{}, response *aurestclientapi.ParsedResponse) error {
	requestBodyReader, contentType, err := c.requestBodyReader(requestBody)
	if err != nil {
		return aurestnontripping.New(ctx, err)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestUrl, requestBodyReader)
	if err != nil {
		return aurestnontripping.New(ctx, err)
	}

	if contentType != "" {
		req.Header.Set(headers.ContentType, contentType)
	}

	if c.RequestManipulator != nil {
		c.RequestManipulator(ctx, req)
	}

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

	responseBody, err := ioutil.ReadAll(responseInternal.Body)
	if err != nil {
		_ = responseInternal.Body.Close()
		return err
	}

	err = responseInternal.Body.Close()
	if err != nil {
		return err
	}

	if len(responseBody) > 0 && response.Body != nil {
		err := json.Unmarshal(responseBody, response.Body)
		if err != nil {
			return aurestnontripping.New(ctx, err)
		}
	}

	return nil
}

func (c *HttpClientImpl) requestBodyReader(requestBody interface{}) (io.Reader, string, error) {
	if requestBody == nil {
		return nil, "", nil
	}
	if asString, ok := requestBody.(string); ok {
		return strings.NewReader(asString), aurestclientapi.ContentTypeApplicationJson, nil
	}
	if asUrlValues, ok := requestBody.(url.Values); ok {
		return strings.NewReader(asUrlValues.Encode()), aurestclientapi.ContentTypeApplicationXWwwFormUrlencoded, nil
	}

	marshalled, err := json.Marshal(requestBody)
	if err != nil {
		return nil, "", err
	}
	return strings.NewReader(string(marshalled)), aurestclientapi.ContentTypeApplicationJson, nil
}
