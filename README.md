# resp
> Fast Redis protocol parser package for Go.

[![GoDoc](https://godoc.org/github.com/nussjustin/resp?status.svg)](https://godoc.org/github.com/nussjustin/resp)
[![Build Status](https://travis-ci.org/nussjustin/resp.svg?branch=master)](https://travis-ci.org/nussjustin/resp)
[![Go Report Card](https://goreportcard.com/badge/github.com/nussjustin/resp)](https://goreportcard.com/report/github.com/nussjustin/resp)
[![codecov](https://codecov.io/gh/nussjustin/resp/branch/master/graph/badge.svg)](https://codecov.io/gh/nussjustin/resp)

Provides fast reader and writer types for the Redis RESP protocol.

## Installation

```sh
go get -u github.com/nussjustin/resp
```

## Testing

To run all unit tests, just call `go test`:

```sh
go test
```

If you want to run integration tests you need to pass the `integration` tag to `go test`:

```sh
go test -tags integration
```

By default integration tests will try to connect to a Redis instance on `127.0.0.1:6379`.

If your instance has a non-default config, you can use the `REDIS_HOST` environment variable, to override the address:

```sh
REDIS_HOST=127.0.0.1:6380   go test -tags integration # different port
REDIS_HOST=192.168.0.1:6380 go test -tags integration # different host
REDIS_HOST=/tmp/redis.sock  go test -tags integration # unix socket
```

Note: If you want to test using a unix socket, make sure that the path to the socket starts with a slash,
We.g. `/tmp/redis.sock`.

## Release History

* 0.0.0
    * Work in progress

## Meta

Justin Nuß – [@nussjustin](https://twitter.com/nussjustin)

Distributed under the MIT license. See ``LICENSE`` for more information.

[https://github.com/nussjustin/resp](https://github.com/nussjustin/)

## Contributing

1. Fork it (<https://github.com/nussjustin/resp/fork>)
2. Create your feature branch (`git checkout -b feature/fooBar`)
3. Commit your changes (`git commit -am 'Add some fooBar'`)
4. Push to the branch (`git push origin feature/fooBar`)
5. Create a new Pull Request