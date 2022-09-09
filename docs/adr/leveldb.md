# Using LevelDB as Regent's Database

## Authors

Preston Evans - preston@sovlabs.io

## Status

Proposed

## Context

Like all blockchain clients, Regent will require a database to persist information across shutdowns. We need to decide on an underlying DB implementation. The data to be stored includes:

- A mapping from DA block number to DA block hash
- A mapping from DA block hash to DA block number
- A mapping from DA block hash to DA block data
- A mapping from DA block hash to the Intermediate merkle hashes needed to validate the rollup data using the namespaced merkle tree
- A mapping from rollup block number to rollup block hash
- A mapping from rollup block hash to rollup block data
- A mapping from rollup block number to rollup block hash
- A mapping from rollup block number to DA block number
- The most recent observed validity proof
  etc.

## Decision

We propose to use LevelDB as the database implementation for Regent. As a log-structured database, LevelDB provides excellent write throughput and high reliability.

The primary drawbacks of LevelDB are its lack of MVCC and its lack of support for transactions. Since our rollup cannot "re-org", MVCC is not needed. In addition, we don't anticipate the need for any queries more complex than a simple range over block data, so the lack of transactions and complex query support is not an issue.

## Consequences

LevelDB will allow us to persist data with high performance and minimal maintenance burden.
