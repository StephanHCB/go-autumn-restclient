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
- support for plugging in a circuitbreaker (not included with this library, see [go-autumn-restclient-circuitbreaker](https://github.com/StephanHCB/go-autumn-restclient-circuitbreaker))
- conditional retry (using a callback so you're flexible about the retry condition)
- support for plugging in a request cache
- support for context aware request logging
- support for pre-request header/request manipulation (using a callback)
- auto marshalling and unmarshalling for both application/json (pass in a struct) and x-www-form-urlencoded (pass in url.Values)
- support for custom CA certificate chains (configuration per instance, so you can even have one instance with the
  default certs and one with a custom CA chain)
- integration with go-autumn-logging

## Usage

Please see the [examples](https://github.com/StephanHCB/go-autumn-restclient/tree/main/example).

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
