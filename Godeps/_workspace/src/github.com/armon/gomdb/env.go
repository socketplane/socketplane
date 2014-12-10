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
	"errors"
	"unsafe"
)

const (
	SUCCESS = 0
)

// mdb_env Environment Flags
const (
	FIXEDMAP   = C.MDB_FIXEDMAP   // mmap at a fixed address (experimental)
	NOSUBDIR   = C.MDB_NOSUBDIR   // no environment directory
	NOSYNC     = C.MDB_NOSYNC     // don't fsync after commit
	RDONLY     = C.MDB_RDONLY     // read only
	NOMETASYNC = C.MDB_NOMETASYNC // don't fsync metapage after commit
	WRITEMAP   = C.MDB_WRITEMAP   // use writable mmap
	MAPASYNC   = C.MDB_MAPASYNC   // use asynchronous msync when MDB_WRITEMAP is use
	NOTLS      = C.MDB_NOTLS      // tie reader locktable slots to Txn objects instead of threads
)

type DBI uint
type Errno int

func (e Errno) Error() string {
	return C.GoString(C.mdb_strerror(C.int(e)))
}

// error codes
var (
	KeyExist        error = Errno(-30799)
	NotFound        error = Errno(-30798)
	PageNotFound    error = Errno(-30797)
	Corrupted       error = Errno(-30796)
	Panic           error = Errno(-30795)
	VersionMismatch error = Errno(-30794)
	Invalid         error = Errno(-30793)
	MapFull         error = Errno(-30792)
	DbsFull         error = Errno(-30791)
	ReadersFull     error = Errno(-30790)
	TlsFull         error = Errno(-30789)
	TxnFull         error = Errno(-30788)
	CursorFull      error = Errno(-30787)
	PageFull        error = Errno(-30786)
	MapResized      error = Errno(-30785)
	Incompatibile   error = Errno(-30784)
)

func Version() string {
	var major, minor, patch *C.int
	ver_str := C.mdb_version(major, minor, patch)
	return C.GoString(ver_str)
}

// Env is opaque structure for a database environment.
// A DB environment supports multiple databases, all residing in the
// same shared-memory map.
type Env struct {
	_env *C.MDB_env
}

// Create an MDB environment handle.
func NewEnv() (*Env, error) {
	var _env *C.MDB_env
	ret := C.mdb_env_create(&_env)
	if ret != SUCCESS {
		return nil, Errno(ret)
	}
	return &Env{_env}, nil
}

// Open an environment handle. If this function fails Close() must be called to discard the Env handle.
func (env *Env) Open(path string, flags uint, mode uint) error {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	ret := C.mdb_env_open(env._env, cpath, C.uint(NOTLS|flags), C.mdb_mode_t(mode))
	if ret != SUCCESS {
		return Errno(ret)
	}
	return nil
}

func (env *Env) Close() error {
	if env._env == nil {
		return errors.New("Environment already closed")
	}
	C.mdb_env_close(env._env)
	env._env = nil
	return nil
}

func (env *Env) Copy(path string) error {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	ret := C.mdb_env_copy(env._env, cpath)
	if ret != SUCCESS {
		return Errno(ret)
	}
	return nil
}

// Statistics for a database in the environment
type Stat struct {
	PSize         uint   // Size of a database page. This is currently the same for all databases.
	Depth         uint   // Depth (height) of the B-tree
	BranchPages   uint64 // Number of internal (non-leaf) pages
	LeafPages     uint64 // Number of leaf pages
	OwerflowPages uint64 // Number of overflow pages
	Entries       uint64 // Number of data items
}

func (env *Env) Stat() (*Stat, error) {
	var _stat C.MDB_stat
	ret := C.mdb_env_stat(env._env, &_stat)
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

type Info struct {
	MapSize    uint64 // Size of the data memory map
	LastPNO    uint64 // ID of the last used page
	LastTxnID  uint64 // ID of the last committed transaction
	MaxReaders uint   // maximum number of threads for the environment
	NumReaders uint   // maximum number of threads used in the environment
}

func (env *Env) Info() (*Info, error) {
	var _info C.MDB_envinfo
	ret := C.mdb_env_info(env._env, &_info)
	if ret != SUCCESS {
		return nil, Errno(ret)
	}
	info := Info{MapSize: uint64(_info.me_mapsize),
		LastPNO:    uint64(_info.me_last_pgno),
		LastTxnID:  uint64(_info.me_last_txnid),
		MaxReaders: uint(_info.me_maxreaders),
		NumReaders: uint(_info.me_numreaders)}
	return &info, nil
}

func (env *Env) Sync(force int) error {
	ret := C.mdb_env_sync(env._env, C.int(force))
	if ret != SUCCESS {
		return Errno(ret)
	}
	return nil
}

func (env *Env) SetFlags(flags uint, onoff int) error {
	ret := C.mdb_env_set_flags(env._env, C.uint(flags), C.int(onoff))
	if ret != SUCCESS {
		return Errno(ret)
	}
	return nil
}

func (env *Env) Flags() (uint, error) {
	var _flags C.uint
	ret := C.mdb_env_get_flags(env._env, &_flags)
	if ret != SUCCESS {
		return 0, Errno(ret)
	}
	return uint(_flags), nil
}

func (env *Env) Path() (string, error) {
	var path string
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	ret := C.mdb_env_get_path(env._env, &cpath)
	if ret != SUCCESS {
		return "", Errno(ret)
	}
	return C.GoString(cpath), nil
}

func (env *Env) SetMapSize(size uint64) error {
	ret := C.mdb_env_set_mapsize(env._env, C.size_t(size))
	if ret != SUCCESS {
		return Errno(ret)
	}
	return nil
}

func (env *Env) SetMaxReaders(size uint) error {
	ret := C.mdb_env_set_maxreaders(env._env, C.uint(size))
	if ret != SUCCESS {
		return Errno(ret)
	}
	return nil
}

func (env *Env) SetMaxDBs(size DBI) error {
	ret := C.mdb_env_set_maxdbs(env._env, C.MDB_dbi(size))
	if ret != SUCCESS {
		return Errno(ret)
	}
	return nil
}

func (env *Env) DBIClose(dbi DBI) {
	C.mdb_dbi_close(env._env, C.MDB_dbi(dbi))
}
