package case2

import "errors"

//go:generate go run github.com/myjupyter/errgen

var (
	// @Item custom1.Item
	// @Mapping map[string]custom2.Details
	// @Tags []custom3.Tag
	// @Error("processing: item=%Item, mapping=%Mapping, tags=%Tags")
	ErrProcessing = errors.New("processing error")

	// @Ptr *custom1.Item
	// @DetailSlice []custom2.Details
	// @TagMap map[string]custom3.Tag
	ErrMultiImport = errors.New("multi import error")
)
