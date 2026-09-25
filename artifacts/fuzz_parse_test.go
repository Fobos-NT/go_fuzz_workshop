package gjson

import "testing"

func FuzzParseJSONUsual(f *testing.F) {
	f.Fuzz(func(t *testing.T, input string) {
		Parse(input)
	})
}
