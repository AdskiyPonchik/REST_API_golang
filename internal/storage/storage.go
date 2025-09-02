package storage

import "errors"

var (
	ErrUrlNotFound = errors.New("url not found")
	ErrURLExists   = errors.New("url exists")
)
