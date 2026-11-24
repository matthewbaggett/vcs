package forge

import (
	"context"
	"io"
)

// Package is one version of a generic package of the repository owner.
type Package struct {
	Name    string
	Version string
	Created string // an RFC 3339 time, as the server gives it
}

// PackageManager lists, uploads, links, and deletes generic packages of the owner of the repository.
// No other registry type is reachable.
type PackageManager interface {
	// PackageList lists the versions of one name, or of every generic package with an empty name.
	PackageList(ctx context.Context, name string) ([]Package, error)
	// PackageUpload sends one file with a PUT. It never replaces: the caller checks first.
	PackageUpload(ctx context.Context, name, version, file string, r io.Reader, size int64) error
	// PackageLink links a package to the repository, so it shows on the page of the repository.
	PackageLink(ctx context.Context, name string) error
	// PackageDelete removes one version.
	PackageDelete(ctx context.Context, name, version string) error
	// PackageURL gives the download address of one file of one version.
	PackageURL(name, version, file string) string
}
