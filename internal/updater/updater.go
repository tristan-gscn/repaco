package updater

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var (
	// Injected at build time via ldflags
	CurrentVersion = "0.1.0"
	GitCommit      = ""
	BuildDate      = ""

	// GitHub repository coordinates
	RepoOwner = "tristan-gscn"
	RepoName  = "repaco"
)

// ReleaseInfo holds the metadata of a remote release
type ReleaseInfo struct {
	Version      string `json:"version"`
	Commit       string `json:"commit"`
	PublishedAt  string `json:"published_at"`
	TagName      string `json:"tag_name"`
	TargetCommit string `json:"target_commitish"`
	AssetURL     string `json:"asset_url"`
	AssetName    string `json:"asset_name"`
}

// GetBinaryName returns the asset binary name for the given OS and architecture.
func GetBinaryName(goos, goarch string) (string, error) {
	switch goos {
	case "darwin":
		switch goarch {
		case "arm64":
			return "repaco-darwin-arm64", nil
		case "amd64":
			return "repaco-darwin-amd64", nil
		}
	case "linux":
		switch goarch {
		case "amd64":
			return "repaco-linux-amd64", nil
		case "arm64":
			return "repaco-linux-arm64", nil
		}
	case "windows":
		if goarch == "amd64" {
			return "repaco-windows-amd64.exe", nil
		}
	}
	return "", fmt.Errorf("plateforme non supportée : %s/%s", goos, goarch)
}

// IsDifferent checks whether the remote version/commit differs from local.
func IsDifferent(currentVer, currentCommit, remoteVer, remoteCommit string) bool {
	// If current version is explicitly 'dev', do not consider it different
	if currentVer == "dev" {
		return false
	}

	// Compare commit hashes if both are available
	if currentCommit != "" && remoteCommit != "" {
		cShort := currentCommit
		if len(cShort) > 7 {
			cShort = cShort[:7]
		}
		rShort := remoteCommit
		if len(rShort) > 7 {
			rShort = rShort[:7]
		}
		return !strings.HasPrefix(remoteCommit, currentCommit) &&
			!strings.HasPrefix(currentCommit, remoteCommit) &&
			cShort != rShort
	}

	// Compare version strings if available
	if remoteVer != "" && currentVer != "" && remoteVer != currentVer {
		return true
	}

	// If remote has a commit and local has none and local is default initial version
	if remoteCommit != "" && currentCommit == "" && (currentVer == "0.1.0" || currentVer == "latest") {
		return true
	}

	return false
}

// CheckForUpdate queries GitHub for the latest release information.
func CheckForUpdate(ctx context.Context) (*ReleaseInfo, bool, error) {
	binaryName, err := GetBinaryName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, false, err
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	var info ReleaseInfo

	// 1. Try static version.json from GitHub Releases (fast and rate-limit free)
	versionURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/latest/version.json", RepoOwner, RepoName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionURL, nil)
	if err == nil {
		req.Header.Set("User-Agent", "repaco-updater")
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				if err == nil && json.Unmarshal(body, &info) == nil {
					info.AssetName = binaryName
					info.AssetURL = fmt.Sprintf("https://github.com/%s/%s/releases/download/latest/%s", RepoOwner, RepoName, binaryName)
					isDiff := IsDifferent(CurrentVersion, GitCommit, info.Version, info.Commit)
					return &info, isDiff, nil
				}
			}
		}
	}

	// 2. Fallback to GitHub Release API
	apiReleaseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/latest", RepoOwner, RepoName)
	reqAPI, err := http.NewRequestWithContext(ctx, http.MethodGet, apiReleaseURL, nil)
	if err == nil {
		reqAPI.Header.Set("User-Agent", "repaco-updater")
		resp, err := client.Do(reqAPI)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var ghRelease struct {
					TagName      string `json:"tag_name"`
					TargetCommit string `json:"target_commitish"`
					PublishedAt  string `json:"published_at"`
				}
				if json.NewDecoder(resp.Body).Decode(&ghRelease) == nil {
					info.Version = ghRelease.TagName
					info.Commit = ghRelease.TargetCommit
					info.PublishedAt = ghRelease.PublishedAt
					info.AssetName = binaryName
					info.AssetURL = fmt.Sprintf("https://github.com/%s/%s/releases/download/latest/%s", RepoOwner, RepoName, binaryName)
					isDiff := IsDifferent(CurrentVersion, GitCommit, info.Version, info.Commit)
					return &info, isDiff, nil
				}
			}
		}
	}

	// 3. Fallback to GitHub Commits API on main branch
	apiCommitURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/main", RepoOwner, RepoName)
	reqCommit, err := http.NewRequestWithContext(ctx, http.MethodGet, apiCommitURL, nil)
	if err == nil {
		reqCommit.Header.Set("User-Agent", "repaco-updater")
		resp, err := client.Do(reqCommit)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var ghCommit struct {
					SHA string `json:"sha"`
				}
				if json.NewDecoder(resp.Body).Decode(&ghCommit) == nil {
					info.Version = "latest"
					info.Commit = ghCommit.SHA
					info.AssetName = binaryName
					info.AssetURL = fmt.Sprintf("https://github.com/%s/%s/releases/download/latest/%s", RepoOwner, RepoName, binaryName)
					isDiff := IsDifferent(CurrentVersion, GitCommit, info.Version, info.Commit)
					return &info, isDiff, nil
				}
			}
		}
	}

	return nil, false, fmt.Errorf("impossible de vérifier les mises à jour sur GitHub")
}

// DownloadAndInstall downloads the new binary from the given asset URL and replaces the running executable.
func DownloadAndInstall(ctx context.Context, assetURL string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("impossible de localiser le binaire repaco actuel : %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("résolution de lien symbolique échouée : %w", err)
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return fmt.Errorf("requête de téléchargement invalide : %w", err)
	}
	req.Header.Set("User-Agent", "repaco-updater")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("échec de connexion lors du téléchargement : %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("erreur HTTP %d lors du téléchargement depuis %s", resp.StatusCode, assetURL)
	}

	// Create temporary binary file in the same directory as the executable to ensure atomic rename
	dir := filepath.Dir(exePath)
	tmpFile, err := os.CreateTemp(dir, ".repaco-update-*.tmp")
	if err != nil {
		// If creating in the same dir fails (e.g. read-only system dir), try os.TempDir()
		tmpFile, err = os.CreateTemp("", ".repaco-update-*.tmp")
		if err != nil {
			return fmt.Errorf("impossible de créer un fichier temporaire : %w", err)
		}
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("erreur lors de l'écriture du binaire téléchargé : %w", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("impossible de définir les permissions d'exécution : %w", err)
	}

	// Atomic replace
	if err := os.Rename(tmpPath, exePath); err != nil {
		// If rename fails (e.g. cross-device link), attempt copy and replace
		src, errOpen := os.Open(tmpPath)
		if errOpen != nil {
			return fmt.Errorf("permission refusée pour mettre à jour %s : %w", exePath, err)
		}
		defer src.Close()

		dst, errCreate := os.OpenFile(exePath, os.O_WRONLY|os.O_TRUNC, 0755)
		if errCreate != nil {
			return fmt.Errorf("permission refusée pour écraser %s (essayez avec sudo) : %w", exePath, errCreate)
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return fmt.Errorf("échec de l'écrasement du binaire : %w", err)
		}
	}

	return nil
}

// Relaunch restarts the repaco process with the newly installed binary.
func Relaunch() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		cmd := exec.Command(exePath, os.Args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return err
		}
		os.Exit(0)
		return nil
	}

	// POSIX exec replacement
	return syscall.Exec(exePath, os.Args, os.Environ())
}

// CheckAndPromptUpdate checks for a new version on GitHub and interactively prompts the user to update.
// Returns true if repaco was updated (and execution relaunched/terminated).
func CheckAndPromptUpdate(ctx context.Context) bool {
	// Skip update check if explicitly disabled or in non-interactive environment
	if os.Getenv("REPACO_NO_UPDATE") == "1" || os.Getenv("CI") == "true" {
		return false
	}

	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	info, isDiff, err := CheckForUpdate(checkCtx)
	if err != nil || !isDiff || info == nil {
		// Silent continuation if no internet, timeout or up to date
		return false
	}

	currCommitStr := GitCommit
	if len(currCommitStr) > 7 {
		currCommitStr = currCommitStr[:7]
	}
	if currCommitStr == "" {
		currCommitStr = "local"
	}

	remoteCommitStr := info.Commit
	if len(remoteCommitStr) > 7 {
		remoteCommitStr = remoteCommitStr[:7]
	}
	if remoteCommitStr == "" {
		remoteCommitStr = info.Version
	}

	fmt.Println()
	fmt.Println("┌────────────────────────────────────────────────────────┐")
	fmt.Println("│  📦 Une nouvelle version de repaco est disponible !    │")
	fmt.Printf("│  Version actuelle : %-10s (%-10s)         │\n", CurrentVersion, currCommitStr)
	fmt.Printf("│  Dernière version : %-10s (%-10s)         │\n", info.Version, remoteCommitStr)
	fmt.Println("└────────────────────────────────────────────────────────┘")
	fmt.Print("Voulez-vous mettre à jour repaco maintenant ? [O/n] : ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false
	}

	choice := strings.TrimSpace(strings.ToLower(input))
	if choice == "" || choice == "o" || choice == "oui" || choice == "y" || choice == "yes" {
		fmt.Printf("⬇️  Téléchargement de %s depuis GitHub...\n", info.AssetName)
		dlCtx, dlCancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer dlCancel()

		if err := DownloadAndInstall(dlCtx, info.AssetURL); err != nil {
			fmt.Printf("❌ Échec de la mise à jour : %v\n", err)
			fmt.Println("Poursuite avec la version actuelle...")
			fmt.Println()
			return false
		}

		fmt.Println("✅ repaco a été mis à jour avec succès !")
		fmt.Println("🚀 Relancement de repaco...")
		fmt.Println()

		if err := Relaunch(); err != nil {
			fmt.Printf("Avertissement : relancement automatique impossible (%v). Veuillez relancer repaco manuellement.\n", err)
			os.Exit(0)
		}
		return true
	}

	fmt.Println("Mise à jour ignorée. Lancement de repaco...")
	fmt.Println()
	return false
}
