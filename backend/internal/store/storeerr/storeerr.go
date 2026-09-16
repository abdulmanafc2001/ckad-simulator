// Package storeerr holds the sentinel errors every Repository
// implementation returns, so callers can match on them with errors.Is
// without caring which backend is in use.
//
// They live in their own package because the implementations (memory,
// sqlite) are imported by the store package's tests, so they cannot import
// the store package back.
package storeerr

import "errors"

var (
	// ErrNotFound is returned when a requested entity does not exist.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists is returned when creating an entity with a duplicate ID.
	ErrAlreadyExists = errors.New("already exists")
)
