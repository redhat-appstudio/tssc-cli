package integrations

import (
	"fmt"
)

// ErrSecretAlreadyExists is returned when the integration secret already exists.
var ErrSecretAlreadyExists = fmt.Errorf("secret already exists")
