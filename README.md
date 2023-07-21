# go-autumn-restclient

A rest client that combines ease of use with support for a number of resilience features. 

## About go-autumn

A collection of libraries for [enterprise microservices](https://github.com/StephanHCB/go-mailer-service/blob/master/README.md) in golang that
- is heavily inspired by Spring Boot / Spring Cloud
- is very opinionated
- names modules by what they do
- unlike Spring Boot avoids certain types of auto-magical behaviour
- is not a library monolith, that is every part only depends on the api parts of the other components
  at most, and the api parts do not add any dependencies.  

Fall is my favourite season, so I'm calling it go-autumn.

## About go-autumn-restclient

It's a rest client that also supports x-www-form-urlencoded.

Each of the individual features isn't spectacular, but in combination I've found it easy and convenient to use.

### Features

- very small dependency footprint
- can use multiple instances with different configurations 
- context aware, including cancel and shutdown
- support for timeouts both at the httpclient and higher levels
- support for plugging in a circuitbreaker (not included with this library, see
  [go-autumn-restclient-circuitbreaker](https://github.com/StephanHCB/go-autumn-restclient-circuitbreaker))
- conditional retry (using a callback so you're flexible about the retry condition)
- support for plugging in a request cache
- support for context aware request logging
- support for pre-request header/request manipulation (using a callback)
- auto marshalling and unmarshalling for both application/json (pass in a struct) and x-www-form-urlencoded (pass in url.Values)
- support for custom CA certificate chains (configuration per instance, so you can even have one instance with the
  default certs and one with a custom CA chain)
- integration with go-autumn-logging (gives you logging framework independence)
- recorder that writes responses into files in a directory (useful for creating real world test cases)
- playback for these recorder files (useful for integration tests)
- request capture (useful for testing)

## Usage

### Construct your client stack

Since all modules implement the same interface, `Client`, you can stack them during the
construction phase.

Let's go through the individual components in the most useful order, and take a look at the
parameters of each `New()` constructor.

Please also see the [examples](https://github.com/StephanHCB/go-autumn-restclient/tree/main/example).

#### 1. The actual http client

At the bottom of the stack you will either have a `httpclient`, or a `playback` or `mock` (in testing).

The regular httpclient is set up as follows:

```
    var timeout time.Duration = 0
    var customCACert []byte = nil
    var requestManipulator aurestclientapi.RequestManipulatorCallback = nil
    
    httpClient, err := auresthttpclient.New(timeout, customCACert, requestManipulator)
    if err != nil {
        return err
    }
```

_If you have the [circuit breaker](https://github.com/StephanHCB/go-autumn-restclient-circuitbreaker)
in your stack, make sure that you set `timeout=0`, or else you will confuse the circuit breaker. 
It has a timeout, too, and will correctly cancel the supplied context and open the circuit breaker
if too many timeouts occur._

_`customCaCert` is a pem certificate. Due to some limitations of the golang http client that I have not yet found a
way to work around, you may need to supply an intermediate certificate here instead of an actual root CA. Or just
include all your certificates in `/etc/ssl/certs` and set this to `nil`._

_The `requestManipulator` callback allows you to make changes to requests, such as inject authorization or
request id headers._

#### 1a. Or use playback (testing with file recordings)

The playback client doesn't actually make requests, instead it reads responses from pre-recorded json files.
This includes header values and http status, and even the errors returned by the http client.

Useful for integration testing.

```
    playbackClient := aurestplayback.New("../resources/http-recordings/")
```

_The only parameter is the path to a directory that contains the recordings._

#### 1b. Or use mock (testing with in-memory responses)

The mock client also doesn't make actual requests, but unlike the playback client, 
you set up all mock responses in-memory at creation time by passing them to the constructor.

Useful for unit testing.

```
    // keyed by fmt.Sprintf("%s %s %v", method, requestUrl, requestBody)
    var mockResponses map[string]aurestclientapi.ParsedResponse = ...
    var mockErrors map[string]error = ...
    
    mockClient := aurestmock.New(mockResponses, mockErrors)
```

#### 1c. Or use verifier (super-basic consumer interaction testing)

This mock client doesn't make actual requests, but instead you set up a list of
expected interactions.

This allows doing very simple consumer tests, useful mostly for their documentation value.

#### 2. Response recording

If your tests use Option 1a (playback), you should insert a response recorder in your production stack.

This will not do anything unless you set an environment variable (`GO_AUTUMN_RESTCLIENT_RECORDER_PATH` by default, 
but it's exposed as a variable so you can customize it). 

If the variable is set, recording files will be written into the directory it points to.

```
    recorderClient := aurestrecorder.New(httpClient)
```

#### 2a. Request capture

This will provide your tests with a recording of requests made, allowing you to assert which requests
were made, in which order.

Only useful for testing.

```
	requestCaptureClient := aurestcapture.New(playbackClient) // or mockClient
```

#### 3. Request logging

This adds http downstream request logging, using [StephanHCB/go-autumn-logging](https://github.com/StephanHCB/go-autumn-logging).

Please read the notes on logging below. Application authors will need to pick a logging implementation, but if you
are writing a library you should NOT do this.

```
    requestLoggingClient := aurestlogging.New(recorderClient)
```

#### 4. Circuit breaker

Circuit breaker is implemented in a separate library because it brings extra dependencies along.

Import [StephanHCB/go-autumn-restclient-circuitbreaker](https://github.com/StephanHCB/go-autumn-restclient-circuitbreaker).

```
    cbClient := aurestbreaker.New(requestLoggingClient, <a bunch of parameters go here, see New()>)
```

#### 5. Retry

At this point, you can add a retry mechanism to the stack. You need to provide a callback that determines
whether a retry is needed. It should return true only if the request should be retried.

You can pass in a second optional callback to be invoked before a retry (but not before the first attempt). 
If this returns an error, that error is passed through and the retry is aborted. It is common to log a message
in this callback. Just pass in `nil` if you do not need it.

_Note you do not need to take care of counting retries. The implementation will still invoke the condition
callback, but then return control to you. This is the main difference between the two callbacks, after
the maximum number of attempts, the retry `condition` is still evaluated but the `beforeRetry` callback is not made._

```
    var repeatCount uint8 = 2 // 0 means only try once (but then why use this at all?)
    var condition aurestclientapi.RetryConditionCallback = func(ctx context.Context, response *aurestclientapi.ParsedResponse, err error) bool {
        return response.Status == http.StatusRequestTimeout
    }
    var beforeRetry aurestclientapi.BeforeRetryCallback = nil
    
    retryingClient := aurestretry.New(requestLoggingClient, repeatCount, condition, beforeRetry)
    
    // or if you have the circuit breaker in the stack, pass in cbClient instead of requestLoggingClient
    // retryingClient := aurestretry.New(cbClient, repeatCount, condition, beforeRetry)
```

## Logging

This library uses the [StephanHCB/go-autumn-logging](https://github.com/StephanHCB/go-autumn-logging) api for
logging framework independent logging.

### Library Authors

If you are writing a library, do NOT import any of the go-autumn-logging-* modules that actually bring in a logging library.
You will deprive application authors of their chance to pick the logging framework of their choice.

In your testing code, call `aulogging.SetupNoLoggerForTesting()` to avoid the nil pointer dereference.

### Application Authors

If you are writing an application, import one of the modules that actually bring in a logging library,
such as go-autumn-logging-zerolog. These modules will provide an implementation and place it in the Logger variable.

Of course, you can also provide your own implementation of the `LoggingImplementation` interface, just
set the `Logger` global singleton to an instance of your implementation.

Then just use the Logger, both during application runtime and tests.
