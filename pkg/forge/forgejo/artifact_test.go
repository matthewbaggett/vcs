package forgejo

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"

	"git.grey.ooo/packages/vcs/pkg/forge"
)

func TestCIArtifactsList(t *testing.T) {
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/o/r/actions/runs/7/artifacts" {
			http.NotFound(w, r)
			return
		}
		w.Write(fixture(t, "artifacts.json"))
	})
	arts, err := c.CIArtifacts(context.Background(), 7)
	if err != nil || len(arts) != 2 {
		t.Fatalf("arts=%+v err=%v", arts, err)
	}
	if arts[0] != (forge.Artifact{ID: 31, Name: "coverage", Size: 2048}) || !arts[1].Expired {
		t.Errorf("arts = %+v", arts)
	}
}

func TestCIArtifactsEmpty(t *testing.T) {
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"artifacts":[],"total_count":0}`)) })
	arts, err := c.CIArtifacts(context.Background(), 7)
	if err != nil || len(arts) != 0 {
		t.Fatalf("arts=%+v err=%v", arts, err)
	}
}

func TestCIArtifactsNotFoundOnForgejo(t *testing.T) {
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/version" {
			w.Write([]byte(`{"version":"16.0.5+gitea-1.22.0"}`))
			return
		}
		http.NotFound(w, r)
	})
	if _, err := c.CIArtifacts(context.Background(), 7); !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestCIArtifactsUnsupportedOnGitea(t *testing.T) {
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/version" {
			w.Write([]byte(`{"version":"1.22.0"}`))
			return
		}
		http.NotFound(w, r)
	})
	if _, err := c.CIArtifacts(context.Background(), 7); !errors.Is(err, forge.ErrUnsupported) {
		t.Fatalf("want ErrUnsupported, got %v", err)
	}
}

func TestCIArtifactGetStreamsZip(t *testing.T) {
	var auth string
	c := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/v1/repos/o/r/actions/artifacts/31/zip" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte("PK-zip-bytes"))
	})
	var buf bytes.Buffer
	n, err := c.CIArtifactGet(context.Background(), forge.Artifact{ID: 31, Name: "coverage"}, &buf)
	if err != nil || n != 12 || buf.String() != "PK-zip-bytes" || auth != "token tok" {
		t.Fatalf("n=%d err=%v body=%q auth=%q", n, err, buf.String(), auth)
	}
	if _, err := c.CIArtifactGet(context.Background(), forge.Artifact{ID: 99}, &buf); !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}
