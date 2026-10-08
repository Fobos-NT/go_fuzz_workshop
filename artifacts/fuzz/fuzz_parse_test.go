package gjson

import (
	"testing"

	"github.com/tidwall/gjson"
)

func FuzzParseJSONUsual(f *testing.F) {
	f.Fuzz(func(t *testing.T, input string) {
		gjson.Parse(input)
	})
}
