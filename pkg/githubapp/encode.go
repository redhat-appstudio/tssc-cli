package githubapp

import (
	"reflect"

	"github.com/google/go-github/v60/github"
)

// encodeGithubAppConfig converts each field of AppConfig to a base64
// encoded string.
func EncodeGithubAppConfig(appConfig *github.AppConfig) map[string][]byte {
	data := make(map[string][]byte)
	val := reflect.ValueOf(appConfig).Elem()
	for i := 0; i < val.NumField(); i++ {
		k := val.Type().Field(i).Name
		v := github.Stringify(val.Field(i).Interface())
		data[k] = []byte(v)
	}
	return data
}
