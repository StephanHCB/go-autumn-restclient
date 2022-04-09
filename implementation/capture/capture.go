package aurestcapture

import (
	"context"
	"fmt"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
)

type RequestCaptureImpl struct {
	Wrapped   aurestclientapi.Client
	recording []string
}

func New(wrapped aurestclientapi.Client) aurestclientapi.Client {
	return &RequestCaptureImpl{
		Wrapped:   wrapped,
		recording: make([]string, 0),
	}
}

func (c *RequestCaptureImpl) Perform(ctx context.Context, method string, requestUrl string, requestBody interface{}, response *aurestclientapi.ParsedResponse) error {
	requestStr := fmt.Sprintf("%s %s %v", method, requestUrl, requestBody)
	c.recording = append(c.recording, requestStr)
	return c.Wrapped.Perform(ctx, method, requestUrl, requestBody, response)
}

func GetRecording(this aurestclientapi.Client) []string {
	captureImpl, ok := this.(*RequestCaptureImpl)
	if ok {
		return captureImpl.recording
	} else {
		return make([]string, 0)
	}
}

func ResetRecording(this aurestclientapi.Client) {
	captureImpl, ok := this.(*RequestCaptureImpl)
	if ok {
		captureImpl.recording = make([]string, 0)
	}
}
