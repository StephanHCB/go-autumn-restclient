package aurestlogging

import (
	"context"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	auloggingapi "github.com/StephanHCB/go-autumn-logging/api"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	aurestnontripping "github.com/StephanHCB/go-autumn-restclient/implementation/errors/nontrippingerror"
	"time"
)

// RequestLoggingOptions allows overriding the log functions used.
//
// This allows easily changing the log level when setting up request logging.
//
// important: do not cache the LeveledLoggingImplementation, create one each time, or some loggers may use
// cached values.
type RequestLoggingOptions struct {
	BeforeRequest func(ctx context.Context) auloggingapi.LeveledLoggingImplementation
	Success       func(ctx context.Context) auloggingapi.LeveledLoggingImplementation
	Failure       func(ctx context.Context) auloggingapi.LeveledLoggingImplementation
}

func Debug(ctx context.Context) auloggingapi.LeveledLoggingImplementation {
	return aulogging.Logger.Ctx(ctx).Debug()
}

func Info(ctx context.Context) auloggingapi.LeveledLoggingImplementation {
	return aulogging.Logger.Ctx(ctx).Info()
}

func Warn(ctx context.Context) auloggingapi.LeveledLoggingImplementation {
	return aulogging.Logger.Ctx(ctx).Warn()
}

type RequestLoggingImpl struct {
	Wrapped aurestclientapi.Client
	Options RequestLoggingOptions
}

func NewWithOptions(wrapped aurestclientapi.Client, opts RequestLoggingOptions) aurestclientapi.Client {
	instance := &RequestLoggingImpl{
		Wrapped: wrapped,
		Options: RequestLoggingOptions{
			BeforeRequest: Debug,
			Success:       Info,
			Failure:       Warn,
		},
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

func New(wrapped aurestclientapi.Client) aurestclientapi.Client {
	return &RequestLoggingImpl{
		Wrapped: wrapped,
		Options: RequestLoggingOptions{
			BeforeRequest: Debug,
			Success:       Info,
			Failure:       Warn,
		},
	}
}

func (c *RequestLoggingImpl) Perform(ctx context.Context, method string, requestUrl string, requestBody interface{}, response *aurestclientapi.ParsedResponse) error {
	c.Options.BeforeRequest(ctx).Printf("downstream %s %s...", method, requestUrl)
	before := time.Now()
	err := c.Wrapped.Perform(ctx, method, requestUrl, requestBody, response)
	millis := time.Now().Sub(before).Milliseconds()
	if err != nil {
		if aurestnontripping.Is(err) {
			c.Options.Failure(ctx).WithErr(err).Printf("downstream %s %s -> %d FAILED (%d ms) (nontripping)", method, requestUrl, response.Status, millis)
		} else {
			c.Options.Failure(ctx).WithErr(err).Printf("downstream %s %s -> %d FAILED (%d ms)", method, requestUrl, response.Status, millis)
		}
	} else {
		c.Options.Success(ctx).Printf("downstream %s %s -> %d OK (%d ms)", method, requestUrl, response.Status, millis)
	}
	return err
}
