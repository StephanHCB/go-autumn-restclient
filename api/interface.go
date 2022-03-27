package aurestclientapi

import (
	"context"
	"net/http"
)

var ContentTypeApplicationJson = "application/json"
var ContentTypeApplicationXWwwFormUrlencoded = "application/x-www-form-urlencoded"

type ParsedResponse struct {
	// Body is an optional reference that you can pre-fill with a reference to a suitable type, and we'll do tolerant reading
	Body    interface{}
	Status  int
	Header  http.Header
}

// Client is a utility class representing a http client.
//
// We provide multiple stackable implementations, typical stacking order is
// - actual http client
// - outgoing request logger
// - a circuit breaker (separate library, import go-autumn-restclient-gobreaker)
// - retry mechanism
type Client interface {
	// Perform makes a HTTP(s) call.
	//
	// If requestBody is nil, no body is sent. Must be nil for GET and OPTIONS.
	//
	// If a requestBody is given, it is json encoded and content type set to application/json, unless
	// you pass in url.Values, then we send x-www-form-urlencoded (for form post requests).
	Perform(ctx context.Context, method string, url string, requestBody interface{}, response *ParsedResponse) error
}

// RequestManipulatorCallback is an optional function you can provide that manipulates the http request
// before it is sent.
//
// Argument of goauresthttpclient.New(). It is allowed to pass in nil for no manipulator.
//
// Use this callback to set extra headers on the request, perhaps Authorization or pass on the Request Id for tracing.
// You can do anything with the request, really.
type RequestManipulatorCallback func(ctx context.Context, r *http.Request)

// RetryConditionCallback is a function you need to provide that determines whether a retry should be attempted.
//
// Argument of goaurestretry.New()
//
// Only useful if you actually place such a retry instance on the stack, which should go above the
// circuit breaker, if any.
//
// Note that once the context is cancelled, further requests fail immediately, so your timeout on the
// circuit breaker needs to leave enough room for possible retries.
type RetryConditionCallback func(ctx context.Context, response *ParsedResponse, err error) bool

// BeforeRetryCallback gets called with the response and error between a failure and a retry, but not before
// the first attempt.
//
// Argument of goaurestretry.New(). It is allowed to pass in nil for no callback.
//
// Only useful if you actually place a retry instance on the stack, which should go above the circuit breaker, if any.
//
// If you return an error, the retry won't proceed and the error gets returned. It is ok to pass through originalError.
type BeforeRetryCallback func(ctx context.Context, originalResponse *ParsedResponse, originalError error) error
