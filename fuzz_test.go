package gjson

import (
	"strconv"
	"strings"
	"testing"

	"go_fuzz_workshop/artifacts"

	"github.com/tidwall/gjson"
)

func serializeJSON(v *artifacts.JSON) string {
	if v == nil {
		return "null"
	}

	switch x := v.Value.(type) {
	case *artifacts.JSON_BoolValue:
		if x.BoolValue {
			return "true"
		}
		return "false"

	case *artifacts.JSON_NumberValue:
		return strconv.FormatFloat(x.NumberValue, 'g', -1, 64)

	case *artifacts.JSON_StringValue:
		return strconv.Quote(x.StringValue)

	case *artifacts.JSON_NullValue:
		return "null"

	case *artifacts.JSON_ObjectValue:
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

	case *artifacts.JSON_ArrayValue:
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
	f.Fuzz(func(t *testing.T, message *artifacts.JSON) {
		gjson.Parse(serializeJSON(message))
	})
}
