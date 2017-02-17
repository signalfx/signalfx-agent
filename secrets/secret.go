package secrets

import (
	"errors"
	"fmt"
	"os"
)

// SecretKeeper is a function that given a key returns an unecrypted secret
type SecretKeeper func(name string) (string, error)

// EnvSecret returns secret from environment variable
func EnvSecret(name string) (string, error) {
	secret, ok := os.LookupEnv(name)
	if ok {
		return secret, nil
	}
	return "", fmt.Errorf("no environment variable %s", name)
}

// K8Secret returns a secret from Kubernetes (unimplemented)
func K8Secret(name string) (string, error) {
	return "", errors.New("kubernetes secrets unimplemented")
}

// SecretKeepers is a list of available secretkeepers
var SecretKeepers = []SecretKeeper{
	EnvSecret,
	K8Secret,
}
