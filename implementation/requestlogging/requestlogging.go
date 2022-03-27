package aurestlogging

import (
	"context"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	aurestnontripping "github.com/StephanHCB/go-autumn-restclient/implementation/errors/nontrippingerror"
	"time"
)

type RequestLoggingImpl struct {
	Wrapped aurestclientapi.Client
}

func New(wrapped aurestclientapi.Client) aurestclientapi.Client {
	return &RequestLoggingImpl{
		Wrapped: wrapped,
	}
}

func (c *RequestLoggingImpl) Perform(ctx context.Context, method string, requestUrl string, requestBody interface{}, response *aurestclientapi.ParsedResponse) error {
	aulogging.Logger.Ctx(ctx).Debug().Printf("downstream %s (%s)...", method, requestUrl)
	before := time.Now()
	err := c.Wrapped.Perform(ctx, method, requestUrl, requestBody, response)
	millis := time.Now().Sub(before).Milliseconds()
	if err != nil {
		if aurestnontripping.Is(err) {
			aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("downstream %s (%s) -> %d FAILED (%d ms) (nontripping)", method, requestUrl, response.Status, millis)
		} else {
			aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("downstream %s (%s) -> %d FAILED (%d ms)", method, requestUrl, response.Status, millis)
		}
	} else {
		aulogging.Logger.Ctx(ctx).Info().Printf("downstream %s (%s) -> %d OK (%d ms)", method, requestUrl, response.Status, millis)
	}
	return err
}
