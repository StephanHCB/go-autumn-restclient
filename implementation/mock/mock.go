package aurestmock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	"time"
)

type MockImpl struct {
	mockResponses map[string]aurestclientapi.ParsedResponse
	mockErrors    map[string]error
	// Now is exposed so tests can fixate the time by overwriting this field
	Now func() time.Time
}

func New(mockResponses map[string]aurestclientapi.ParsedResponse, mockErrors map[string]error) aurestclientapi.Client {
	return &MockImpl{
		mockResponses: mockResponses,
		mockErrors:    mockErrors,
		Now:           time.Now,
	}
}

func (c *MockImpl) Perform(ctx context.Context, method string, requestUrl string, requestBody interface{}, response *aurestclientapi.ParsedResponse) error {
	requestStr := fmt.Sprintf("%s %s %v", method, requestUrl, requestBody)

	mockError, ok := c.mockErrors[requestStr]
	if ok {
		return mockError
	}

	mockResponse, ok := c.mockResponses[requestStr]
	if ok {
		response.Header = mockResponse.Header
		response.Status = mockResponse.Status
		response.Time = c.Now()
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
