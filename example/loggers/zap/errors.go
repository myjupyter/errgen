package main

import "errors"

//go:generate ../../../bin/errgen -zap -no-hooks

// @StringField string
// @Int64Field int64
// @IntField int
// @Uint64Field uint64
// @Float64Field float64
// @BoolField bool
// @TimeField time.Time
// @DurationField time.Duration
// @IntSliceField []int
// @ObjectSliceField []Object
// @MapType map[string]string
var ErrInternal = errors.New("internal error")

type Object struct {
}
