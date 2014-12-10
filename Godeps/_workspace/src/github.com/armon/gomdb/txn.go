package mdb

/*
#cgo freebsd CFLAGS: -DMDB_DSYNC=O_SYNC
#cgo openbsd CFLAGS: -DMDB_DSYNC=O_SYNC
#cgo netbsd CFLAGS: -DMDB_DSYNC=O_SYNC
#include <stdlib.h>
#include <stdio.h>
#include "lmdb.h"
*/
import "C"

import (
	"math"
	"runtime"
	"unsafe"
)

// DBIOpen Database Flags
const (
	REVERSEKEY = C.MDB_REVERSEKEY // use reverse string keys
	DUPSORT    = C.MDB_DUPSORT    // use sorted duplicates
	INTEGERKEY = C.MDB_INTEGERKEY // numeric keys in native byte order. The keys must all be of the same size.
	DUPFIXED   = C.MDB_DUPFIXED   // with DUPSORT, sorted dup items have fixed size
	INTEGERDUP = C.MDB_INTEGERDUP // with DUPSORT, dups are numeric in native byte order
	REVERSEDUP = C.MDB_REVERSEDUP // with DUPSORT, use reverse string dups
	CREATE     = C.MDB_CREATE     // create DB if not already existing
)

// put flags
const (
	NODUPDATA   = C.MDB_NODUPDATA
	NOOVERWRITE = C.MDB_NOOVERWRITE
	RESERVE     = C.MDB_RESERVE
	APPEND      = C.MDB_APPEND
	APPENDDUP   = C.MDB_APPENDDUP
)

// Txn is Opaque structure for a transaction handle.
// All database operations require a transaction handle.
// Transactions may be read-only or read-write.
type Txn struct {
	_txn *C.MDB_txn
}

func (env *Env) BeginTxn(parent *Txn, flags uint) (*Txn, error) {
	var _txn *C.MDB_txn
	var ptxn *C.MDB_txn
	if parent == nil {
		ptxn = nil
	} else {
		ptxn = parent._txn
	}
	if flags&RDONLY == 0 {
		runtime.LockOSThread()
	}
	ret := C.mdb_txn_begin(env._env, ptxn, C.uint(flags), &_txn)
	if ret != SUCCESS {
		runtime.UnlockOSThread()
		return nil, Errno(ret)
	}
	return &Txn{_txn}, nil
}

func (txn *Txn) Commit() error {
	ret := C.mdb_txn_commit(txn._txn)
	runtime.UnlockOSThread()
	if ret != SUCCESS {
		return Errno(ret)
	}
	txn._txn = nil
	return nil
}

func (txn *Txn) Abort() {
	if txn._txn == nil {
		return
	}
	C.mdb_txn_abort(txn._txn)
	runtime.UnlockOSThread()
	txn._txn = nil
}

func (txn *Txn) Reset() {
	C.mdb_txn_reset(txn._txn)
}

func (txn *Txn) Renew() error {
	ret := C.mdb_txn_renew(txn._txn)
	if ret != SUCCESS {
		return Errno(ret)
	}
	return nil
}

func (txn *Txn) DBIOpen(name string, flags uint) (DBI, error) {
	var _dbi C.MDB_dbi
	var cname *C.char
	if name == "" {
		cname = nil
	} else {
		cname = C.CString(name)
		defer C.free(unsafe.Pointer(cname))
	}
	ret := C.mdb_dbi_open(txn._txn, cname, C.uint(flags), &_dbi)
	if ret != SUCCESS {
		return DBI(math.NaN()), Errno(ret)
	}
	return DBI(_dbi), nil
}

func (txn *Txn) Stat(dbi DBI) (*Stat, error) {
	var _stat C.MDB_stat
	ret := C.mdb_stat(txn._txn, C.MDB_dbi(dbi), &_stat)
	if ret != SUCCESS {
		return nil, Errno(ret)
	}
	stat := Stat{PSize: uint(_stat.ms_psize),
		Depth:         uint(_stat.ms_depth),
		BranchPages:   uint64(_stat.ms_branch_pages),
		LeafPages:     uint64(_stat.ms_leaf_pages),
		OwerflowPages: uint64(_stat.ms_overflow_pages),
		Entries:       uint64(_stat.ms_entries)}
	return &stat, nil
}

func (txn *Txn) Drop(dbi DBI, del int) error {
	ret := C.mdb_drop(txn._txn, C.MDB_dbi(dbi), C.int(del))
	if ret != SUCCESS {
		return Errno(ret)
	}
	return nil
}

func (txn *Txn) Get(dbi DBI, key []byte) ([]byte, error) {
	ckey := &C.MDB_val{mv_size: C.size_t(len(key)),
		mv_data: unsafe.Pointer(&key[0])}
	var cval C.MDB_val
	ret := C.mdb_get(txn._txn, C.MDB_dbi(dbi), ckey, &cval)
	if ret != SUCCESS {
		return nil, Errno(ret)
	}
	val := C.GoBytes(cval.mv_data, C.int(cval.mv_size))
	return val, nil
}

func (txn *Txn) Put(dbi DBI, key []byte, val []byte, flags uint) error {
	ckey := &C.MDB_val{mv_size: C.size_t(len(key)),
		mv_data: unsafe.Pointer(&key[0])}
	cval := &C.MDB_val{mv_size: C.size_t(len(val)),
		mv_data: unsafe.Pointer(&val[0])}
	ret := C.mdb_put(txn._txn, C.MDB_dbi(dbi), ckey, cval, C.uint(flags))
	if ret != SUCCESS {
		return Errno(ret)
	}
	return nil
}

func (txn *Txn) Del(dbi DBI, key, val []byte) error {
	ckey := &C.MDB_val{mv_size: C.size_t(len(key)),
		mv_data: unsafe.Pointer(&key[0])}
	var cval *C.MDB_val
	if val == nil {
		cval = nil
	} else {
		cval = &C.MDB_val{mv_size: C.size_t(len(val)),
			mv_data: unsafe.Pointer(&val[0])}
	}
	ret := C.mdb_del(txn._txn, C.MDB_dbi(dbi), ckey, cval)
	if ret != SUCCESS {
		return Errno(ret)
	}
	return nil
}

type Cursor struct {
	_cursor *C.MDB_cursor
}

func (txn *Txn) CursorOpen(dbi DBI) (*Cursor, error) {
	var _cursor *C.MDB_cursor
	ret := C.mdb_cursor_open(txn._txn, C.MDB_dbi(dbi), &_cursor)
	if ret != SUCCESS {
		return nil, Errno(ret)
	}
	return &Cursor{_cursor}, nil
}

func (txn *Txn) CursorRenew(cursor *Cursor) error {
	ret := C.mdb_cursor_renew(txn._txn, cursor._cursor)
	if ret != SUCCESS {
		return Errno(ret)
	}
	return nil
}
