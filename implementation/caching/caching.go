package aurestcaching

import (
	"context"
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
	RetentionTime                 time.Duration
	Cache                         *tinylru.LRU
}

type CacheEntry struct {
	Recorded time.Time
	Response *aurestclientapi.ParsedResponse
}

func New(
	wrapped aurestclientapi.Client,
	useCacheCondition aurestclientapi.CacheConditionCallback,
	storeResponseInCacheCondition aurestclientapi.CacheResponseConditionCallback,
	retentionTime time.Duration,
	cacheSize int,
) aurestclientapi.Client {
	cache := &tinylru.LRU{}
	cache.Resize(cacheSize)

	return &CachingImpl{
		Wrapped:                       wrapped,
		UseCacheCondition:             useCacheCondition,
		StoreResponseInCacheCondition: storeResponseInCacheCondition,
		RetentionTime:                 retentionTime,
		Cache:                         cache,
	}
}

func (c *CachingImpl) Perform(ctx context.Context, method string, requestUrl string, requestBody interface{}, response *aurestclientapi.ParsedResponse) error {
	canCache := c.UseCacheCondition(ctx, method, requestUrl, requestBody)
	if canCache {
		key := fmt.Sprintf("%s %s %v", method, requestUrl, requestBody)
		cachedResponseRaw, ok := c.Cache.Get(key)
		if ok {
			cachedResponse, ok := cachedResponseRaw.(CacheEntry)
			if ok {
				age := time.Now().Sub(cachedResponse.Recorded)
				if age < c.RetentionTime {
					aulogging.Logger.Ctx(ctx).Info().Printf("downstream %s (%s) -> %d cached %d seconds ago", method, requestUrl, response.Status, age.Seconds())
					response.Header = cachedResponse.Response.Header
					response.Status = cachedResponse.Response.Status
					response.Body = cachedResponse.Response.Body
					return nil
				} else {
					c.Cache.Delete(key)
				}
			}
		}
		err := c.Wrapped.Perform(ctx, method, requestUrl, requestBody, response)
		if err == nil && canCache && c.StoreResponseInCacheCondition(ctx, method, requestUrl, requestBody, response) {
			_, _ = c.Cache.Set(key, CacheEntry{
				Recorded: time.Now(),
				Response: response,
			})
		}
		return err
	} else {
		return c.Wrapped.Perform(ctx, method, requestUrl, requestBody, response)
	}
}
