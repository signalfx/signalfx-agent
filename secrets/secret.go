package secrets

import (
	"errors"
	"fmt"
	"os"
)

type SecretKeeper func(name string) (string, error)

func EnvSecret(name string) (string, error) {
	secret, ok := os.LookupEnv(name)
	if ok {
		return secret, nil
	}
	return "", fmt.Errorf("no environment variable %s", name)
}

func K8Secret(name string) (string, error) {
	return "", errors.New("kubernetes secrets unimplemented")
}

var SecretKeepers = []SecretKeeper{
	EnvSecret,
	K8Secret,
}
