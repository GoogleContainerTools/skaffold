# dlog

Go library to parse the binary Docker Logs stream into plain text.

[![GoDoc](https://godoc.org/github.com/ahmetalpbalkan/dlog?status.svg)](https://godoc.org/github.com/ahmetalpbalkan/dlog)
[![Build Status](https://travis-ci.org/ahmetalpbalkan/dlog.svg?branch=master)](https://travis-ci.org/ahmetalpbalkan/dlog)
[![Coverage Status](https://coveralls.io/repos/github/ahmetalpbalkan/dlog/badge.svg?branch=master)](https://coveralls.io/github/ahmetalpbalkan/dlog?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/ahmetalpbalkan/dlog)](https://goreportcard.com/report/github.com/ahmetalpbalkan/dlog)

`dlog` offers a single method: `NewReader(r io.Reader) io.Reader`. You are
supposed to give the response body of the `/containers/<id>/logs`. The returned
reader strips off the log headers and just gives the plain text to be used.

Here is how a log line from container looks like in the  the raw docker logs
stream:

```text
01 00 00 00 00 00 00 1f 52 6f 73 65 73 20 61 72  65 ...
│  ─────┬── ─────┬─────  R  o  s  e  s     a  r   e ...
│       │        │
└stdout │        │
        │        └─ 0x0000001f = log message is 31 bytes
      unused
```

You can get the logs stream from [go-dockerclient][gocl]'s [`Logs()`][gocl-logs]
method, or by calling the [container logs endpoint][rapi] direclty via the UNIX socket
directly.

See [`example_test.go`](./example_test.go) for an example usage.

This library is written in vanilla Go and has no external dependencies.

[gocl]: https://github.com/fsouza/go-dockerclient
[gocl-logs]: https://godoc.org/github.com/fsouza/go-dockerclient#Client.Logs
[rapi]: https://docs.docker.com/engine/reference/api/docker_remote_api_v1.24/#/get-container-logs

-----

Licensed under Apache 2.0. Copyright 2017 [Ahmet Alp Balkan][ab].

[ab]: https://ahmetalpbalkan.com/
