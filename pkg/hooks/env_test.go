package hooks

import (
	"reflect"
	"testing"
)

func Test_valuesToEnv(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]interface{}
		want   map[string]string
	}{{
		name:   "empty map",
		values: map[string]interface{}{},
		want:   map[string]string{},
	}, {
		name:   "simple map",
		values: map[string]interface{}{"key": "value"},
		want:   map[string]string{"KEY": "value"},
	}, {
		name: "nested map",
		values: map[string]interface{}{
			"key": map[string]interface{}{
				"nested": "value",
			},
		},
		want: map[string]string{"KEY__NESTED": "value"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := valuesToEnv(tt.values, "")
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("flattenMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
