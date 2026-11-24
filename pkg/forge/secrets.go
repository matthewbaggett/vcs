package forge

import "context"

// Secret is one repository secret. The server never returns the value, so the type holds no value.
type Secret struct {
	Name    string
	Created string
}

// SecretManager lists, sets, and deletes the Actions secrets of the repository.
type SecretManager interface {
	SecretList(ctx context.Context) ([]Secret, error)
	SecretSet(ctx context.Context, name, value string) error
	SecretDelete(ctx context.Context, name string) error
}
