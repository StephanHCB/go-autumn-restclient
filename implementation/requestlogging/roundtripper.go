package aurestlogging

import (
	"net/http"
)

type LoggingRoundTripper struct {
	wrapped http.RoundTripper
	Options RequestLoggingOptions
}

func NewLoggingRoundTripper(wrapped http.RoundTripper) *LoggingRoundTripper {
	return NewLoggingRoundTripperWithOpts(wrapped, defaultOpts())
}

func NewLoggingRoundTripperWithOpts(wrapped http.RoundTripper, opts RequestLoggingOptions) *LoggingRoundTripper {
	instance := &LoggingRoundTripper{
		wrapped: wrapped,
		Options: defaultOpts(),
	}
	if opts.BeforeRequest != nil {
		instance.Options.BeforeRequest = opts.BeforeRequest
	}
	if opts.Success != nil {
		instance.Options.Success = opts.Success
	}
	if opts.Failure != nil {
		instance.Options.Failure = opts.Failure
	}
	return instance
}

func defaultOpts() RequestLoggingOptions {
	return RequestLoggingOptions{
		BeforeRequest: Debug,
		Success:       Info,
		Failure:       Warn,
	}
}

func (c *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	startTime := logRequest(req.Context(), req.Method, req.URL.String(), &c.Options)

	response, err := c.wrapped.RoundTrip(req)

	statusCode := 0
	if response != nil {
		statusCode = response.StatusCode
	}
	logResponse(req.Context(), req.Method, req.URL.String(), statusCode, err, startTime, &c.Options)

	return response, err
}
