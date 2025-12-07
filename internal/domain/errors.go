package domain

import "errors"

// Sentinel ошибки доменного слоя
var (
	ErrCommentNotFound = errors.New("comment not found")
	ErrInvalidParent   = errors.New("invalid parent comment")
	ErrEmptyContent    = errors.New("comment content cannot be empty")
)
