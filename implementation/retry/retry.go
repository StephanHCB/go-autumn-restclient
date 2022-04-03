package aurestretry

import (
	"context"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
)

type RetryImpl struct {
	Wrapped     aurestclientapi.Client
	RepeatCount uint8

	RetryCondition aurestclientapi.RetryConditionCallback
	BeforeRetry    aurestclientapi.BeforeRetryCallback
}

func New(
	wrapped aurestclientapi.Client,
	repeatCount uint8, // 0 means only try once (but then why use this at all?)
	condition aurestclientapi.RetryConditionCallback,
	beforeRetryOrNil aurestclientapi.BeforeRetryCallback,
) aurestclientapi.Client {
	return &RetryImpl{
		Wrapped:        wrapped,
		RepeatCount:    repeatCount,
		RetryCondition: condition,
		BeforeRetry:    beforeRetryOrNil,
	}
}

func (c *RetryImpl) Perform(ctx context.Context, method string, requestUrl string, requestBody interface{}, response *aurestclientapi.ParsedResponse) error {
	var attempt uint8
	var err error
	for attempt = 1; attempt <= c.RepeatCount+1; attempt++ {
		err = c.Wrapped.Perform(ctx, method, requestUrl, requestBody, response)

		if c.RetryCondition(ctx, response, err) {
			// (*)
			if attempt == c.RepeatCount+1 {
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
				aulogging.Logger.Ctx(ctx).Warn().WithErr(err2).Printf("before retry callback failed for %s %s after attempt %d", method, requestUrl, attempt)
				aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("caused by original error (see error field)")
				return err2
			}
		}
	}
	// this line is actually unreachable, see (*) but go doesn't understand this
	return err
}
