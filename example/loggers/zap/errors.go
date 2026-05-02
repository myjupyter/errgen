package main

import "errors"

//go:generate go run github.com/myjupyter/errgen -zap -no-hooks

// @StringField string
// @Int64Field int64
// @IntField int
// @Uint64Field uint64
// @UintField uint
// @UintptrField uintptr
// @Float64Field float64
// @BoolField bool
// @TimeField time.Time
// @DurationField time.Duration
// @IntSliceField []int
// @ObjectSliceField []Object
// @MapType map[string]string
var ErrInternal = errors.New("internal error")

// @EntityType string
// @ID int
// @Code(404)
// @Error("'%EntityType' with ID %ID not found")
var ErrEntityNotFound = errors.New("entity not found")

type Object struct {
}
