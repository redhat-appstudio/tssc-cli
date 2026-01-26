package engine

import (
	"encoding/json"

	"helm.sh/helm/v3/pkg/chartutil"
)

// UnstructuredType converts an unstructured item into a chartutil.Values.
func UnstructuredType(item interface{}) (chartutil.Values, error) {
	data, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}
	return Unstructured(data)
}

// Unstructured converts a JSON payload into a chartutil.Values.
func Unstructured(payload []byte) (chartutil.Values, error) {
	var result chartutil.Values
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}
