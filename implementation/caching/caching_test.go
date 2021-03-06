package aurestcaching

import (
	"context"
	"errors"
	"fmt"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	aurestcapture "github.com/StephanHCB/go-autumn-restclient/implementation/capture"
	aurestmock "github.com/StephanHCB/go-autumn-restclient/implementation/mock"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func tstMock() aurestclientapi.Client {
	mockClient := aurestmock.New(
		map[string]aurestclientapi.ParsedResponse{
			"GET http://ok <nil>": {
				Body:   nil,
				Status: 200,
				Header: map[string][]string{
					headers.ContentType: []string{aurestclientapi.ContentTypeApplicationJson},
				},
			},
			"GET http://cache-me <nil>": {
				Body:   []string{"first", "second"},
				Status: 200,
				Header: map[string][]string{
					headers.ContentType: []string{aurestclientapi.ContentTypeApplicationJson},
				},
			},
			"GET http://notfound <nil>": {
				Body:   nil,
				Status: 400,
				Header: map[string][]string{},
			},
			"GET http://cache-me/err <nil>": {
				Body:   nil,
				Status: 500,
				Header: map[string][]string{
					headers.ContentType: []string{aurestclientapi.ContentTypeApplicationJson},
				},
			},
			"GET http://cache-me/404 <nil>": {
				Body:   nil,
				Status: 404,
				Header: map[string][]string{
					headers.ContentType: []string{aurestclientapi.ContentTypeApplicationJson},
				},
			},
		},
		map[string]error{
			"GET http://err <nil>":          errors.New("some transport error"),
			"GET http://cache-me/err <nil>": errors.New("some transport error"),
		},
	)
	recordingMockClient := aurestcapture.New(mockClient)
	return recordingMockClient
}

func tstCut(mock aurestclientapi.Client) aurestclientapi.Client {
	return New(mock,
		func(ctx context.Context, method string, url string, requestBody interface{}) bool {
			return strings.Contains(url, "cache-me")
		},
		func(ctx context.Context, method string, url string, requestBody interface{}, response *aurestclientapi.ParsedResponse) bool {
			return strings.Contains(url, "cache-me") && response.Status == 200
		},
		nil,
		100*time.Millisecond,
		10,
	)
}

func TestSuccessCacheHitAndMiss(t *testing.T) {
	aulogging.SetupNoLoggerForTesting()

	mock := tstMock()
	cut := tstCut(mock)

	aurestcapture.ResetRecording(mock)
	response := &aurestclientapi.ParsedResponse{
		Body: &[]string{},
	}
	err := cut.Perform(context.Background(), "GET", "http://cache-me", nil, response)
	require.Nil(t, err)
	require.Equal(t, "&[first second]", fmt.Sprintf("%v", response.Body))
	require.Equal(t, []string{"GET http://cache-me <nil>"}, aurestcapture.GetRecording(mock))

	// now do something else
	aurestcapture.ResetRecording(mock)
	response = &aurestclientapi.ParsedResponse{}
	err = cut.Perform(context.Background(), "GET", "http://ok", nil, response)
	require.Nil(t, err)
	require.Equal(t, []string{"GET http://ok <nil>"}, aurestcapture.GetRecording(mock))

	// now try a second time, this time should hit the cache, so we shouldn't see a request, but get the response
	aurestcapture.ResetRecording(mock)
	response = &aurestclientapi.ParsedResponse{
		Body: &[]string{},
	}
	err = cut.Perform(context.Background(), "GET", "http://cache-me", nil, response)
	require.Nil(t, err)
	require.Equal(t, "&[first second]", fmt.Sprintf("%v", response.Body))
	require.Equal(t, []string{}, aurestcapture.GetRecording(mock))
}

func TestErrorsAreNotCached(t *testing.T) {
	aulogging.SetupNoLoggerForTesting()

	mock := tstMock()
	cut := tstCut(mock)

	aurestcapture.ResetRecording(mock)
	response := &aurestclientapi.ParsedResponse{}
	err := cut.Perform(context.Background(), "GET", "http://cache-me/err", nil, response)
	require.NotNil(t, err)
	require.Equal(t, []string{"GET http://cache-me/err <nil>"}, aurestcapture.GetRecording(mock))

	// now try a second time, should go out again because the cache doesn't store failed requests
	aurestcapture.ResetRecording(mock)
	response = &aurestclientapi.ParsedResponse{}
	err = cut.Perform(context.Background(), "GET", "http://cache-me/err", nil, response)
	require.NotNil(t, err)
	require.Equal(t, []string{"GET http://cache-me/err <nil>"}, aurestcapture.GetRecording(mock))
}

func TestStoreResponseConditionWorks(t *testing.T) {
	aulogging.SetupNoLoggerForTesting()

	mock := tstMock()
	cut := tstCut(mock)

	aurestcapture.ResetRecording(mock)
	response := &aurestclientapi.ParsedResponse{}
	err := cut.Perform(context.Background(), "GET", "http://cache-me/404", nil, response)
	require.Nil(t, err)
	require.Equal(t, []string{"GET http://cache-me/404 <nil>"}, aurestcapture.GetRecording(mock))
	require.Equal(t, 404, response.Status)

	// now try a second time, should go out again because the store condition was false (we set it up not to store != 200)
	aurestcapture.ResetRecording(mock)
	response = &aurestclientapi.ParsedResponse{}
	err = cut.Perform(context.Background(), "GET", "http://cache-me/404", nil, response)
	require.Nil(t, err)
	require.Equal(t, []string{"GET http://cache-me/404 <nil>"}, aurestcapture.GetRecording(mock))
	require.Equal(t, 404, response.Status)
}

func TestCacheDeletion(t *testing.T) {
	aulogging.SetupNoLoggerForTesting()

	mock := tstMock()
	cut := tstCut(mock)

	aurestcapture.ResetRecording(mock)
	response := &aurestclientapi.ParsedResponse{
		Body: &[]string{},
	}
	err := cut.Perform(context.Background(), "GET", "http://cache-me", nil, response)
	require.Nil(t, err)
	require.Equal(t, "&[first second]", fmt.Sprintf("%v", response.Body))
	require.Equal(t, []string{"GET http://cache-me <nil>"}, aurestcapture.GetRecording(mock))

	// now try a second time, this time should hit the cache, so we shouldn't see a request, but get the response
	aurestcapture.ResetRecording(mock)
	response = &aurestclientapi.ParsedResponse{
		Body: &[]string{},
	}
	err = cut.Perform(context.Background(), "GET", "http://cache-me", nil, response)
	require.Nil(t, err)
	require.Equal(t, "&[first second]", fmt.Sprintf("%v", response.Body))
	require.Equal(t, []string{}, aurestcapture.GetRecording(mock))

	// now wait until cache entry has aged
	time.Sleep(110 * time.Millisecond)

	// now try a third time, this time should go out again, because cache age
	aurestcapture.ResetRecording(mock)
	response = &aurestclientapi.ParsedResponse{
		Body: &[]string{},
	}
	err = cut.Perform(context.Background(), "GET", "http://cache-me", nil, response)
	require.Nil(t, err)
	require.Equal(t, "&[first second]", fmt.Sprintf("%v", response.Body))
	require.Equal(t, []string{"GET http://cache-me <nil>"}, aurestcapture.GetRecording(mock))
}

func TestCacheDeletionCorruptJson(t *testing.T) {
	aulogging.SetupNoLoggerForTesting()

	mock := tstMock()
	cut := tstCut(mock)

	aurestcapture.ResetRecording(mock)
	response := &aurestclientapi.ParsedResponse{
		Body: &[]string{},
	}
	err := cut.Perform(context.Background(), "GET", "http://cache-me", nil, response)
	require.Nil(t, err)
	require.Equal(t, "&[first second]", fmt.Sprintf("%v", response.Body))
	require.Equal(t, []string{"GET http://cache-me <nil>"}, aurestcapture.GetRecording(mock))

	// now let's manipulate the cache so the json is syntactically invalid
	cache := cut.(*CachingImpl).Cache
	key := defaultKeyFunction(nil, "GET", "http://cache-me", nil)
	entryRaw, ok := cache.Get(key)
	require.True(t, ok)
	entry := entryRaw.(CacheEntry)
	entry.ResponseBodyJson = []byte("not a valid json")
	cache.Set(key, entry)

	// now try a second time. Since the cache entry is invalid json, parsing it will fail and the request
	// will go out despite the cache entry
	aurestcapture.ResetRecording(mock)
	response = &aurestclientapi.ParsedResponse{
		Body: &[]string{},
	}
	err = cut.Perform(context.Background(), "GET", "http://cache-me", nil, response)
	require.Nil(t, err)
	require.Equal(t, "&[first second]", fmt.Sprintf("%v", response.Body))
	require.Equal(t, []string{"GET http://cache-me <nil>"}, aurestcapture.GetRecording(mock))

	// and the entry should have been removed from the cache (but we can't test this because it will just
	// have been added again)
}
