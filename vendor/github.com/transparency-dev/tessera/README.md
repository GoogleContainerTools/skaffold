# Tessera

[![Go Report Card](https://goreportcard.com/badge/github.com/transparency-dev/tessera)](https://goreportcard.com/report/github.com/transparency-dev/tessera)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/transparency-dev/tessera/badge)](https://scorecard.dev/viewer/?uri=github.com/transparency-dev/tessera)
[![Benchmarks](https://img.shields.io/badge/Benchmarks-blue.svg)](https://transparency-dev.github.io/tessera/dev/bench/)
[![Slack Status](https://img.shields.io/badge/Slack-Chat-blue.svg)](https://transparency-dev.slack.com/)

Tessera is a Go library for building [tile-based transparency logs (tlogs)](https://c2sp.org/tlog-tiles).
It is the logical successor to the approach [Trillian v1][] takes in building and operating logs.

The implementation and its APIs bake-in
[current best-practices based on the lessons learned](https://transparency.dev/articles/tile-based-logs/)
over the past decade of building and operating transparency logs in production environments and at scale.

Tessera was introduced at the Transparency.Dev summit in October 2024.
Watch [Introducing Tessera](https://www.youtube.com/watch?v=9j_8FbQ9qSc) for all the details,
but here's a summary of the high level goals:

*   [tlog-tiles API][] and storage
*   Support for both cloud and on-premises infrastructure
    *   [GCP](./storage/gcp/)
    *   [AWS](./storage/aws/)
    *   [MySQL](./storage/mysql/)
    *   [POSIX](./storage/posix/)
*   Make it easy to build and deploy new transparency logs on supported infrastructure
    *   Library instead of microservice architecture
    *   No additional services to manage
    *   Lower TCO for operators compared with Trillian v1
*   Fast sequencing and integration of entries
*   Optional functionality which can be enabled for those ecosystems/logs which need it (only pay the cost for what you need):
    *   "Best-effort" de-duplication of entries
    *   Synchronous integration
*   Broadly similar write-throughput and write-availability, and potentially _far_ higher read-throughput
    and read-availability compared to Trillian v1 (dependent on underlying infrastructure)
*   Enable building of arbitrary log personalities, including support for the peculiarities of a
    [Static CT API][] compliant log.

The main non-goal is to support transparency logs using anything other than the [tlog-tiles API][].
While it is possible to deploy a custom personality in front of Tessera that adapts the tlog-tiles API
into any other API, this strategy will lose a lot of the read scaling that Tessera is designed for.

## Table of Contents

- [Status](#status)
- [Roadmap](#roadmap)
- [Concepts](#concepts)
- [Usage](#usage)
  - [Getting Started](#getting-started)
  - [Writing Personalities](#writing-personalities)
- [Features](#features)
- [Lifecycles](#lifecycles)
- [Contributing](#contributing)
- [License](#license)
- [Contact](#contact)

## Status

Tessera is under active development, and is considered production ready since the
[Beta release](https://github.com/transparency-dev/tessera/releases/tag/v0.2.0).
See the table below for details.

### Storage drivers

| Driver                  | Appender | Migration | Antispam | Garbage Collection | Notes                                         |
| ----------------------- | :------: | :-------: | :------: | :----------------: | --------------------------------------------- |
| Amazon Web Services     |    ✅    |     ⚠️    |    ✅    |          ✅        |                                               |
| Google Cloud Platform   |    ✅    |     ⚠️    |    ✅    |          ✅        |                                               |
| POSIX filesystem        |    ✅    |     ⚠️    |    ✅    |          ✅        |                                               |
| MySQL                   |    ⚠️    |     ⚠️    |    ❌    |          N/A       | MySQL will remain in BETA for the time being. |


> [!Note]
> Please get in touch if you are interested in using any of the features or drivers held back in BETA above.

Users of GCP, AWS, MySQL, and POSIX are welcome to try the relevant [Getting Started](#getting-started) guide.

## Roadmap

Production ready around mid 2025.

|  #  | Step                                                      | Status |
| :-: | --------------------------------------------------------- | :----: |
|  1  | Drivers for GCP, AWS, MySQL, and POSIX                    |   ✅   |
|  2  | [tlog-tiles API][] support                                |   ✅   |
|  3  | Example code and terraform scripts for easy onboarding    |   ✅   |
|  4  | Stable API                                                |   ✅   |
|  5  | Data migration between releases                           |   ✅   |
|  6  | Data migration between drivers                            |   ✅   |
|  7  | Witness support                                           |   ✅   |
|  8  | Monitoring and metrics                                    |   ✅   |
|  9  | Production ready                                          |   ✅   |
|  10 | Mirrored logs (#576)                                      |   ⚠️   |
|  11 | Preordered logs (#575)                                    |   ❌   |
|  12 | Trillian v1 to Tessera migration (#577)                   |   ❌   |
|  N  | Fancy features (to be expanded upon later)                |   ❌   |

The current API is unlikely to change in any significant way, however the API is subject to minor breaking changes until we tag 1.0.

### What’s happening to Trillian v1?

[Trillian v1][] is still in use in production environments by
multiple organisations in multiple ecosystems, and is likely to remain so for the mid-term. 

New ecosystems, or existing ecosystems looking to evolve, should strongly consider planning a
migration to Tessera and adopting the patterns it encourages.

> [!Tip]
> To achieve the full benefits of Tessera, logs must use the [tlog-tiles API][].

## Concepts

This section introduces concepts and terms that will be used throughout the user guide.

### Sequencing

When data is added to a log, it is first stored in memory for some period (this can be controlled via the [batching options](https://pkg.go.dev/github.com/transparency-dev/tessera#WithBatching)).
If the process dies in this state, the entry will be lost.

Once a batch of entries is processed by the sequencer, the new data will transition from a volatile state to one where it is durably assigned an index.
If the process dies in this state, the entry will be safe, though it will not be available through the read API of the log until the leaf has been [Integrated](#integration).
Once an index number has been issued to a leaf, no other data will ever be issued the same index number.
All index numbers are contiguous and start from 0.

> [!IMPORTANT]
> Within a batch, there is no guarantee about which order index numbers will be assigned.
> The only way to ensure that sequential calls to `Add` are given sequential indices is by blocking until a sequencing batch is completed.
> This can be achieved by configuring a batch size of 1, though this will make sequencing expensive!

### Integration

Integration is a background process that happens when a Tessera lifecycle object has been created.
This process takes sequenced entries and merges them into the log.
Once this process has been completed, a new entry will:
 - Be available via the read API at the index that was returned from sequencing
 - Have Merkle tree hashes that commit to this data being included in the tree

### Publishing

Publishing is a background process that creates a new Checkpoint for the latest tree.
This background process runs periodically (configurable via [WithCheckpointInterval](https://pkg.go.dev/github.com/transparency-dev/tessera#AppendOptions.WithCheckpointInterval)) and performs the following steps:
  1. Create a new Checkpoint and sign it with the signer provided by [WithCheckpointSigner](https://pkg.go.dev/github.com/transparency-dev/tessera#AppendOptions.WithCheckpointSigner)
  2. Contact witnesses and collect enough countersignatures to satisfy any witness policy configured by [WithWitnesses](https://pkg.go.dev/github.com/transparency-dev/tessera#AppendOptions.WithWitnesses)
  3. If the witness policy is satisfied, make this new Checkpoint public available

An entry is considered published once it is committed to by a published Checkpoint (i.e. a published Checkpoint's size is larger than the entry's assigned index).
Due to the nature of append-only logs, all Checkpoints issued after this point will also commit to inclusion of this entry.

## Usage

### Getting Started

The best place to start is the [codelab](./cmd/conformance#codelab). 
This will walk you through setting up your first log, writing some entries to it via HTTP, and inspecting the contents.

Take a look at the example personalities in the `/cmd/` directory:
  - [posix](./cmd/conformance/posix/): example of operating a log backed by a local filesystem
    - This example runs an HTTP web server that takes arbitrary data and adds it to a file-based log.
  - [mysql](./cmd/conformance/mysql/): example of operating a log that uses MySQL
    - This example is easiest deployed via `docker compose`, which allows for easy setup and teardown.
  - [gcp](./cmd/conformance/gcp/): example of operating a log running in GCP.
    - This example can be deployed via terraform, see the [deployment instructions](./deployment/live/gcp/conformance#manual-deployment).
  - [aws](./cmd/conformance/aws/): example of operating a log running on AWS.
    - This example can be deployed via terraform, see the [deployment instructions](./deployment/live/aws/codelab#aws-codelab-deployment).
  - [posix-oneshot](./cmd/examples/posix-oneshot/): example of a command line tool to add entries to a log stored on the local filesystem
    - This example is not a long-lived process; running the command integrates entries into the log which lives only as files.

The `main.go` files for each of these example personalities try to strike a balance when demonstrating features of Tessera between simplicity, and demonstrating best practices.
Please raise issues against the repo, or chat to us in [Slack](#contact) if you have ideas for making the examples more accessible!

### Writing Personalities

#### Introduction

Tessera is a library written in Go.
It is designed to efficiently serve logs that allow read access via the [tlog-tiles API][].
The code you write that calls Tessera is referred to as a personality, because it tailors the generic library to your ecosystem.

Before starting to write your own personality, it is strongly recommended that you have familiarized yourself with the provided personalities referenced in [Getting Started](#getting-started).
When writing your Tessera personality, the first decision you need to make is which of the native drivers to use:
 *   [GCP](./storage/gcp/)
 *   [AWS](./storage/aws/)
 *   [MySQL](./storage/mysql/)
 *   [POSIX](./storage/posix/)

The easiest drivers to operate and to scale are the cloud implementations: GCP and AWS.
These are the recommended choice for the majority of users running in production.

If you aren't using a cloud provider, then your options are MySQL and POSIX:
- POSIX is the simplest to get started with as it needs little in the way of extra infrastructure, and
  if you already serve static files as part of your business/project this could be a good fit.
- Alternatively, if you are used to operating user-facing applications backed by a RDBMS, then MySQL could
  be a natural fit.

To get a sense of the rough performance you can expect from the different backends, take a look at
[docs/performance.md](/docs/performance.md).


#### Setup

Once you've picked a storage driver, you can start writing your personality!
You'll need to import the Tessera library:
```shell
# This imports the library at main.
# This should be set to the latest release version to get a stable release.
go get github.com/transparency-dev/tessera@main
```

#### Constructing the Appender

Import the main `tessera` package, and the driver for the storage backend you want to use:
```go file=README_test.go region=common_imports
	"github.com/transparency-dev/tessera"

	// Choose one!
	"github.com/transparency-dev/tessera/storage/posix"
	// "github.com/transparency-dev/tessera/storage/aws"
	// "github.com/transparency-dev/tessera/storage/gcp"
	// "github.com/transparency-dev/tessera/storage/mysql"

```

Now you'll need to instantiate the lifecycle object for the native driver you are using.

By far the most common way to operate logs is in an append-only manner, and the rest of this guide will discuss
this mode.
For lifecycle states other than Appender mode, take a look at [Lifecycles](#lifecycles) below.

Here's an example of creating an `Appender` for the POSIX driver:
```go file=README_test.go region=construct_example
	driver, _ := posix.New(ctx, "/tmp/mylog")
	signer := createSigner()

	appender, shutdown, reader, err := tessera.NewAppender(
		ctx, driver, tessera.NewAppendOptions().WithCheckpointSigner(signer))
```

See the documentation for each driver implementation to understand the parameters that each takes.

The final part of configuring Tessera is to set up the addition features that you want to use.
These optional libraries can be used to provide common log behaviours.
See [Features](#features) after reading the rest of this section for more details.

#### Writing to the Log

Now you should have a Tessera instance configured for your environment with the correct features set up.
Now the fun part - writing to the log!

```go file=README_test.go region=use_appender_example
	appender, shutdown, reader, err := tessera.NewAppender(
		ctx, driver, tessera.NewAppendOptions().WithCheckpointSigner(signer))
	if err != nil {
		panic(err)
	}

	index, err := appender.Add(ctx, tessera.NewEntry(data))()
```

The `AppendOptions` allow Tessera behaviour to be tuned.
Take a look at the methods named `With*` on the `AppendOptions` struct in the root package, e.g. [`WithBatching`](https://pkg.go.dev/github.com/transparency-dev/tessera@main#AppendOptions.WithBatching) to see the available options are how they should be used.

Writing to the log follows this flow:
 1. Call `Add` with a new entry created with the data to be added as a leaf in the log.
    - This method returns a _future_ of the form `func() (Index, error)`.
 2. Call this future function, which will block until the data passed into `Add` has been sequenced
    - On success, an index number is _durably_ assigned and returned
    - On failure, the error is returned
    
Once an index has been returned, the new data is sequenced, but not necessarily integrated into the log.

As discussed above in [Integration](#integration), sequenced entries will be _asynchronously_ integrated into the log and be made available via the read API.
Some personalities may need to block until this has been performed, e.g. because they will provide the requester with an inclusion proof, which requires integration.
Such personalities are recommended to use [Synchronous Publication](#synchronous-publication) to perform this blocking.

#### Reading from the Log

Data that has been written to the log needs to be made available for clients and verifiers.
Tessera makes the log readable via the [tlog-tiles API][].
In the case of AWS and GCP, the data to be served is written to object storage and served directly by the cloud provider.
The log operator only needs to ensure that these object storage instances are publicly readable, and set up a URL to point to them.

In the case of MySQL and POSIX, the log operator will need to take more steps to make the data available.
POSIX writes out the files exactly as per the API spec, so the log operator can serve these via an HTTP File Server.

MySQL is the odd implementation in that it requires personality code to handle read traffic.
See the example personalities written for MySQL to see how this Go web server should be configured.

## Features

### Antispam

In some scenarios, particularly where logs are publicly writable such as Certificate Transparency, it's possible for logs to be asked,
whether maliciously or accidentally, to add entries they already contain. Generally, this is undesirable, and so Tessera provides an
optional mechanism to try to detect and ignore duplicate entries on a best-effort basis.

Logs that do not allow public submissions directly to the log may want to operate without this optional antispam measure, instead relying on the
personality to never generate duplicates. This can allow for significantly cheaper operation and faster write throughput.

The antispam mechanism consists of two layers which sit in front of the underlying `Add` implementation of the storage:
1. The first layer is an `InMemory` cache which keeps track of a configurable number of recently-added entries.
   If a recently-seen entry is spotted by the same application instance, this layer will short-circuit the addition
   of the duplicate, and instead return and index previously assigned to this entry. Otherwise the requested entry is
   passed on to the second layer.
2. The second layer is a `Persistent` index of a hash of the entry to its assigned position in the log.
   Similarly to the first layer, this second layer will look for a record in its stored data which matches the incoming
   entry, and if such a record exists, it will short-circuit the addition of the duplicate entry and return a previous
   version's assigned position in the log.

These layes are configured by the `WithAntispam` method of the
[AppendOptions](https://pkg.go.dev/github.com/transparency-dev/tessera@main#AppendOptions.WithAntispam) and
[MigrateOptions](https://pkg.go.dev/github.com/transparency-dev/tessera@main#AppendOptions.WithAntispam).

> [!Tip]
> Persistent antispam is fairly expensive in terms of storage-compute, so should only be used where it is actually necessary.

> [!Note]
> Tessera's antispam mechanism is _best effort_; there is no guarantee that all duplicate entries will be suppressed.
> This is a trade-off; fully-atomic "strong" de-duplication is _extremely_ expensive in terms of throughput and compute costs, and
> would limit Tessera to only being able to use transactional type storage backends.

### Witnessing

Logs are required to be append-only data structures.
This property can be verified by witnesses, and signatures from witnesses can be provided in the published checkpoint to increase confidence for users of the log.

Personalities can configure Tessera with options that specify witnesses compatible with the [C2SP Witness Protocol](https://github.com/C2SP/C2SP/blob/main/tlog-witness.md).
Configuring the witnesses is done by creating a top-level [`WitnessGroup`](https://pkg.go.dev/github.com/transparency-dev/tessera@main#WitnessGroup) that contains either sub `WitnessGroup`s or [`Witness`es](https://pkg.go.dev/github.com/transparency-dev/tessera@main#Witness).
Each `Witness` is configured with a URL at which the witness can be requested to make witnessing operations via the C2SP Witness Protocol, and a Verifier for the key that it must sign with.
`WitnessGroup`s are configured with their sub-components, and a number of these components that must be satisfied in order for the group to be satisfied.

These primitives allow arbitrarily complex witness policies to be specified.

Once a top-level `WitnessGroup` is configured, it is passed in to the `Appender` lifecycle options using
[AppendOptions#WithWitnesses](https://pkg.go.dev/github.com/transparency-dev/tessera@main#AppendOptions.WithWitnesses).
If this method is not called then no witnessing will be configured.

> [!Note]
> If the policy cannot be satisfied then no checkpoint will be published.
> It is up to the log operator to ensure that a satisfiable policy is configured, and that the requested publishing rate is acceptable to the configured witnesses.

### Synchronous Publication

Synchronous Publication is provided by [`tessera.PublicationAwaiter`](https://pkg.go.dev/github.com/transparency-dev/tessera#PublicationAwaiter).
This allows applications built with Tessera to block until leaves passed via calls to `Add()` are committed to via a public checkpoint.

> [!Tip]
> This is useful if e.g. your application needs to return an inclusion proof in response to a request to add an entry to the log.

## Lifecycles

### Appender

This is the most common lifecycle mode. Appender allows the application to add leaves, which will be assigned positions in the log
contiguous to any entries the log has already committed to.

This mode is instantiated via [`tessera.NewAppender`](https://pkg.go.dev/github.com/transparency-dev/tessera@main#NewAppender), and
configured using the [`tessera.NewAppendOptions`](https://pkg.go.dev/github.com/transparency-dev/tessera@main#NewAppendOptions) struct.

This is described above in [Constructing the Appender](#constructing-the-appender).

See more details in the [Lifecycle Design: Appender](https://github.com/transparency-dev/tessera/blob/main/docs/design/lifecycle.md#appender).

### Migration Target

This mode is used to migrate a log from one location to another.

This is instantiated via [`tessera.NewMigrationTarget`](https://pkg.go.dev/github.com/transparency-dev/tessera@main#NewMigrationTarget),
and configured using the [`tessera.NewMigratonOptions`](https://pkg.go.dev/github.com/transparency-dev/tessera@main#NewMigrationOptions) struct.

> [!Tip]
> This mode enables the migration of logs between different Tessera storage backends, e.g. you may wish to switch
> serving infrastructure because:
>    * You're migrating between/to/from cloud providers for some reason.
>    * You're "freezing" your log, and want to move it to a cheap read-only location.
>
> You can also use this mode to migrate a [tlog-tiles][] compliant log _into_ Tessera.

Binaries for migrating _into_ each of the storage implementations can be found at [./cmd/experimental/migrate/](./cmd/experimental/migrate/).
These binaries take the URL of a remote tiled log, and copy it into the target location.
These binaries ought to be sufficient for most use-cases.
Users that need to write their own migration binary should use the provided binaries as a reference codelab.

See more details in the [Lifecycle Design: Migration](https://github.com/transparency-dev/tessera/blob/main/docs/design/lifecycle.md#migration).

### Freezing a Log

Freezing a log prevents new writes to the log, but still allows read access.
We recommend that operators allow all pending [sequenced](#sequencing) entries to be [integrated](#integration), and all integrated entries to be [published](#publishing) via a Checkpoint before proceeding.
Once all pending entries are published, the log is now _quiescent_, as described in [Lifecycle Design: Quiescent](https://github.com/transparency-dev/tessera/blob/main/docs/design/lifecycle.md#quiescent).

To ensure all pending entries are published, keep an instance object for the current lifecycle state in a running process, but disable writes to this at the personality level.
For example, a personality that takes HTTP requests from the Internet and calls `Appender.Add` should keep a process running with an `Appender`, but disable any code paths that lead to `Add` being invoked (e.g. by flipping a flag that changes this behaviour).
The instantiated `Appender` allows its background processes to keep running, ensuring all entries are sequenced, integrated, and published.

Determining when this is complete can be done by inspecting the databases or via the OpenTelemetry metrics which instrument this code;
once the next-available sequence number and published checkpoint size have converged and remain stable, the log is in a quiescent state.

A quiescent log using GCP, AWS, or POSIX that is now permanently read-only can be made cheaper to operate. The implementations no longer need any running binaries running Tessera code. Any databases created for this log (i.e. the sequencing tables, or antispam) can be deleted. The read-path can be served directly from the storage buckets (for GCP, AWS) or via a standard HTTP file server (for POSIX).

A log using MySQL must continue to run a personality in order to serve the read path, and thus cannot benefit from the same degree of cost savings when frozen.

### Deleting a Log

Deleting a log is generally performed after [Freezing a Log](#freezing-a-log).

Deleting a GCP, AWS, or POSIX log that has already been frozen just requires deleting the storage bucket or files from disk.

Deleting a MySQL log can be done by turning down the personality binaries, and then deleting the database.

### Sharding a Log

A common way to deploy logs is to run multiple logs in parallel, each of which accepts a distinct subset of entries.
For example, CT shards logs temporally, based on the expiry date of the certificate.

Tessera currently has no special support for sharding logs.
The recommended way to instantiate a new shard of a log is simply to create a new log as described above.
This requires the full stack to be instantiated, including:
 - any DB instances
 - a personality binary for each log

#589 tracks adding more elegant support for sharing resources for sharded logs.
Please upvote that issue if you would like us to prioritize it.

## Contributing

See [CONTRIBUTING.md](/CONTRIBUTING.md) for details.

## License

This repo is licensed under the Apache 2.0 license, see [LICENSE](/LICENSE) for details.

## Contact

- Slack: https://transparency-dev.slack.com/ ([invitation](https://transparency.dev/slack/))
- Mailing list: https://groups.google.com/forum/#!forum/trillian-transparency

## Acknowledgements

Tessera builds upon the hard work, experience, and lessons from many _many_ folks involved in
transparency ecosystems over the years.

[tlog-tiles API]: https://c2sp.org/tlog-tiles
[Static CT API]: https://c2sp.org/static-ct-api
[Trillian v1]: https://github.com/google/trillian
