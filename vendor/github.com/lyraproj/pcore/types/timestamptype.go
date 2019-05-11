package types

import (
	"bytes"
	"io"
	"math"
	"time"

	"reflect"
	"sync"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
)

type (
	TimestampType struct {
		min time.Time
		max time.Time
	}

	// Timestamp represents TimestampType as a value
	Timestamp time.Time
)

// MAX_UNIX_SECS is an offset of 62135596800 seconds to sec that
// represents the number of seconds from 1970-01-01:00:00:00 UTC. This offset
// must be retracted from the MaxInt64 value in order for it to end up
// as that value internally.
const MaxUnixSecs = math.MaxInt64 - 62135596800

var MinTime = time.Time{}
var MaxTime = time.Unix(MaxUnixSecs, 999999999)
var timestampTypeDefault = &TimestampType{MinTime, MaxTime}

var TimestampMetaType px.ObjectType

var DefaultTimestampFormatsWoTz []*TimestampFormat
var DefaultTimestampFormats []*TimestampFormat
var DefaultTimestampFormatParser *TimestampFormatParser

func init() {
	TimestampMetaType = newObjectType(`Pcore::TimestampType`,
		`Pcore::ScalarType {
	attributes => {
		from => { type => Optional[Timestamp], value => undef },
		to => { type => Optional[Timestamp], value => undef }
	}
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newTimestampType2(args...)
		})

	tp := NewTimestampFormatParser()
	DefaultTimestampFormatParser = tp

	DefaultTimestampFormatsWoTz = []*TimestampFormat{
		tp.ParseFormat(`%FT%T.%N`),
		tp.ParseFormat(`%FT%T`),
		tp.ParseFormat(`%F %T.%N`),
		tp.ParseFormat(`%F %T`),
		tp.ParseFormat(`%F`),
	}

	DefaultTimestampFormats = []*TimestampFormat{
		tp.ParseFormat(`%FT%T.%N %Z`),
		tp.ParseFormat(`%FT%T %Z`),
		tp.ParseFormat(`%F %T.%N %Z`),
		&TimestampFormat{layout: time.RFC3339},
		tp.ParseFormat(`%F %T %Z`),
		tp.ParseFormat(`%F %Z`),
	}
	DefaultTimestampFormats = append(DefaultTimestampFormats, DefaultTimestampFormatsWoTz...)

	newGoConstructor2(`Timestamp`,

		func(t px.LocalTypes) {
			t.Type(`Formats`, `Variant[String[2],Array[String[2], 1]]`)
		},

		func(d px.Dispatch) {
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return WrapTimestamp(time.Now())
			})
		},

		func(d px.Dispatch) {
			d.Param(`Variant[Integer,Float]`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				arg := args[0]
				if i, ok := arg.(integerValue); ok {
					return WrapTimestamp(time.Unix(int64(i), 0))
				}
				s, f := math.Modf(float64(arg.(floatValue)))
				return WrapTimestamp(time.Unix(int64(s), int64(f*1000000000.0)))
			})
		},

		func(d px.Dispatch) {
			d.Param(`String[1]`)
			d.OptionalParam(`Formats`)
			d.OptionalParam(`String[1]`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				formats := DefaultTimestampFormats
				tz := ``
				if len(args) > 1 {
					formats = toTimestampFormats(args[1])
					if len(args) > 2 {
						tz = args[2].String()
					}
				}
				return ParseTimestamp(args[0].String(), formats, tz)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Struct[string => String[1],Optional[format] => Formats,Optional[timezone] => String[1]]`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				hash := args[0].(*Hash)
				str := hash.Get5(`string`, emptyString).String()
				formats := toTimestampFormats(hash.Get5(`format`, px.Undef))
				tz := hash.Get5(`timezone`, emptyString).String()
				return ParseTimestamp(str, formats, tz)
			})
		})
}

func DefaultTimestampType() *TimestampType {
	return timestampTypeDefault
}

func NewTimestampType(min time.Time, max time.Time) *TimestampType {
	return &TimestampType{min, max}
}

func TimeFromHash(hash *Hash) (time.Time, bool) {
	str := hash.Get5(`string`, emptyString).String()
	formats := toTimestampFormats(hash.Get5(`format`, undef))
	tz := hash.Get5(`timezone`, emptyString).String()
	return parseTime(str, formats, tz)
}

func TimeFromString(value string) time.Time {
	return time.Time(*ParseTimestamp(value, DefaultTimestampFormats, ``))
}

func newTimestampType2(args ...px.Value) *TimestampType {
	argc := len(args)
	if argc > 2 {
		panic(illegalArgumentCount(`Timestamp[]`, `0 or 2`, argc))
	}
	if argc == 0 {
		return timestampTypeDefault
	}
	convertArg := func(args []px.Value, argNo int) time.Time {
		arg := args[argNo]
		var (
			t  time.Time
			ok bool
		)
		switch arg := arg.(type) {
		case *Timestamp:
			t, ok = time.Time(*arg), true
		case *Hash:
			t, ok = TimeFromHash(arg)
		case stringValue:
			t, ok = TimeFromString(arg.String()), true
		case integerValue:
			t, ok = time.Unix(int64(arg), 0), true
		case floatValue:
			s, f := math.Modf(float64(arg))
			t, ok = time.Unix(int64(s), int64(f*1000000000.0)), true
		case *DefaultValue:
			if argNo == 0 {
				t, ok = time.Time{}, true
			} else {
				t, ok = time.Unix(MaxUnixSecs, 999999999), true
			}
		default:
			t, ok = time.Time{}, false
		}
		if ok {
			return t
		}
		panic(illegalArgumentType(`Timestamp[]`, 0, `Variant[Hash,String,Integer,Float,Default]`, args[0]))
	}

	min := convertArg(args, 0)
	if argc == 2 {
		return &TimestampType{min, convertArg(args, 1)}
	} else {
		return &TimestampType{min, MaxTime}
	}
}

func (t *TimestampType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *TimestampType) Default() px.Type {
	return timestampTypeDefault
}

func (t *TimestampType) Equals(other interface{}, guard px.Guard) bool {
	if ot, ok := other.(*TimestampType); ok {
		return t.min.Equal(ot.min) && t.max.Equal(ot.max)
	}
	return false
}

func (t *TimestampType) Get(key string) (px.Value, bool) {
	switch key {
	case `from`:
		v := px.Undef
		if t.min != MinTime {
			v = WrapTimestamp(t.min)
		}
		return v, true
	case `to`:
		v := px.Undef
		if t.max != MaxTime {
			v = WrapTimestamp(t.max)
		}
		return v, true
	default:
		return nil, false
	}
}

func (t *TimestampType) IsInstance(o px.Value, g px.Guard) bool {
	return t.IsAssignable(o.PType(), g)
}

func (t *TimestampType) IsAssignable(o px.Type, g px.Guard) bool {
	if ot, ok := o.(*TimestampType); ok {
		return (t.min.Before(ot.min) || t.min.Equal(ot.min)) && (t.max.After(ot.max) || t.max.Equal(ot.max))
	}
	return false
}

func (t *TimestampType) MetaType() px.ObjectType {
	return TimestampMetaType
}

func (t *TimestampType) Parameters() []px.Value {
	if t.max.Equal(MaxTime) {
		if t.min.Equal(MinTime) {
			return px.EmptyValues
		}
		return []px.Value{stringValue(t.min.String())}
	}
	if t.min.Equal(MinTime) {
		return []px.Value{WrapDefault(), stringValue(t.max.String())}
	}
	return []px.Value{stringValue(t.min.String()), stringValue(t.max.String())}
}

func (t *TimestampType) ReflectType(c px.Context) (reflect.Type, bool) {
	return reflect.TypeOf(time.Time{}), true
}

func (t *TimestampType) CanSerializeAsString() bool {
	return true
}

func (t *TimestampType) SerializationString() string {
	return t.String()
}

func (t *TimestampType) String() string {
	return px.ToString2(t, None)
}

func (t *TimestampType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *TimestampType) PType() px.Type {
	return &TypeType{t}
}

func (t *TimestampType) Name() string {
	return `Timestamp`
}

func WrapTimestamp(time time.Time) *Timestamp {
	return (*Timestamp)(&time)
}

func ParseTimestamp(str string, formats []*TimestampFormat, tz string) *Timestamp {
	if t, ok := parseTime(str, formats, tz); ok {
		return WrapTimestamp(t)
	}
	fs := bytes.NewBufferString(``)
	for i, f := range formats {
		if i > 0 {
			fs.WriteByte(',')
		}
		fs.WriteString(f.format)
	}
	panic(px.Error(px.TimestampCannotBeParsed, issue.H{`str`: str, `formats`: fs.String()}))
}

func parseTime(str string, formats []*TimestampFormat, tz string) (time.Time, bool) {
	usedTz := tz
	if usedTz == `` {
		usedTz = `UTC`
	}
	loc := loadLocation(usedTz)

	for _, f := range formats {
		ts, err := time.ParseInLocation(f.layout, str, loc)
		if err == nil {
			if usedTz != ts.Location().String() {
				if tz != `` {
					panic(px.Error(px.TimestampTzAmbiguity, issue.H{`parsed`: ts.Location().String(), `given`: tz}))
				}
				// Golang does real weird things when the string contains a timezone that isn't equal
				// to the given timezone. Instead of loading the given zone, it creates a new location
				// with a similarly named zone but with offset 0. It doesn't matter if Parse or ParseInLocation
				// is used. Both has the same weird behavior. For this reason, a new loadLocation is performed
				// here followed by a reparse.
				loc, _ = time.LoadLocation(ts.Location().String())
				ts, err = time.ParseInLocation(f.layout, str, loc)
				if err != nil {
					continue
				}
			}
			return ts.UTC(), true
		}
	}
	return time.Time{}, false
}

func loadLocation(tz string) *time.Location {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		panic(px.Error(px.InvalidTimezone, issue.H{`zone`: tz, `detail`: err.Error()}))
	}
	return loc
}

func (tv *Timestamp) Equals(o interface{}, g px.Guard) bool {
	if ov, ok := o.(*Timestamp); ok {
		return tv.Int() == ov.Int()
	}
	return false
}

func (tv *Timestamp) Float() float64 {
	t := (*time.Time)(tv)
	y := t.Year()
	// Timestamps that represent a date before the year 1678 or after 2262 can
	// be represented as nanoseconds in an int64.
	if 1678 < y && y < 2262 {
		return float64(float64(t.UnixNano()) / 1000000000.0)
	}
	// Fall back to microsecond precision
	us := t.Unix()*1000000 + int64(t.Nanosecond())/1000
	return float64(us) / 1000000.0
}

func (tv *Timestamp) Format(format string) string {
	return DefaultTimestampFormatParser.ParseFormat(format).Format(tv)
}

func (tv *Timestamp) Format2(format, tz string) string {
	return DefaultTimestampFormatParser.ParseFormat(format).Format2(tv, tz)
}

func (tv *Timestamp) Reflect(c px.Context) reflect.Value {
	return reflect.ValueOf((time.Time)(*tv))
}

func (tv *Timestamp) ReflectTo(c px.Context, dest reflect.Value) {
	rv := tv.Reflect(c)
	if !rv.Type().AssignableTo(dest.Type()) {
		panic(px.Error(px.AttemptToSetWrongKind, issue.H{`expected`: rv.Type().String(), `actual`: dest.Type().String()}))
	}
	dest.Set(rv)
}

func (tv *Timestamp) Time() time.Time {
	return time.Time(*tv)
}

func (tv *Timestamp) Int() int64 {
	return (*time.Time)(tv).Unix()
}

func (tv *Timestamp) CanSerializeAsString() bool {
	return true
}

func (tv *Timestamp) SerializationString() string {
	return tv.String()
}

func (tv *Timestamp) String() string {
	return px.ToString2(tv, None)
}

func (tv *Timestamp) ToKey(b *bytes.Buffer) {
	t := (*time.Time)(tv)
	b.WriteByte(1)
	b.WriteByte(HkTimestamp)
	n := t.Unix()
	b.WriteByte(byte(n >> 56))
	b.WriteByte(byte(n >> 48))
	b.WriteByte(byte(n >> 40))
	b.WriteByte(byte(n >> 32))
	b.WriteByte(byte(n >> 24))
	b.WriteByte(byte(n >> 16))
	b.WriteByte(byte(n >> 8))
	b.WriteByte(byte(n))
	n = int64(t.Nanosecond())
	b.WriteByte(byte(n >> 56))
	b.WriteByte(byte(n >> 48))
	b.WriteByte(byte(n >> 40))
	b.WriteByte(byte(n >> 32))
	b.WriteByte(byte(n >> 24))
	b.WriteByte(byte(n >> 16))
	b.WriteByte(byte(n >> 8))
	b.WriteByte(byte(n))
}

func (tv *Timestamp) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	_, err := io.WriteString(b, (*time.Time)(tv).Format(DefaultTimestampFormats[0].layout))
	if err != nil {
		panic(err)
	}
}

func (tv *Timestamp) PType() px.Type {
	t := time.Time(*tv)
	return &TimestampType{t, t}
}

const (
	// Strings recognized as golang "layout" elements

	loLongMonth   = `January`
	loMonth       = `Jan`
	loLongWeekDay = `Monday`
	loWeekDay     = `Mon`
	loTZ          = `MST`
	loZeroMonth   = `01`
	loZeroDay     = `02`
	loZeroHour12  = `03`
	loZeroMinute  = `04`
	loZeroSecond  = `05`
	loYear        = `06`
	loHour        = `15`
	loNumMonth    = `1`
	loLongYear    = `2006`
	loDay         = `2`
	loUnderDay    = `_2`
	loHour12      = `3`
	loMinute      = `4`
	loSecond      = `5`
	loPM          = `PM`
	loPm          = `pm`

	loNumColonSecondsTZ = `-07:00:00`
	loNumTZ             = `-0700`
	loNumColonTZ        = `-07:00`
	loNumShortTZ        = `-07`

	/* Potential future additions
	loISO8601SecondsTZ      = `Z070000`
	loISO8601ColonSecondsTZ = `Z07:00:00`
	loISO8601TZ             = `Z0700`
	loISO8601ColonTZ        = `Z07:00`
	loISO8601ShortTz        = `Z07`
	loFracSecond0           = `.000`
	loFracSecond9           = `.999`
	*/
)

type (
	TimestampFormat struct {
		format string
		layout string
	}

	TimestampFormatParser struct {
		lock    sync.Mutex
		formats map[string]*TimestampFormat
	}
)

func NewTimestampFormatParser() *TimestampFormatParser {
	return &TimestampFormatParser{formats: make(map[string]*TimestampFormat, 17)}
}

func (p *TimestampFormatParser) ParseFormat(format string) *TimestampFormat {
	p.lock.Lock()
	defer p.lock.Unlock()

	if fmt, ok := p.formats[format]; ok {
		return fmt
	}
	bld := bytes.NewBufferString(``)
	strftimeToLayout(bld, format)
	fmt := &TimestampFormat{format, bld.String()}
	p.formats[format] = fmt
	return fmt
}

func (f *TimestampFormat) Format(t *Timestamp) string {
	return (*time.Time)(t).Format(f.layout)
}

func (f *TimestampFormat) Format2(t *Timestamp, tz string) string {
	return (*time.Time)(t).In(loadLocation(tz)).Format(f.layout)
}

func strftimeToLayout(bld *bytes.Buffer, str string) {
	state := stateLiteral
	colons := 0
	padChar := '0'
	width := -1
	formatStart := 0
	upper := false

	for pos, c := range str {
		if state == stateLiteral {
			if c == '%' {
				state = statePad
				formatStart = pos
				padChar = '0'
				width = -1
				upper = false
				colons = 0
			} else {
				bld.WriteRune(c)
			}
			continue
		}

		switch c {
		case '-':
			if state != statePad {
				panic(badFormatSpecifier(str, formatStart, pos))
			}
			padChar = 0
			state = stateWidth
		case '^':
			if state != statePad {
				panic(badFormatSpecifier(str, formatStart, pos))
			}
			upper = true
		case '_':
			if state != statePad {
				panic(badFormatSpecifier(str, formatStart, pos))
			}
			padChar = ' '
			state = stateWidth
		case 'Y':
			bld.WriteString(loLongYear)
			state = stateLiteral
		case 'y':
			bld.WriteString(loYear)
			state = stateLiteral
		case 'C':
			panic(notSupportedByGoTimeLayout(str, formatStart, pos, `century`))
		case 'm':
			switch padChar {
			case 0:
				bld.WriteString(loNumMonth)
			case '0':
				bld.WriteString(loZeroMonth)
			case ' ':
				panic(notSupportedByGoTimeLayout(str, formatStart, pos, `space padded month`))
			}
			state = stateLiteral
		case 'B':
			if upper {
				panic(notSupportedByGoTimeLayout(str, formatStart, pos, `upper cased month`))
			}
			bld.WriteString(loLongMonth)
			state = stateLiteral
		case 'b', 'h':
			if upper {
				panic(notSupportedByGoTimeLayout(str, formatStart, pos, `upper cased short month`))
			}
			bld.WriteString(loMonth)
			state = stateLiteral
		case 'd':
			switch padChar {
			case 0:
				bld.WriteString(loDay)
			case '0':
				bld.WriteString(loZeroDay)
			case ' ':
				bld.WriteString(loUnderDay)
			}
			state = stateLiteral
		case 'e':
			bld.WriteString(loUnderDay)
			state = stateLiteral
		case 'j':
			panic(notSupportedByGoTimeLayout(str, formatStart, pos, `year of the day`))
		case 'H':
			switch padChar {
			case ' ':
				panic(notSupportedByGoTimeLayout(str, formatStart, pos, `blank padded 24 hour`))
			case 0:
				panic(notSupportedByGoTimeLayout(str, formatStart, pos, `short 24 hour`))
			default:
				bld.WriteString(loHour)
			}
			state = stateLiteral
		case 'k':
			panic(notSupportedByGoTimeLayout(str, formatStart, pos, `blank padded 24 hour`))
		case 'I':
			bld.WriteString(loZeroHour12)
			state = stateLiteral
		case 'l':
			bld.WriteString(loHour12)
			state = stateLiteral
		case 'P':
			bld.WriteString(loPm)
			state = stateLiteral
		case 'p':
			bld.WriteString(loPM)
			state = stateLiteral
		case 'M':
			switch padChar {
			case ' ':
				panic(notSupportedByGoTimeLayout(str, formatStart, pos, `blank padded minute`))
			case 0:
				bld.WriteString(loMinute)
			default:
				bld.WriteString(loZeroMinute)
			}
			state = stateLiteral
		case 'S':
			switch padChar {
			case ' ':
				panic(notSupportedByGoTimeLayout(str, formatStart, pos, `blank padded second`))
			case 0:
				bld.WriteString(loSecond)
			default:
				bld.WriteString(loZeroSecond)
			}
			state = stateLiteral
		case 'L':
			if formatStart == 0 || str[formatStart-1] != '.' {
				panic(notSupportedByGoTimeLayout(str, formatStart, pos, `fraction not preceded by dot in format`))
			}
			if padChar == '0' {
				bld.WriteString(`000`)
			} else {
				bld.WriteString(`999`)
			}
			state = stateLiteral
		case 'N':
			if formatStart == 0 || str[formatStart-1] != '.' {
				panic(notSupportedByGoTimeLayout(str, formatStart, pos, `fraction not preceded by dot in format`))
			}
			digit := byte('9')
			if padChar == '0' {
				digit = '0'
			}
			w := width
			if width == -1 {
				w = 9
			}
			for i := 0; i < w; i++ {
				bld.WriteByte(digit)
			}
			state = stateLiteral
		case 'z':
			switch colons {
			case 0:
				bld.WriteString(loNumTZ)
			case 1:
				bld.WriteString(loNumColonTZ)
			case 2:
				bld.WriteString(loNumColonSecondsTZ)
			default:
				// Not entirely correct since loosely defined num TZ not supported in Go
				bld.WriteString(loNumShortTZ)
			}
			state = stateLiteral
		case 'Z':
			bld.WriteString(loTZ)
			state = stateLiteral
		case 'A':
			bld.WriteString(loLongWeekDay)
			state = stateLiteral
		case 'a':
			bld.WriteString(loWeekDay)
			state = stateLiteral
		case 'u', 'w':
			panic(notSupportedByGoTimeLayout(str, formatStart, pos, `numeric week day`))
		case 'G', 'g':
			panic(notSupportedByGoTimeLayout(str, formatStart, pos, `week based year`))
		case 'V':
			panic(notSupportedByGoTimeLayout(str, formatStart, pos, `week number of the based year`))
		case 's':
			panic(notSupportedByGoTimeLayout(str, formatStart, pos, `seconds since epoch`))
		case 'Q':
			panic(notSupportedByGoTimeLayout(str, formatStart, pos, `milliseconds since epoch`))
		case 't':
			bld.WriteString("\t")
			state = stateLiteral
		case 'n':
			bld.WriteString("\n")
			state = stateLiteral
		case '%':
			bld.WriteByte('%')
			state = stateLiteral
		case 'c':
			strftimeToLayout(bld, `%a %b %-d %T %Y`)
			state = stateLiteral
		case 'D', 'x':
			strftimeToLayout(bld, `%m/%d/%y`)
			state = stateLiteral
		case 'F':
			strftimeToLayout(bld, `%Y-%m-%d`)
			state = stateLiteral
		case 'r':
			strftimeToLayout(bld, `%I:%M:%S %p`)
			state = stateLiteral
		case 'R':
			strftimeToLayout(bld, `%H:%M`)
			state = stateLiteral
		case 'X', 'T':
			strftimeToLayout(bld, `%H:%M:%S`)
			state = stateLiteral
		case '+':
			strftimeToLayout(bld, `%a %b %-d %H:%M:%S %Z %Y`)
			state = stateLiteral
		default:
			if c < '0' || c > '9' {
				panic(badFormatSpecifier(str, formatStart, pos))
			}
			if state == statePad && c == '0' {
				padChar = '0'
			} else {
				n := int(c) - 0x30
				if width == -1 {
					width = n
				} else {
					width = width*10 + n
				}
			}
			state = stateWidth
		}
	}

	if state != stateLiteral {
		panic(badFormatSpecifier(str, formatStart, len(str)))
	}
}

func notSupportedByGoTimeLayout(str string, start, pos int, description string) issue.Reported {
	return px.Error(px.NotSupportedByGoTimeLayout, issue.H{`format_specifier`: str[start : pos+1], `description`: description})
}

func toTimestampFormats(fmt px.Value) []*TimestampFormat {
	formats := DefaultTimestampFormats
	switch fmt := fmt.(type) {
	case *Array:
		formats = make([]*TimestampFormat, fmt.Len())
		fmt.EachWithIndex(func(f px.Value, i int) {
			formats[i] = DefaultTimestampFormatParser.ParseFormat(f.String())
		})
	case stringValue:
		formats = []*TimestampFormat{DefaultTimestampFormatParser.ParseFormat(fmt.String())}
	}
	return formats
}
