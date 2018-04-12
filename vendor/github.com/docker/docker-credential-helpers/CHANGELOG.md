# Changelog

This changelog tracks the releases of docker-credential-helpers.
This project includes different binaries per platform.
The platform released is identified after the tag name.

## v0.6.0 (Go client, Linux)

- New credential helper on Linux using `pass`
- New entry point for passing environment variables when calling a credential helper
- Add a Makefile rule generating a Windows release binary

### Note

`pass` needs to be configured for `docker-credential-pass` to work properly.
It must be initialized with a `gpg2` key ID. Make sure your GPG key exists is in `gpg2` keyring as `pass` uses `gpg2` instead of the regular `gpg`.

## v0.5.2 (Mac OS X, Windows, Linux)

- Add a `version` command to output the version
- Fix storing URLs without scheme, and use `https://` by default

## v0.5.1 (Go client, Mac OS X, Windows, Linux)

- Redirect credential helpers' standard error to the caller's
- Prevent invalid credentials and credentials queries

## v0.5.0 (Mac OS X)

- Add a label for Docker credentials and filter credentials lookup to filter keychain lookups

## v0.4.2 (Mac OS X, Windows)

- Fix osxkeychain list
- macOS binary is now signed on release
- Generate a `.exe` instead

## v0.4.1 (Mac OS X)

- Fixes to support older version of OSX (10.10, 10.11)

## v0.4.0 (Go client, Mac OS X, Windows, Linux)

- Full implementation for OSX ready
- Fix some windows issues
- Implement client.List, change list API
- mac: delete credentials before adding them to avoid already exist error (fixes #37)

## v0.3.0 (Go client)

- Add Go client library to talk with the native programs.

## v0.2.0 (Mac OS X, Windows, Linux)

- Initial release of docker-credential-secretservice for Linux.
- Use new secrets payload introduced in https://github.com/docker/docker/pull/20970.

## v0.1.0 (Mac OS X, Windows)

- Initial release of docker-credential-osxkeychain for Mac OS X.
- Initial release of docker-credential-wincred for Microsoft Windows.
