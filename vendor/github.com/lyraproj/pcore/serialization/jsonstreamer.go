package serialization

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/lyraproj/pcore/pcore"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

// NewJsonStreamer creates a new streamer that will produce JSON when
// receiving values
func NewJsonStreamer(out io.Writer) px.ValueConsumer {
	return &jsonStreamer{out, firstInArray}
}

type jsonStreamer struct {
	out   io.Writer
	state int
}

// DataToJson streams the given value to a Json ValueConsumer using a
// Serializer. This function is deprecated Use a Serializer directly with
// NewJsonStreamer
func DataToJson(value px.Value, out io.Writer) {
	he := make([]*types.HashEntry, 0, 2)
	he = append(he, types.WrapHashEntry2(`rich_data`, types.BooleanFalse))
	NewSerializer(pcore.RootContext(), types.WrapHash(he)).Convert(value, NewJsonStreamer(out))
	assertOk(out.Write([]byte("\n")))
}

func (j *jsonStreamer) AddArray(len int, doer px.Doer) {
	j.delimit(func() {
		j.state = firstInArray
		assertOk(j.out.Write([]byte{'['}))
		doer()
		assertOk(j.out.Write([]byte{']'}))
	})
}

func (j *jsonStreamer) AddHash(len int, doer px.Doer) {
	j.delimit(func() {
		assertOk(j.out.Write([]byte{'{'}))
		j.state = firstInObject
		doer()
		assertOk(j.out.Write([]byte{'}'}))
	})
}

func (j *jsonStreamer) Add(element px.Value) {
	j.delimit(func() {
		j.write(element)
	})
}

func (j *jsonStreamer) AddRef(ref int) {
	j.delimit(func() {
		assertOk(fmt.Fprintf(j.out, `{"%s":%d}`, PcoreRefKey, ref))
	})
}

func (j *jsonStreamer) CanDoBinary() bool {
	return false
}

func (j *jsonStreamer) CanDoComplexKeys() bool {
	return false
}

func (j *jsonStreamer) StringDedupThreshold() int {
	return 20
}

func (j *jsonStreamer) delimit(doer px.Doer) {
	switch j.state {
	case firstInArray:
		doer()
		j.state = afterElement
	case firstInObject:
		doer()
		j.state = afterKey
	case afterKey:
		assertOk(j.out.Write([]byte{':'}))
		doer()
		j.state = afterValue
	case afterValue:
		assertOk(j.out.Write([]byte{','}))
		doer()
		j.state = afterKey
	default: // Element
		assertOk(j.out.Write([]byte{','}))
		doer()
	}
}

func (j *jsonStreamer) write(e px.Value) {
	var v []byte
	var err error
	switch e := e.(type) {
	case px.StringValue:
		v, err = json.Marshal(e.String())
	case px.Float:
		v, err = json.Marshal(e.Float())
	case px.Integer:
		v, err = json.Marshal(e.Int())
	case px.Boolean:
		v, err = json.Marshal(e.Bool())
	default:
		v = []byte(`null`)
	}
	assertOk(0, err)
	assertOk(j.out.Write(v))
}

func assertOk(_ int, err error) {
	if err != nil {
		panic(px.Error(px.Failure, issue.H{`message`: err}))
	}
}
