gomdb
=====

Go wrapper for OpenLDAP Lightning Memory-Mapped Database (LMDB).
Read more about LMDB here: http://symas.com/mdb/

GoDoc available here: http://godoc.org/github.com/szferi/gomdb

Build
=======

`git clone -b mdb.master --single-branch git://git.openldap.org/openldap.git`

`make`

`make install`

It will install to /usr/local

`export LD_LIBRARY_PATH=/usr/local/lib`

`go test -v`

On FreeBSD 10, you must explicitly set `CC` (otherwise it will fail with a cryptic error):

`CC=clang go test -v`


TODO
======

 * write more documentation
 * write more unit test
 * benchmark
 * figure out how can you write go binding for `MDB_comp_func` and `MDB_rel_func`
 * Handle go `*Cursor` close with `txn.Commit` and `txn.Abort` transparently

