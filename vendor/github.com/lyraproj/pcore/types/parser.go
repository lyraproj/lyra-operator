package types

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
)

// States:
const (
	exElement     = 0 // Expect value literal
	exParam       = 1 // Expect value literal
	exKey         = 2 // Expect exKey or end of hash
	exValue       = 3 // Expect value
	exEntryValue  = 4 // Expect value
	exPEntryValue = 5
	exRocket      = 6 // Expect rocket
	exListComma   = 7 // Expect comma or end of array
	exParamsComma = 8 // Expect comma or end of parameter list
	exHashComma   = 9 // Expect comma or end of hash
)

var breakCollection = errors.New(`bc`)

var breakScan = errors.New(`b`)

const ParseError = `PARSE_ERROR`

func init() {
	issue.Hard(ParseError, `%{message}`)
}

func expect(state int) (s string) {
	switch state {
	case exElement, exParam:
		s = `literal`
	case exKey:
		s = `entry key`
	case exValue, exEntryValue:
		s = `entry value`
	case exRocket:
		s = `'=>'`
	case exListComma:
		s = `one of ',' or ']'`
	case exParamsComma:
		s = `one of ',' or ')'`
	case exHashComma:
		s = `one of ',' or '}'`
	}
	return
}

func Parse(s string) (px.Value, error) {
	d := NewCollector()

	sr := utils.NewStringReader(s)
	var sf func(t token) error
	var tp px.Value
	state := exElement
	arrayHash := false

	badSyntax := func(t token) error {
		var ts string
		if t.i == 0 {
			ts = `EOF`
		} else {
			ts = t.s
		}
		return fmt.Errorf(`expected %s, got '%s'`, expect(state), ts)
	}

	sf = func(t token) (err error) {
		if tp != nil {
			if t.i == leftBracket || t.i == leftParen || t.i == leftCurlyBrace {
				// Revert state to previous
				if state == exListComma {
					state = exElement
				} else if state == exParamsComma {
					state = exParam
				} else if state == exHashComma {
					state = exValue
				} else if state == exRocket {
					state = exKey
				}
			} else {
				d.Add(tp)
				tp = nil
			}
		}
		switch t.i {
		case end:
			if state != exListComma {
				err = errors.New(`unexpected end of input`)
			}
			if arrayHash {
				he := d.PopLast().(px.MapEntry)
				d.Add(singleMap(he.Key(), he.Value()))
			}
		case rightCurlyBrace:
			if state != exHashComma && state != exKey {
				err = badSyntax(t)
			} else {
				err = breakCollection
			}
		case rightBracket:
			if state != exListComma && state != exElement {
				err = badSyntax(t)
			} else {
				err = breakCollection
			}
		case rightParen:
			if state != exParamsComma && state != exElement {
				err = badSyntax(t)
			} else {
				err = breakCollection
			}
		case rocket:
			if state == exRocket {
				state = exValue
			} else if state == exListComma {
				// Entry
				state = exEntryValue
				arrayHash = true
			} else if state == exParamsComma {
				// Entry
				state = exPEntryValue
				arrayHash = true
			} else {
				err = badSyntax(t)
			}
		case comma:
			if state == exListComma {
				state = exElement
			} else if state == exParamsComma {
				state = exParam
			} else if state == exHashComma {
				state = exKey
			} else {
				err = badSyntax(t)
			}
		default:
			entry := state == exEntryValue || state == exPEntryValue
			switch state {
			case exElement, exEntryValue:
				state = exListComma
			case exParam, exPEntryValue:
				state = exParamsComma
			case exKey:
				state = exRocket
			case exValue:
				state = exHashComma
			default:
				err = badSyntax(t)
			}
			switch t.i {
			case leftCurlyBrace:
				stp := tp
				sv := state
				state = exKey
				tp = nil
				d.AddHash(0, func() { err = scan(sr, sf) })
				if err != breakCollection {
					break
				}
				err = nil
				state = sv
				if stp == nil {
					break
				}
				ps := []px.Value{d.PopLast()}
				if he, ok := stp.(*HashEntry); ok {
					he.Value().(*DeferredType).params = ps
				} else {
					stp.(*DeferredType).params = ps
				}
				d.Add(stp)
			case leftBracket:
				stp := tp
				saveSt := state
				saveAh := arrayHash
				arrayHash = false
				state = exElement
				tp = nil
				d.AddArray(0, func() { err = scan(sr, sf) })
				if arrayHash {
					d.Add(fixArrayHash(d.PopLast().(*Array)))
				}
				arrayHash = saveAh
				if err != breakCollection {
					if err == nil {
						state = exListComma
						return badSyntax(token{i: end})
					}
					break
				}
				err = nil
				state = saveSt
				if stp == nil {
					break
				}
				ll := d.PopLast().(*Array)
				var dp *DeferredType
				if he, ok := stp.(*HashEntry); ok {
					dp = he.Value().(*DeferredType)
				} else {
					dp = stp.(*DeferredType)
				}
				dp.params = ll.AppendTo(make([]px.Value, 0, ll.Len()))
				d.Add(stp)
			case leftParen:
				stp := tp
				sv := state
				saveAh := arrayHash
				arrayHash = false
				state = exParam
				tp = nil
				d.AddArray(0, func() { err = scan(sr, sf) })
				if arrayHash {
					d.Add(fixArrayHash(d.PopLast().(*Array)))
				}
				arrayHash = saveAh
				if err != breakCollection {
					if err == nil {
						state = exParamsComma
						return badSyntax(token{i: end})
					}
					break
				}
				err = nil
				state = sv
				if stp == nil {
					break
				}
				ll := d.PopLast().(*Array)
				if he, ok := stp.(*HashEntry); ok {
					dt := he.Value().(*DeferredType).tn
					if dt != `Deferred` {
						params := append(make([]px.Value, 0, ll.Len()+1), WrapString(dt))
						stp = WrapHashEntry(he.Key(), NewDeferred(`new`, ll.AppendTo(params)...))
					} else {
						params := ll.Slice(1, ll.Len()).AppendTo(make([]px.Value, 0, ll.Len()-1))
						stp = WrapHashEntry(he.Key(), NewDeferred(ll.At(0).String(), params...))
					}
				} else {
					dt := stp.(*DeferredType).tn
					if dt != `Deferred` {
						params := append(make([]px.Value, 0, ll.Len()+1), WrapString(dt))
						stp = NewDeferred(`new`, ll.AppendTo(params)...)
					} else {
						params := ll.Slice(1, ll.Len()).AppendTo(make([]px.Value, 0, ll.Len()-1))
						stp = NewDeferred(ll.At(0).String(), params...)
					}
				}
				d.Add(stp)
			case integer:
				var i int64
				i, err = strconv.ParseInt(t.s, 0, 64)
				if err == nil {
					d.Add(WrapInteger(i))
				}
			case float:
				var f float64
				f, err = strconv.ParseFloat(t.s, 64)
				if err == nil {
					d.Add(WrapFloat(f))
				}
			case identifier:
				switch t.s {
				case `true`:
					d.Add(BooleanTrue)
				case `false`:
					d.Add(BooleanFalse)
				case `default`:
					d.Add(WrapDefault())
				case `undef`:
					d.Add(undef)
				default:
					d.Add(WrapString(t.s))
				}
			case stringLiteral:
				d.Add(WrapString(t.s))
			case regexpLiteral:
				d.Add(WrapRegexp(t.s))
			case name:
				tp = &DeferredType{tn: t.s}
			}
			if err == nil && entry {
				// Concatenate last two values to a HashEntry
				if tp != nil {
					tp = WrapHashEntry(d.PopLast(), tp)
				} else {
					v := d.PopLast()
					d.Add(WrapHashEntry(d.PopLast(), v))
				}
			}
		}
		return err
	}

	// Initial lex receiver deals with alias syntax "type X = <the rest>"
	typeName := ``
	initial := func(t token) (err error) {
		if t.i == identifier && t.s == `type` {
			err = scan(sr, func(t token) (err error) {
				switch t.i {
				case name:
					typeName = t.s
					err = scan(sr, func(t token) (err error) {
						if t.i == equal {
							err = scan(sr, sf)
							if err == nil {
								err = breakScan
							}
						} else {
							err = badSyntax(t)
						}
						return
					})
				case comma:
					state = exElement
					d.Add(WrapString(`type`))
					err = scan(sr, sf)
				case rocket:
					state = exEntryValue
					d.Add(WrapString(`type`))
					err = scan(sr, sf)
				default:
					err = badSyntax(t)
				}
				if err == nil {
					err = breakScan
				}
				return
			})
		} else {
			err = sf(t)
			if err == nil && state != end {
				err = scan(sr, sf)
			}
		}
		if err == nil {
			err = breakScan
		}
		return
	}

	err := scan(sr, initial)
	if err != nil && err != breakScan {
		return nil, px.Error2(issue.NewLocation(``, sr.Line(), sr.Column()), ParseError, issue.H{`message`: err.Error()})
	}
	dv := d.Value()
	if typeName != `` {
		dv = NamedType(px.RuntimeNameAuthority, typeName, dv)
	}
	return dv, nil
}

func fixArrayHash(av *Array) *Array {
	es := make([]px.Value, 0, av.Len())

	// Array may contain hash entries that must be concatenated into a single hash
	var en []*HashEntry
	av.Each(func(v px.Value) {
		if he, ok := v.(*HashEntry); ok {
			if en == nil {
				en = []*HashEntry{he}
			} else {
				en = append(en, he)
			}
		} else {
			if en != nil {
				es = append(es, WrapHash(en))
				en = nil
			}
			es = append(es, v)
		}
	})
	if en != nil {
		es = append(es, WrapHash(en))
	}
	return WrapValues(es)
}
