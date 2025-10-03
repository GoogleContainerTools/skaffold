# Tessera on Google Cloud Platform

This document describes the storage implementation for running Tessera on Google Cloud Platform (GCP).

## Overview

This design takes advantage of GCS for long term storage and low cost & complexity serving of read traffic, 
but leverages something more transactional for coordinating the cluster.

New entries flow in from the binary built with Tessera into transactional storage, where they're held
temporarily to batch them up, and then assigned sequence numbers as each batch is flushed.
This allows the `Add` API call to quickly return with *durably assigned* sequence numbers.

From there, an async process derives the entry bundles and Merkle tree structure from the sequenced batches,
writes these to GCS for serving, before finally removing integrated bundles from the transactional storage.

Since entries are all sequenced by the time they're stored, and sequencing is done in "chunks", it's worth
noting that all tree derivations are therefore idempotent.

## Transactional storage

The transactional storage is implemented with Cloud Spanner, and uses a schema with the following tables:

### `Tessera`
This table is used to identify the current schema version.

### `SeqCoord`
A table with a single row which is used to keep track of the next assignable sequence number.

### `Seq`
This holds batches of entries keyed by the sequence number assigned to the first entry in the batch.

### `IntCoord`
This table is used to coordinate integration of sequenced batches in the `Seq` table.

### `PubCoord`
This table is used to coordinate publication of new checkpoints, ensuring that checkpoints are
not published more frequently than configured.

### `GCCoord`
This table is used to coordinate garbage collection of partial tiles and entry bundles which have been
made obsolete by the continued growth of the log.

## Life of a leaf

1. Leaves are submitted by the binary built using Tessera via a call the storage's `Add` func.
1. The storage library batches these entries up, and, after a configurable period of time has elapsed
   or the batch reaches a configurable size threshold, the batch is written to the `Seq` table which effectively
   assigns a sequence numbers to the entries using the following algorithm:
   In a transaction:
   1. selects next from `SeqCoord` with for update ← this blocks other FE from writing their pools, but only for a short duration.
   1. Inserts batch of entries into `Seq` with key `SeqCoord.next`
   1. Update `SeqCoord` with `next+=len(batch)`
1. Newly sequenced entries are periodically appended to the tree:
   In a transaction:
   1. select `seq` from `IntCoord` with for update ← this blocks other integrators from proceeding.
   1. Select one or more consecutive batches from `Seq` for update, starting at `IntCoord.seq`
   1. Write leaf bundles to GCS using batched entries
   1. Integrate in Merkle tree and write tiles to GCS
   1. Delete consumed batches from `Seq`
   1. Update `IntCoord` with `seq+=num_entries_integrated` and the latest `rootHash`
1. Checkpoints representing the latest state of the tree are published at the configured interval.

## Antispam

An experimental implementation has been tested which uses Spanner to store the `<identity_hash>` --> `sequence`
mapping. This works well using "slack" Spanner CPU available in the smallest possible footprint, and consequently
is comparably cheap requiring only extra Spanner storage costs.

### Alternatives Considered

Other transactional storage systems are available on GCP, e.g. CloudSQL or AlloyDB.
Experiments were run using CloudSQL (MySQL), AlloyDB, and Spanner.

Spanner worked out to be the cheapest while also removing much of the administrative overhead which 
would come from even a managed MySQL instance, and so was selected.

The experimental implementation was tested to around 1B entries of 1KB each at a write rate of 1500/s.
This was done using the smallest possible Spanner alloc of 100 Processing Units.
