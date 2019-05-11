package types

import (
	"github.com/lyraproj/pcore/px"
)

func toFloat(v px.Value) (float64, bool) {
	if iv, ok := v.(floatValue); ok {
		return float64(iv), true
	}
	return 0.0, false
}

func toInt(v px.Value) (int64, bool) {
	if iv, ok := v.(integerValue); ok {
		return int64(iv), true
	}
	return 0, false
}

func init() {
	px.ToInt = toInt
	px.ToFloat = toFloat
}
