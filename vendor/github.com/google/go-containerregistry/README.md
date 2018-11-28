# go-containerregistry

[![Build Status](https://travis-ci.org/google/go-containerregistry.svg?branch=master)](https://travis-ci.org/google/go-containerregistry)
[![GoDoc](https://godoc.org/github.com/google/go-containerregistry?status.svg)](https://godoc.org/github.com/google/go-containerregistry)
[![Go Report Card](https://goreportcard.com/badge/google/go-containerregistry)](https://goreportcard.com/report/google/go-containerregistry)
[![Code Coverage](https://codecov.io/gh/google/go-containerregistry/branch/master/graph/badge.svg)](https://codecov.io/gh/google/go-containerregistry)


## Introduction

This is a golang library for working with container registries. It's largely based on the [Python library of the same name](https://github.com/google/containerregistry), but more hip and uses GitHub as the source of truth.

## Tools

This repo hosts three tools built on top of the library.

### ko

[`ko`](cmd/ko/README.md) is a tool for building and deploying golang applications to kubernetes.

### crane

[`crane`](cmd/crane/doc/crane.md) is a tool for interacting with remote images and registries.

### gcrane

[`gcrane`](cmd/gcrane/README.md) is a GCR-specific variant of `crane` that has richer output for
the `ls` subcommand and some basic garbage collection support.
