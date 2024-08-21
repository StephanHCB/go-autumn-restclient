package aurestrecorder

import (
	"bytes"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	"io"
	"net/http"
	"strings"
	"time"
)

type RecorderRoundTripper struct {
	wrapped               http.RoundTripper
	recorderPath          string
	constructFilenameFunc ConstructFilenameFunction
}

func NewRecorderRoundTripper(wrapped http.RoundTripper, additionalOptions ...RecorderOptions) *RecorderRoundTripper {
	recorderPath, filenameFunc := initRecorderPathAndFilenameFunc(additionalOptions)
	return &RecorderRoundTripper{
		wrapped:               wrapped,
		recorderPath:          recorderPath,
		constructFilenameFunc: filenameFunc,
	}
}

func (c *RecorderRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := c.wrapped.RoundTrip(req)

	if response != nil && c.recorderPath != "" {
		parsedResponse := aurestclientapi.ParsedResponse{
			Body:   string(readBodyAndReset(response)),
			Status: response.StatusCode,
			Header: response.Header,
			Time:   time.Now(),
		}

		var requestBodyString string
		var requestBody io.ReadCloser
		if req.Body != nil {
			requestBody, _ = req.GetBody()
			requestBodyString = readBody(requestBody)
		}
		recordResponseData(req.Method, req.URL.String(), requestBodyString, &parsedResponse, err, c.recorderPath, c.constructFilenameFunc)
	}
	return response, err
}

func readBodyAndReset(res *http.Response) []byte {
	bodyBytes, _ := io.ReadAll(res.Body)
	//reset the response body to the original unread state
	res.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return bodyBytes
}

func readBody(requestBody io.ReadCloser) string {
	if requestBody != nil {
		buf := new(strings.Builder)
		_, _ = io.Copy(buf, requestBody)
		return buf.String()
	}
	return ""
}
