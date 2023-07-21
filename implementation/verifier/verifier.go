package aurestverifier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type VerifierImpl struct {
	expectations    []Expectation
	firstUnexpected *Request
}

type Request struct {
	Name   string // key for the request
	Method string
	Header http.Header // currently not tested, just supplied as documentation for now
	Url    string
	Body   interface{}
}

type ResponseOrError struct {
	Response aurestclientapi.ParsedResponse
	Error    error
}

type Expectation struct {
	Request  Request
	Response ResponseOrError
	Matched  bool
}

func (e Expectation) matches(method string, requestUrl string, requestBody interface{}) bool {
	// this is a very simple "must match 100%" for the first version
	urlMatches := e.Request.Url == requestUrl
	methodMatches := e.Request.Method == method
	bodyMatches := requestBodyAsString(e.Request.Body) == requestBodyAsString(requestBody)

	return urlMatches && methodMatches && bodyMatches
}

func requestBodyAsString(requestBody interface{}) string {
	if requestBody == nil {
		return ""
	}
	if asCustom, ok := requestBody.(aurestclientapi.CustomRequestBody); ok {
		if b, err := io.ReadAll(asCustom.BodyReader); err == nil {
			return string(b)
		} else {
			return fmt.Sprintf("ERROR: %s", err.Error())
		}
	}
	if asString, ok := requestBody.(string); ok {
		return asString
	}
	if asUrlValues, ok := requestBody.(url.Values); ok {
		asString := asUrlValues.Encode()
		return asString
	}

	marshalled, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Sprintf("ERROR: %s", err.Error())
	}
	return string(marshalled)
}

func headersSortedAsString(spec http.Header) string {
	var result strings.Builder

	sortedKeys := make([]string, 0)
	for k, _ := range spec {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	for _, k := range sortedKeys {
		result.WriteString(fmt.Sprintf("%s: ", k))
		for _, v := range spec[k] {
			result.WriteString(fmt.Sprintf("%s ", v))
		}
	}
	return result.String()
}

func New() (aurestclientapi.Client, *VerifierImpl) {
	instance := &VerifierImpl{
		expectations: make([]Expectation, 0),
	}
	return instance, instance
}

func (c *VerifierImpl) Perform(ctx context.Context, method string, requestUrl string, requestBody interface{}, response *aurestclientapi.ParsedResponse) error {
	expected, err := c.currentExpectation(method, requestUrl, requestBody)
	if err != nil {
		return err
	}

	if expected.Response.Error != nil {
		return expected.Response.Error
	}

	mockResponse := expected.Response.Response

	response.Header = mockResponse.Header
	response.Status = mockResponse.Status
	response.Time = mockResponse.Time
	if response.Body != nil && mockResponse.Body != nil {
		// copy over through json round trip
		marshalled, _ := json.Marshal(mockResponse.Body)
		_ = json.Unmarshal(marshalled, response.Body)
	}

	return nil
}

func (c *VerifierImpl) currentExpectation(method string, requestUrl string, requestBody interface{}) (Expectation, error) {
	for i, e := range c.expectations {
		if !e.Matched {
			if e.matches(method, requestUrl, requestBody) {
				c.expectations[i].Matched = true
				return e, nil
			} else {
				if c.firstUnexpected == nil {
					c.firstUnexpected = &Request{
						Name:   fmt.Sprintf("unmatched expectation %d - %s", i+1, e.Request.Name),
						Method: method,
						Header: nil, // not currently available
						Url:    requestUrl,
						Body:   requestBody,
					}
				}
				return Expectation{}, fmt.Errorf("unmatched expectation %d - %s", i+1, e.Request.Name)
			}
		}
	}

	if c.firstUnexpected == nil {
		c.firstUnexpected = &Request{
			Name:   fmt.Sprintf("no expectations remaining - unexpected request at end"),
			Method: method,
			Header: nil, // not currently available
			Url:    requestUrl,
			Body:   requestBody,
		}
	}
	return Expectation{}, errors.New("no expectations remaining - unexpected request at end")
}

func (c *VerifierImpl) AddExpectation(requestMatcher Request, response aurestclientapi.ParsedResponse, err error) {
	c.expectations = append(c.expectations, Expectation{
		Request: requestMatcher,
		Response: ResponseOrError{
			Response: response,
			Error:    err,
		},
	})
}

func (c *VerifierImpl) FirstUnexpectedOrNil() *Request {
	return c.firstUnexpected
}
