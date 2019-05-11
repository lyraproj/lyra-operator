package api

import "github.com/lyraproj/pcore/px"

type Resource interface {
	Step

	Type() px.ObjectType

	HandlerId() px.TypedName

	ExtId() px.Value
}
