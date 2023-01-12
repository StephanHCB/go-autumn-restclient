package aurestclientapi

import (
	"context"
	"io"
	"net/http"
	"time"
)

var ContentTypeApplicationJson = "application/json"
var ContentTypeApplicationXWwwFormUrlencoded = "application/x-www-form-urlencoded"

type ParsedResponse struct {
	// Body is an optional reference that you can pre-fill with a reference to a suitable type, and we'll do tolerant reading
	Body   interface{}
	Status int
	Header http.Header
	// Time is set to the time the request was made. Even when it comes from cache, it will be set to the original time.
	Time time.Time
}

// CustomRequestBody allows you the greatest amount of control over the request by directly supplying the
// body io.Reader, the length, and the content type header.
type CustomRequestBody struct {
	BodyReader  io.Reader // Tip: &bytes.Buffer{} implements io.Reader
	BodyLength  int
	ContentType string
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
// Argument of auresthttpclient.New(). It is allowed to pass in nil for no manipulator.
//
// Use this callback to set extra headers on the request, perhaps Authorization or pass on the Request Id for tracing.
// You can do anything with the request, really.
type RequestManipulatorCallback func(ctx context.Context, r *http.Request)

// RetryConditionCallback is a function you need to provide that determines whether a retry should be attempted.
//
// Argument of aurestretry.New()
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
// Argument of aurestretry.New(). It is allowed to pass in nil for no callback.
//
// Only useful if you actually place a retry instance on the stack, which should go above the circuit breaker, if any.
//
// If you return an error, the retry won't proceed and the error gets returned. It is ok to pass through originalError.
type BeforeRetryCallback func(ctx context.Context, originalResponse *ParsedResponse, originalError error) error

// CacheConditionCallback gets called to determine whether a given request should be looked up
// in the cache before making it for real.
//
// Argument of aurestcaching.New(). Cannot be nil.
//
// Return true to visit the cache (warning, may give you old state and not resend identical requests).
//
// Return false to always make the real request.
type CacheConditionCallback func(ctx context.Context, method string, url string, requestBody interface{}) bool

// CacheResponseConditionCallback gets called after a successful request (no error) to determine
// whether the result should be cached.
//
// This only gets called if CacheConditionCallback was already true, because otherwise the request is not relevant
// for the cache.
//
// Argument of aurestcaching.New(). Cannot be nil.
//
// Return true to put the response in the cache.
//
// Return false to not store it (maybe it's a 404 or something).
type CacheResponseConditionCallback func(ctx context.Context, method string, url string, requestBody interface{}, response *ParsedResponse) bool

// CacheKeyFunction allows you to override the default key generator function, which is based only on
// method and url.
//
// Argument of aurestcaching.New(). May be nil.
//
// You must return a unique string that is used as the cache key. If two requests map to the same key, you will get
// weird behaviour.
type CacheKeyFunction func(ctx context.Context, method string, url string, requestBody interface{}) string

// MetricsCallbackFunction allows you to instrument the http client stack with callbacks in a variety of
// places. Designed for metrics collection, but of course you can use it for lots of other stuff.
//
// Argument of various Instrument() calls. May always be nil.
//
// Not all parameters will always be set. For example, the latency is only known at the request logging level,
// and the request/response body size is only known while that is being processed.
type MetricsCallbackFunction func(ctx context.Context, method string, url string, status int, err error, latency time.Duration, size int)
