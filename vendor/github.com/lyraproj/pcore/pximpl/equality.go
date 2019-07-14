package pximpl

import (
	"fmt"
	"regexp"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/semver/semver"
)

// PuppetEquals is like Equals but:
//   int and float values with same value are considered equal
//   string comparisons are case insensitive
//
func init() {
	px.PuppetMatch = match

	px.PuppetEquals = equals
}

func equals(a px.Value, b px.Value) bool {
	switch a := a.(type) {
	case px.StringValue:
		return a.EqualsIgnoreCase(b)
	case px.Integer:
		lhs := a.Int()
		switch b := b.(type) {
		case px.Integer:
			return lhs == b.Int()
		case px.Number:
			return float64(lhs) == b.Float()
		}
		return false
	case px.Float:
		lhs := a.Float()
		if rhv, ok := b.(px.Number); ok {
			return lhs == rhv.Float()
		}
		return false
	case *types.Array:
		if rhs, ok := b.(*types.Array); ok {
			if a.Len() == rhs.Len() {
				idx := 0
				return a.All(func(el px.Value) bool {
					eq := px.PuppetEquals(el, rhs.At(idx))
					idx++
					return eq
				})
			}
		}
		return false
	case *types.Hash:
		if rhs, ok := b.(*types.Hash); ok {
			if a.Len() == rhs.Len() {
				return a.AllPairs(func(key, value px.Value) bool {
					rhsValue, ok := rhs.Get(key)
					return ok && px.PuppetEquals(value, rhsValue)
				})
			}
		}
		return false
	case px.Equality:
		return a.Equals(b, nil)
	default:
		return px.Equals(a, b, nil)
	}
}

// PuppetMatch implements the Puppet =~ semantics
func match(a px.Value, b px.Value) bool {
	result := false
	switch b := b.(type) {
	case px.Type:
		result = px.IsInstance(b, a)

	case px.StringValue, *types.Regexp:
		var rx *regexp.Regexp
		if s, ok := b.(px.StringValue); ok {
			var err error
			rx, err = regexp.Compile(s.String())
			if err != nil {
				panic(px.Error(px.MatchNotRegexp, issue.H{`detail`: err.Error()}))
			}
		} else {
			rx = b.(*types.Regexp).Regexp()
		}

		sv, ok := a.(px.StringValue)
		if !ok {
			panic(px.Error(px.MatchNotString, issue.H{`left`: a.PType()}))
		}
		if group := rx.FindStringSubmatch(sv.String()); group != nil {
			result = true
		}

	case *types.SemVer, *types.SemVerRange:
		var version semver.Version

		if v, ok := a.(*types.SemVer); ok {
			version = v.Version()
		} else if s, ok := a.(px.StringValue); ok {
			var err error
			version, err = semver.ParseVersion(s.String())
			if err != nil {
				panic(px.Error(px.NotSemver, issue.H{`detail`: err.Error()}))
			}
		} else {
			panic(px.Error(px.NotSemver,
				issue.H{`detail`: fmt.Sprintf(`A value of type %s cannot be converted to a SemVer`, a.PType().String())}))
		}
		if lv, ok := b.(*types.SemVer); ok {
			result = lv.Version().Equals(version)
		} else {
			result = b.(*types.SemVerRange).VersionRange().Includes(version)
		}

	default:
		result = px.PuppetEquals(b, a)
	}
	return result
}
