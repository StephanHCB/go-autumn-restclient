package aurestmock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
)

type MockImpl struct {
	recording     []string
	mockResponses map[string]aurestclientapi.ParsedResponse
	mockErrors    map[string]error
}

func New(mockResponses map[string]aurestclientapi.ParsedResponse, mockErrors map[string]error) aurestclientapi.Client {
	return &MockImpl{
		recording:     make([]string, 0),
		mockResponses: mockResponses,
		mockErrors:    mockErrors,
	}
}

func (c *MockImpl) Perform(ctx context.Context, method string, requestUrl string, requestBody interface{}, response *aurestclientapi.ParsedResponse) error {
	requestStr := fmt.Sprintf("%s %s %v", method, requestUrl, requestBody)
	c.recording = append(c.recording, requestStr)

	mockError, ok := c.mockErrors[requestStr]
	if ok {
		return mockError
	}

	mockResponse, ok := c.mockResponses[requestStr]
	if ok {
		response.Header = mockResponse.Header
		response.Status = mockResponse.Status
		if response.Body != nil && mockResponse.Body != nil {
			// copy over through json round trip
			marshalled, _ := json.Marshal(mockResponse.Body)
			_ = json.Unmarshal(marshalled, response.Body)
		}
		return nil
	} else {
		return errors.New("no mock error and also no mock response found - error")
	}
}

func GetRecording(mock aurestclientapi.Client) []string {
	mockImpl, ok := mock.(*MockImpl)
	if ok {
		return mockImpl.recording
	} else {
		return make([]string, 0)
	}
}

func ResetRecording(mock aurestclientapi.Client) {
	mockImpl, ok := mock.(*MockImpl)
	if ok {
		mockImpl.recording = make([]string, 0)
	}
}
