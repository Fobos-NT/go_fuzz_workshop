package artifacts

import (
	"strconv"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
	mutator "github.com/yandex-cloud/go-protobuf-mutator"
	"google.golang.org/protobuf/proto"
)

func serializeJSON(v *JSON) string {
	if v == nil {
		return "null"
	}

	switch x := v.Value.(type) {
	case *JSON_BoolValue:
		if x.BoolValue {
			return "true"
		}
		return "false"

	case *JSON_NumberValue:
		return strconv.FormatFloat(x.NumberValue, 'g', -1, 64)

	case *JSON_StringValue:
		return strconv.Quote(x.StringValue)

	case *JSON_NullValue:
		return "null"

	case *JSON_ObjectValue:
		if x.ObjectValue == nil {
			return "{}"
		}

		var b strings.Builder
		b.WriteByte('{')

		first := true

		for _, field := range x.ObjectValue.Fields {
			if field == nil {
				continue
			}

			if !first {
				b.WriteByte(',')
			}
			first = false

			b.WriteString(strconv.Quote(field.Key))
			b.WriteByte(':')
			b.WriteString(serializeJSON(field.Value))
		}

		b.WriteByte('}')
		return b.String()

	case *JSON_ArrayValue:
		if x.ArrayValue == nil {
			return "[]"
		}

		var b strings.Builder
		b.WriteByte('[')

		for i, value := range x.ArrayValue.Values {
			if i > 0 {
				b.WriteByte(',')
			}

			b.WriteString(serializeJSON(value))
		}

		b.WriteByte(']')
		return b.String()

	default:
		return "null"
	}
}

func FuzzParseJSON(f *testing.F) {
	// protobuf: object {}
	seed1 := []byte{0x22, 0x00}

	// protobuf: array {}
	seed2 := []byte{0x2a, 0x00}

	f.Add(seed1)
	f.Add(seed2)

	f.Fuzz(func(t *testing.T, data []byte) {
		var message JSON

		if err := proto.Unmarshal(data, &message); err != nil {
			return
		}

		m := mutator.New(1, 4096)

		if err := m.MutateProto(&message); err != nil {
			return
		}

		json := serializeJSON(&message)

		gjson.Parse(json)
	})
}
