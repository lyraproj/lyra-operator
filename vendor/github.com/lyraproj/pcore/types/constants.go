package types

import (
	"github.com/lyraproj/pcore/px"
)

var emptyArray = WrapValues([]px.Value{})
var emptyMap = WrapHash([]*HashEntry{})
var emptyString = stringValue(``)
var undef = WrapUndef()

func init() {
	px.EmptyArray = emptyArray
	px.EmptyMap = emptyMap
	px.EmptyString = emptyString
	px.Undef = undef
}
