package aurestretry

import (
	"context"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	"time"
)

type RetryImpl struct {
	Wrapped     aurestclientapi.Client
	RepeatCount uint8

	RetryCondition aurestclientapi.RetryConditionCallback
	BeforeRetry    aurestclientapi.BeforeRetryCallback

	RetryingMetricsCallback aurestclientapi.MetricsCallbackFunction
	GivingUpMetricsCallback aurestclientapi.MetricsCallbackFunction
}

func New(
	wrapped aurestclientapi.Client,
	repeatCount uint8, // 0 means only try once (but then why use this at all?)
	condition aurestclientapi.RetryConditionCallback,
	beforeRetryOrNil aurestclientapi.BeforeRetryCallback,
) aurestclientapi.Client {
	return &RetryImpl{
		Wrapped:                 wrapped,
		RepeatCount:             repeatCount,
		RetryCondition:          condition,
		BeforeRetry:             beforeRetryOrNil,
		RetryingMetricsCallback: doNothingMetricsCallback,
		GivingUpMetricsCallback: doNothingMetricsCallback,
	}
}

// Instrument adds instrumentation to a http client.
//
// Either of the callbacks may be nil.
func Instrument(
	client aurestclientapi.Client,
	retryingMetricsCallback aurestclientapi.MetricsCallbackFunction,
	givingUpMetricsCallback aurestclientapi.MetricsCallbackFunction,
) {
	retryingClient, ok := client.(*RetryImpl)
	if !ok {
		return
	}

	if retryingMetricsCallback != nil {
		retryingClient.RetryingMetricsCallback = retryingMetricsCallback
	}
	if givingUpMetricsCallback != nil {
		retryingClient.GivingUpMetricsCallback = givingUpMetricsCallback
	}
}

func doNothingMetricsCallback(_ context.Context, _ string, _ string, _ int, _ error, _ time.Duration, _ int) {

}

func (c *RetryImpl) Perform(ctx context.Context, method string, requestUrl string, requestBody interface{}, response *aurestclientapi.ParsedResponse) error {
	var attempt uint8
	var err error
	for attempt = 1; attempt <= c.RepeatCount+1; attempt++ {
		err = c.Wrapped.Perform(ctx, method, requestUrl, requestBody, response)

		if c.RetryCondition(ctx, response, err) {
			// (*)
			if attempt == c.RepeatCount+1 {
				c.GivingUpMetricsCallback(ctx, method, requestUrl, response.Status, err, 0, 0)
				aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("giving up on %s %s after attempt %d", method, requestUrl, attempt)
				return err
			}
		} else {
			// no retry needed
			return err
		}

		if c.BeforeRetry != nil {
			err2 := c.BeforeRetry(ctx, response, err)
			if err2 != nil {
				c.GivingUpMetricsCallback(ctx, method, requestUrl, response.Status, err2, 0, 0)
				aulogging.Logger.Ctx(ctx).Warn().WithErr(err2).Printf("before retry callback failed for %s %s after attempt %d", method, requestUrl, attempt)
				aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("caused by original error (see error field)")
				return err2
			}
		}
		c.RetryingMetricsCallback(ctx, method, requestUrl, response.Status, nil, 0, 0)
	}
	// this line is actually unreachable, see (*) but go doesn't understand this
	return err
}
