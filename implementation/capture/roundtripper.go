package aurestcapture

import (
	"fmt"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	"io"
	"net/http"
	"strings"
)

func NewRoundTripper(wrapped aurestclientapi.Client) http.RoundTripper {
	return &RequestCaptureImpl{Wrapped: wrapped}
}

func (c *RequestCaptureImpl) RoundTrip(req *http.Request) (*http.Response, error) {
	requestStr := fmt.Sprintf("%s %s %v", req.Method, req.URL.String(), req.Body)
	c.recording = append(c.recording, requestStr)

	var bodyDto *[]byte
	parsedResponse := aurestclientapi.ParsedResponse{
		Body: &bodyDto,
	}

	err := c.Wrapped.Perform(req.Context(), req.Method, req.URL.String(), req.Body, &parsedResponse)

	newReader := strings.NewReader(string(**(parsedResponse.Body.(**[]byte))))
	readCloser := io.NopCloser(newReader)

	return &http.Response{
		Status:           "",
		StatusCode:       parsedResponse.Status,
		Proto:            "",
		ProtoMajor:       0,
		ProtoMinor:       0,
		Header:           parsedResponse.Header,
		Body:             readCloser,
		ContentLength:    0,
		TransferEncoding: nil,
		Close:            false,
		Uncompressed:     false,
		Trailer:          nil,
		Request:          nil,
		TLS:              nil,
	}, err
}
