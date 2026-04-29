package main

import (
	"errors"
	"net/url"
	"testing"
)

func mustParseURL(t *testing.T, rawPath string) *url.URL {
	t.Helper()
	u, err := url.Parse("http://example.com" + rawPath)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestParseURLID(t *testing.T) {
	const tracksPrefix = "/tracks/"

	tests := []struct {
		name    string
		path    string
		prefix  string
		wantID  int64
		wantErr error
	}{
		{
			name:    "positive id",
			path:    "/tracks/42",
			prefix:  tracksPrefix,
			wantID:  42,
			wantErr: nil,
		},
		{
			name:    "id one",
			path:    "/tracks/1",
			prefix:  tracksPrefix,
			wantID:  1,
			wantErr: nil,
		},
		{
			name:    "large id",
			path:    "/tracks/9223372036854775807",
			prefix:  tracksPrefix,
			wantID:  9223372036854775807,
			wantErr: nil,
		},
		{
			name:    "trim spaces around id segment",
			path:    "/tracks/ 7 ",
			prefix:  tracksPrefix,
			wantID:  7,
			wantErr: nil,
		},
		{
			name:    "empty after prefix",
			path:    "/tracks/",
			prefix:  tracksPrefix,
			wantID:  0,
			wantErr: errURLPathNotFound,
		},
		{
			name:    "only prefix path",
			path:    "/tracks/",
			prefix:  "/tracks/",
			wantID:  0,
			wantErr: errURLPathNotFound,
		},
		{
			name:    "extra path segment",
			path:    "/tracks/1/extra",
			prefix:  tracksPrefix,
			wantID:  0,
			wantErr: errURLPathNotFound,
		},
		{
			name:    "path without trailing slash before id",
			path:    "/tracks",
			prefix:  tracksPrefix,
			wantID:  0,
			wantErr: errURLPathNotFound,
		},
		{
			name:    "non-numeric id",
			path:    "/tracks/abc",
			prefix:  tracksPrefix,
			wantID:  0,
			wantErr: errURLInvalidID,
		},
		{
			name:    "empty id segment with spaces",
			path:    "/tracks/   ",
			prefix:  tracksPrefix,
			wantID:  0,
			wantErr: errURLPathNotFound,
		},
		{
			name:    "zero id",
			path:    "/tracks/0",
			prefix:  tracksPrefix,
			wantID:  0,
			wantErr: errURLInvalidID,
		},
		{
			name:    "negative id",
			path:    "/tracks/-5",
			prefix:  tracksPrefix,
			wantID:  0,
			wantErr: errURLInvalidID,
		},
		{
			name:    "hex not accepted",
			path:    "/tracks/0x10",
			prefix:  tracksPrefix,
			wantID:  0,
			wantErr: errURLInvalidID,
		},
		{
			name:    "float not accepted",
			path:    "/tracks/3.14",
			prefix:  tracksPrefix,
			wantID:  0,
			wantErr: errURLInvalidID,
		},
		{
			name:    "wrong prefix leaves slash in remainder",
			path:    "/tracks/99",
			prefix:  "/other/",
			wantID:  0,
			wantErr: errURLPathNotFound,
		},
		{
			name:    "empty raw path",
			path:    "",
			prefix:  tracksPrefix,
			wantID:  0,
			wantErr: errURLPathNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := mustParseURL(t, tt.path)
			gotID, err := parseURLID(u, tt.prefix)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("parseURLID() err = %v, want nil", err)
				}
				if gotID != tt.wantID {
					t.Errorf("parseURLID() id = %d, want %d", gotID, tt.wantID)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("parseURLID() err = %v, want %v", err, tt.wantErr)
			}
			if gotID != 0 {
				t.Errorf("parseURLID() id = %d, want 0 on error", gotID)
			}
		})
	}
}
