package resolver

import (
	"reflect"
	"testing"
)

func Test_commaSeparatedToSlice(t *testing.T) {
	tests := []struct {
		name           string
		commaSeparated string
		want           []string
	}{{
		name:           "empty string",
		commaSeparated: "",
		want:           nil,
	}, {
		name:           "single element",
		commaSeparated: "1",
		want:           []string{"1"},
	}, {
		name:           "multiple elements with spaces",
		commaSeparated: "1, 2, 3",
		want:           []string{"1", "2", "3"},
	}, {
		name:           "multiple elements without spaces",
		commaSeparated: "1,2,3,",
		want:           []string{"1", "2", "3"},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commaSeparatedToSlice(tt.commaSeparated)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("commaSeparatedToSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}
