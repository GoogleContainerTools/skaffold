Contributing to go-yaml
=======================

Thank you for your interest in contributing to go-yaml!

This document provides guidelines and instructions for contributing to this
project.


## Code of Conduct

By participating in this project, you agree to follow our Code of Conduct.

We expect all contributors to:
- Be respectful and inclusive
- Use welcoming and inclusive language
- Be collaborative and constructive
- Focus on what is best for both the Go and YAML communities


## How to Contribute


### Reporting Issues

Before submitting an issue, please:
- Check if the issue already exists in our issue tracker
- Use a clear and descriptive title
- Provide detailed steps to reproduce the issue
- Include relevant code samples and error messages
- Specify your Go version and operating system


### Pull Requests

1. Fork the repository
1. Create a new branch for your changes
1. Make your changes following our coding conventions
   - If you are not sure about the coding conventions, please ask
   - Look at the existing code for examples
1. Write clear commit messages
1. Update tests and documentation
1. Submit a pull request


### Coding Conventions

- Follow standard Go coding conventions
- Use `gofmt` to format your code
- Write descriptive comments for non-obvious code
- Add tests for your work
- Keep line length to 80 characters
- Use meaningful variable and function names
- Start doc and comment sentences on a new line


### Commit Conventions

- No merge commits
- Commit subject line should:
  - Start with a capital letter
  - Not end with a period
  - Be no more than 50 characters


### Testing

- Ensure all tests pass
- Add new tests for new functionality
- Update existing tests when modifying functionality


## Development Setup

- Install Go (see [go.mod](https://github.com/yaml/go-yaml/blob/main/go.mod) for
  minimum required version)
- Fork and clone the repository
- Make your changes
- Run tests and linters


## Using the Makefile

The repository contains a `GNUmakefile` that provides a number of useful
targets:

- `make test` runs the tests
- `make test v=1 count=3` runs the tests with options
- `make test GO-VERSION=1.23.4` runs the tests with a specific Go version
- `make shell` opens a shell with the project's dependencies set up
- `make shell GO-VERSION=1.23.4` opens a shell with a specific Go version
- `make fmt` runs `go fmt`
- `make tidy` runs `go mod tidy`
- `make install` runs `go install`
- `make distclean` cleans the project completely


## Getting Help

If you need help, you can:
- Open an issue with your question
- Read through our documentation
- Join our [Slack channel](https://cloud-native.slack.com/archives/C08PPAT8PS7)


## We are a Work in Progress

This project is very much a team effort.
We are just getting things rolling and trying to get the foundations in place.
There are lots of opinions and ideas about how to do things, even within the
core team.

Once our process is more mature, we will likely change the rules here.
We'll make the new rules as a team.
For now, please stick to the rules as they are.

This project is focused on serving the needs of both the Go and YAML
communities.
Sometimes those needs can be in conflict, but we'll try to find common ground.


## Thank You

Thank you for contributing to go-yaml!
