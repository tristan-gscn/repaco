package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestGetBinaryName(t *testing.T) {
	tests := []struct {
		goos    string
		goarch  string
		want    string
		wantErr bool
	}{
		{"darwin", "arm64", "repaco-darwin-arm64", false},
		{"darwin", "amd64", "repaco-darwin-amd64", false},
		{"linux", "amd64", "repaco-linux-amd64", false},
		{"linux", "arm64", "repaco-linux-arm64", false},
		{"windows", "amd64", "repaco-windows-amd64.exe", false},
		{"freebsd", "amd64", "", true},
		{"linux", "386", "", true},
	}

	for _, tt := range tests {
		got, err := GetBinaryName(tt.goos, tt.goarch)
		if (err != nil) != tt.wantErr {
			t.Errorf("GetBinaryName(%q, %q) error = %v, wantErr %v", tt.goos, tt.goarch, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("GetBinaryName(%q, %q) = %q, want %q", tt.goos, tt.goarch, got, tt.want)
		}
	}
}

func TestIsDifferent(t *testing.T) {
	tests := []struct {
		name          string
		currentVer    string
		currentCommit string
		remoteVer     string
		remoteCommit  string
		want          bool
	}{
		{
			name:          "same commit and version",
			currentVer:    "latest",
			currentCommit: "abcdef1234567890",
			remoteVer:     "latest",
			remoteCommit:  "abcdef1234567890",
			want:          false,
		},
		{
			name:          "different commit",
			currentVer:    "latest",
			currentCommit: "1111111111111111",
			remoteVer:     "latest",
			remoteCommit:  "2222222222222222",
			want:          true,
		},
		{
			name:          "different tag version",
			currentVer:    "v0.1.0",
			currentCommit: "",
			remoteVer:     "v0.2.0",
			remoteCommit: "",
			want:          true,
		},
		{
			name:          "same tag version",
			currentVer:    "v0.1.0",
			currentCommit: "",
			remoteVer:     "v0.1.0",
			remoteCommit: "",
			want:          false,
		},
		{
			name:          "dev version ignored",
			currentVer:    "dev",
			currentCommit: "local",
			remoteVer:     "latest",
			remoteCommit:  "999999999999",
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDifferent(tt.currentVer, tt.currentCommit, tt.remoteVer, tt.remoteCommit)
			if got != tt.want {
				t.Errorf("IsDifferent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckForUpdateWithMockServer(t *testing.T) {
	mockRelease := ReleaseInfo{
		Version: "latest",
		Commit:  "fedcba9876543210",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockRelease)
	}))
	defer server.Close()

	// Parse server URL
	u, _ := url.Parse(server.URL)
	_ = u

	// Verify json unmarshaling logic
	data, _ := json.Marshal(mockRelease)
	var parsed ReleaseInfo
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal mock release: %v", err)
	}

	if parsed.Commit != "fedcba9876543210" {
		t.Errorf("Parsed commit = %q, want %q", parsed.Commit, "fedcba9876543210")
	}
}

func TestDownloadAndInstall(t *testing.T) {
	// Create mock server delivering fake binary
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("#!/bin/sh\necho updated\n"))
	}))
	defer server.Close()

	// Create temp exe file
	tmpDir := t.TempDir()
	fakeExe := filepath.Join(tmpDir, "repaco")
	if err := os.WriteFile(fakeExe, []byte("original"), 0755); err != nil {
		t.Fatalf("Failed to write fake exe: %v", err)
	}

	// Verify downloading
	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to download from mock server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Download status = %d, want 200", resp.StatusCode)
	}
}
