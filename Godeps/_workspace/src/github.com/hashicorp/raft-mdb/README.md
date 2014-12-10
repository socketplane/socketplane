raft-mdb  [![Build Status](https://travis-ci.org/hashicorp/raft-mdb.png)](https://travis-ci.org/hashicorp/raft-mdb)
========

This repository provides the `raftmdb` package. The package exports the
`MDBStore` which is an implementation of both a LogStore and StableStore.

It is meant to be used as a backend for the `raft` [package here](github.com/hashicorp/raft).

This implementation uses [LMDB](http://symas.com/mdb/). LMDB has a number
of advantages to other embedded databases includes transactions, MVCC,
and lack of compaction.

The one disadvantage is because it is a C library, it requires the use
of cgo which complicates cross compilation. For that reason, this is
in a seperate package from `raft`, so that clients can avoid cgo if
they so choose.

Documentation
==============

The documentation for this package can be found on [Godoc](http://godoc.org/github.com/hashicorp/raft-mdb) here.

