package aurestcaching

import (
	"context"
	"encoding/json"
	"fmt"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	"github.com/tidwall/tinylru"
	"time"
)

type CachingImpl struct {
	Wrapped aurestclientapi.Client

	UseCacheCondition             aurestclientapi.CacheConditionCallback
	StoreResponseInCacheCondition aurestclientapi.CacheResponseConditionCallback
	CacheKeyFunction              aurestclientapi.CacheKeyFunction
	RetentionTime                 time.Duration
	Cache                         *tinylru.LRU

	CacheHitMetricsCallback     aurestclientapi.MetricsCallbackFunction
	CacheMissMetricsCallback    aurestclientapi.MetricsCallbackFunction
	CacheInvalidMetricsCallback aurestclientapi.MetricsCallbackFunction

	// Now is exposed so tests can fixate the time by overwriting this field
	Now func() time.Time
}

type CacheEntry struct {
	Recorded           time.Time
	ResponseHeaderJson []byte
	ResponseBodyJson   []byte
	ResponseStatus     int
}

func New(
	wrapped aurestclientapi.Client,
	useCacheCondition aurestclientapi.CacheConditionCallback,
	storeResponseInCacheCondition aurestclientapi.CacheResponseConditionCallback,
	cacheKeyFunction aurestclientapi.CacheKeyFunction,
	retentionTime time.Duration,
	cacheSize int,
) aurestclientapi.Client {
	cache := &tinylru.LRU{}
	cache.Resize(cacheSize)

	if cacheKeyFunction == nil {
		cacheKeyFunction = defaultKeyFunction
	}

	return &CachingImpl{
		Wrapped:                       wrapped,
		UseCacheCondition:             useCacheCondition,
		StoreResponseInCacheCondition: storeResponseInCacheCondition,
		CacheKeyFunction:              cacheKeyFunction,
		RetentionTime:                 retentionTime,
		Cache:                         cache,
		Now:                           time.Now,
		CacheHitMetricsCallback:       doNothingMetricsCallback,
		CacheMissMetricsCallback:      doNothingMetricsCallback,
		CacheInvalidMetricsCallback:   doNothingMetricsCallback,
	}
}

// Instrument adds instrumentation to a http client.
//
// Either of the callbacks may be nil.
func Instrument(
	client aurestclientapi.Client,
	cacheHitMetricsCallback aurestclientapi.MetricsCallbackFunction,
	cacheMissMetricsCallback aurestclientapi.MetricsCallbackFunction,
	cacheInvalidMetricsCallback aurestclientapi.MetricsCallbackFunction,
) {
	cachingClient, ok := client.(*CachingImpl)
	if !ok {
		return
	}

	if cacheHitMetricsCallback != nil {
		cachingClient.CacheHitMetricsCallback = cacheHitMetricsCallback
	}
	if cacheMissMetricsCallback != nil {
		cachingClient.CacheMissMetricsCallback = cacheMissMetricsCallback
	}
	if cacheInvalidMetricsCallback != nil {
		cachingClient.CacheInvalidMetricsCallback = cacheInvalidMetricsCallback
	}
}

func doNothingMetricsCallback(_ context.Context, _ string, _ string, _ int, _ error, _ time.Duration, _ int) {

}

func defaultKeyFunction(_ context.Context, method string, requestUrl string, _ interface{}) string {
	return fmt.Sprintf("%s %s", method, requestUrl)
}

func (c *CachingImpl) Perform(ctx context.Context, method string, requestUrl string, requestBody interface{}, response *aurestclientapi.ParsedResponse) error {
	canCache := c.UseCacheCondition(ctx, method, requestUrl, requestBody)
	if canCache {
		key := c.CacheKeyFunction(ctx, method, requestUrl, requestBody)
		cachedResponseRaw, ok := c.Cache.Get(key)
		if ok {
			cachedResponse, ok := cachedResponseRaw.(CacheEntry)
			if ok {
				age := c.Now().Sub(cachedResponse.Recorded)
				if age < c.RetentionTime {
					err := json.Unmarshal(cachedResponse.ResponseBodyJson, response.Body)
					err2 := json.Unmarshal(cachedResponse.ResponseHeaderJson, &response.Header)
					response.Status = cachedResponse.ResponseStatus
					response.Time = cachedResponse.Recorded
					if err == nil && err2 == nil {
						// cache successfully used
						c.CacheHitMetricsCallback(ctx, method, requestUrl, response.Status, nil, 0, len(cachedResponse.ResponseBodyJson))
						aulogging.Logger.Ctx(ctx).Info().Printf("downstream %s %s -> %d cached %d seconds ago", method, requestUrl, response.Status, age.Milliseconds()/1000)
						return nil
					} else {
						// invalid cache entry -- delete
						c.CacheInvalidMetricsCallback(ctx, method, requestUrl, 0, err, 0, 0)
						aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("downstream %s %s -> %d cache FAIL, see error -- deleting cache entry", method, requestUrl, response.Status)
						c.Cache.Delete(key)
					}
				} else {
					// cache miss - entry there but too old
					c.CacheMissMetricsCallback(ctx, method, requestUrl, 0, nil, 0, 0)
					c.Cache.Delete(key)
				}
			} else {
				// invalid cache entry -- delete
				c.CacheInvalidMetricsCallback(ctx, method, requestUrl, 0, nil, 0, 0)
				aulogging.Logger.Ctx(ctx).Error().Printf("downstream %s %s -> %d cache FAIL, invalid type -- deleting cache entry", method, requestUrl, response.Status)
				c.Cache.Delete(key)
			}
		} else {
			// cache miss
			c.CacheMissMetricsCallback(ctx, method, requestUrl, 0, nil, 0, 0)
		}
		err := c.Wrapped.Perform(ctx, method, requestUrl, requestBody, response)
		if err == nil && canCache && c.StoreResponseInCacheCondition(ctx, method, requestUrl, requestBody, response) {
			bodyJson, err := json.Marshal(response.Body)
			headerJson, err2 := json.Marshal(&response.Header)
			status := response.Status
			if err == nil && err2 == nil {
				_, _ = c.Cache.Set(key, CacheEntry{
					Recorded:           response.Time,
					ResponseBodyJson:   bodyJson,
					ResponseHeaderJson: headerJson,
					ResponseStatus:     status,
				})
			}
		}
		return err
	} else {
		return c.Wrapped.Perform(ctx, method, requestUrl, requestBody, response)
	}
}
