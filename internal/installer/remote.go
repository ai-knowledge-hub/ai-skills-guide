package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
	"github.com/gofrs/flock"
)

const (
	maxRegistryBytes          = 10 << 20
	maxCompressedArchiveBytes = 100 << 20
	maxExtractedArchiveBytes  = 250 << 20
	maxArchiveFiles           = 10000
)

type RemoteInstallOptions struct {
	RegistryURL       string
	AllowedOrigins    []string
	CacheDir          string
	Offline           bool
	Module            string
	Runtime           string
	ExecutionRuntimes []string
	ID                string
	Version           string
	TargetRoot        string
	Force             bool
	HTTPClient        *http.Client
}

type RemoteInstallResult struct {
	Destination string
	Version     string
	Registry    registry.SkillEntry
	FromCache   bool
}

type installTransaction struct {
	Destination string `json:"destination"`
	Stage       string `json:"stage"`
	Backup      string `json:"backup"`
}

type installTreePlan struct {
	SourceDir  string
	TargetRoot string
	ID         string
}

type closureInstallOperation struct {
	Destination string `json:"destination"`
	Stage       string `json:"stage"`
	Backup      string `json:"backup,omitempty"`
	HadPrevious bool   `json:"had_previous"`
}

type closureInstallTransaction struct {
	Phase      string                    `json:"phase"`
	Operations []closureInstallOperation `json:"operations"`
}

var installTransactionFault func(string) error
var closureInstallTransactionFault func(string, int) error

// InstallRemoteRelease resolves, verifies, validates, and atomically commits a
// released package. Runtime state is untouched until every remote input passes.
func InstallRemoteRelease(ctx context.Context, options RemoteInstallOptions) (RemoteInstallResult, error) {
	if strings.TrimSpace(options.RegistryURL) == "" {
		return RemoteInstallResult{}, errors.New("registry resolution failed: --registry-url is required; retry with an HTTPS registry URL")
	}
	if strings.TrimSpace(options.CacheDir) == "" {
		return RemoteInstallResult{}, errors.New("cache preparation failed: cache directory is required; choose a writable --cache-dir")
	}
	if err := validatePackageID(options.ID); err != nil {
		return RemoteInstallResult{}, fmt.Errorf("registry resolution failed: %w", err)
	}
	if strings.TrimSpace(options.TargetRoot) == "" {
		return RemoteInstallResult{}, errors.New("install preparation failed: target root is required")
	}
	targetRoot, err := filepath.Abs(options.TargetRoot)
	if err != nil {
		return RemoteInstallResult{}, fmt.Errorf("install preparation failed: resolve target root: %w", err)
	}
	if err := recoverInterruptedClosureInstalls(filepath.Dir(targetRoot)); err != nil {
		return RemoteInstallResult{}, fmt.Errorf("install recovery failed: %w", err)
	}
	if err := recoverInterruptedInstalls(options.TargetRoot); err != nil {
		return RemoteInstallResult{}, fmt.Errorf("install recovery failed: %w", err)
	}
	registryURL, allowedOrigins, err := validateRemoteOrigins(options.RegistryURL, options.AllowedOrigins)
	if err != nil {
		return RemoteInstallResult{}, fmt.Errorf("registry resolution failed: %w", err)
	}
	client := restrictedHTTPClient(options.HTTPClient, allowedOrigins)

	indexBytes, cacheRegistry, err := loadRegistryIndex(
		ctx,
		client,
		registryURL,
		options.CacheDir,
		options.ID,
		options.Version,
		options.Offline,
	)
	if err != nil {
		return RemoteInstallResult{}, err
	}
	index, err := registry.ParseIndex(indexBytes, registryURL.String())
	if err != nil {
		return RemoteInstallResult{}, fmt.Errorf("registry validation failed: %w; use a compatible registry snapshot", err)
	}
	entry, ok := registry.FindSkill(index, options.ID)
	if !ok {
		return RemoteInstallResult{}, fmt.Errorf("version resolution failed: entry %s is absent from the registry; check the module and ID", options.ID)
	}
	version, err := registry.ResolveVersion(entry, options.Version)
	if err != nil {
		return RemoteInstallResult{}, fmt.Errorf("version resolution failed: %w; choose a listed version", err)
	}
	if index.RegistryVersion != "1.3" {
		return RemoteInstallResult{}, fmt.Errorf("admission failed: registry version %s has no version-bound manifest digest; use a 1.3 registry for remote installation", index.RegistryVersion)
	}
	artifactURL, err := validateReleaseURL(version.ArtifactURL, "artifact_url", allowedOrigins)
	if err != nil {
		return RemoteInstallResult{}, fmt.Errorf("artifact URL validation failed: %w; use a registry with an allowed HTTPS artifact origin", err)
	}
	if len(version.SHA256) != sha256.Size*2 {
		return RemoteInstallResult{}, fmt.Errorf("checksum validation failed: registry checksum for %s@%s is not SHA-256; do not install this release", entry.ID, version.Version)
	}
	if _, err := hex.DecodeString(version.SHA256); err != nil {
		return RemoteInstallResult{}, fmt.Errorf("checksum validation failed: registry checksum for %s@%s is malformed; do not install this release", entry.ID, version.Version)
	}
	if _, err := validateReleaseURL(version.ManifestURL, "manifest_url", allowedOrigins); err != nil {
		return RemoteInstallResult{}, fmt.Errorf("manifest URL validation failed: %w; use a registry with an allowed HTTPS manifest origin", err)
	}
	if len(version.ManifestSHA256) != sha256.Size*2 {
		return RemoteInstallResult{}, fmt.Errorf("manifest validation failed: registry manifest digest for %s@%s is not SHA-256; do not install this release", entry.ID, version.Version)
	}
	if _, err := hex.DecodeString(version.ManifestSHA256); err != nil {
		return RemoteInstallResult{}, fmt.Errorf("manifest validation failed: registry manifest digest for %s@%s is malformed; do not install this release", entry.ID, version.Version)
	}

	archivePath, fromCache, err := loadArtifact(ctx, client, artifactURL, options.CacheDir, strings.ToLower(version.SHA256), options.Offline)
	if err != nil {
		return RemoteInstallResult{}, err
	}
	extractRoot, err := os.MkdirTemp(options.CacheDir, ".extract-")
	if err != nil {
		return RemoteInstallResult{}, fmt.Errorf("extraction preparation failed: %w; choose a writable --cache-dir", err)
	}
	defer os.RemoveAll(extractRoot)
	if err := extractVerifiedTarGz(archivePath, extractRoot); err != nil {
		return RemoteInstallResult{}, fmt.Errorf("archive validation failed: %w; remove the cached release and retry", err)
	}
	manifestName, err := manifestNameForModule(options.Module)
	if err != nil {
		return RemoteInstallResult{}, fmt.Errorf("package validation failed: %w", err)
	}
	manifestPath := filepath.Join(extractRoot, manifestName)
	if err := verifyFileSHA256(manifestPath, strings.ToLower(version.ManifestSHA256)); err != nil {
		return RemoteInstallResult{}, fmt.Errorf("manifest validation failed: archive manifest does not match the selected version's registry digest; the release was not installed")
	}
	manifest, err := registry.ValidatePackageManifest(manifestPath)
	if err != nil {
		return RemoteInstallResult{}, fmt.Errorf("package validation failed: %w; the release was not installed", err)
	}
	if manifest.ID != entry.ID || manifest.Version != version.Version {
		return RemoteInstallResult{}, fmt.Errorf("package identity validation failed: archive declares %s@%s, registry selected %s@%s; the release was not installed", manifest.ID, manifest.Version, entry.ID, version.Version)
	}
	if !supportsRuntime(manifest.Runtimes, options.Runtime) {
		return RemoteInstallResult{}, fmt.Errorf("package validation failed: archive does not declare runtime %s; the release was not installed", options.Runtime)
	}
	if err := validateExecutionCompatibility(manifest, options.Runtime, options.ExecutionRuntimes); err != nil {
		return RemoteInstallResult{}, fmt.Errorf("package compatibility validation failed: %w; the release was not installed", err)
	}
	manifestEntry := registry.ProjectManifest(manifest)
	if entry.Deprecated {
		manifestEntry.Deprecated = true
		manifestEntry.ReplacedBy = entry.ReplacedBy
	}
	manifestEntry = registry.ApplyLifecycleProjection(manifestEntry)
	if err := ValidateOperationalInstall(manifestEntry); err != nil {
		return RemoteInstallResult{}, fmt.Errorf("package admission failed: %w", err)
	}
	if strings.EqualFold(options.Module, "plugins") && manifestEntry.Includes != nil && !manifest.Artifact.SelfContained {
		return RemoteInstallResult{}, fmt.Errorf("package admission failed: plugin %s is not a self-contained release; the release was not installed", manifest.ID)
	}
	var installPlans []installTreePlan
	if strings.EqualFold(options.Module, "plugins") {
		if _, err := PreparePluginRuntimeArtifacts(extractRoot, options.Runtime); err != nil {
			return RemoteInstallResult{}, fmt.Errorf("runtime package validation failed: %w; the release was not installed", err)
		}
		installPlans, err = remotePluginInstallPlans(extractRoot, options.TargetRoot, options.Runtime, options.ExecutionRuntimes, manifest)
		if err != nil {
			return RemoteInstallResult{}, fmt.Errorf("plugin dependency admission failed: %w; the release was not installed", err)
		}
	}
	if cacheRegistry {
		selectors := []string{normalizedVersionSelector(options.Version)}
		if selectors[0] == "latest" {
			selectors = append(selectors, version.Version)
		}
		if err := writeRegistrySnapshots(options.CacheDir, registryURL, entry.ID, selectors, indexBytes); err != nil {
			return RemoteInstallResult{}, fmt.Errorf("registry cache commit failed: %w; existing cached snapshots were preserved", err)
		}
	}

	var destination string
	if len(installPlans) > 0 {
		destination, err = atomicInstallClosure(installPlans, options.TargetRoot, entry.ID, options.Force)
	} else {
		destination, err = atomicInstallTree(extractRoot, options.TargetRoot, entry.ID, options.Force)
	}
	if err != nil {
		return RemoteInstallResult{}, fmt.Errorf("install commit failed: %w; existing installation was preserved or left in a recoverable backup", err)
	}
	manifestEntry.Latest = version.Version
	manifestEntry.Versions = []registry.VersionEntry{version}
	return RemoteInstallResult{Destination: destination, Version: version.Version, Registry: manifestEntry, FromCache: fromCache}, nil
}
func validateRemoteOrigins(rawRegistryURL string, extra []string) (*url.URL, map[string]struct{}, error) {
	registryURL, err := url.Parse(rawRegistryURL)
	if err != nil || registryURL.Scheme != "https" || registryURL.Host == "" || registryURL.User != nil {
		return nil, nil, fmt.Errorf("registry URL must be an absolute HTTPS URL without user information")
	}
	origins := map[string]struct{}{originOf(registryURL): {}}
	for _, raw := range extra {
		candidate, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || candidate.Scheme != "https" || candidate.Host == "" || candidate.User != nil || (candidate.Path != "" && candidate.Path != "/") || candidate.RawQuery != "" || candidate.Fragment != "" {
			return nil, nil, fmt.Errorf("allowed origin %q must contain only an HTTPS scheme and host", raw)
		}
		origins[originOf(candidate)] = struct{}{}
	}
	return registryURL, origins, nil
}

func validateReleaseURL(raw, field string, origins map[string]struct{}) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return nil, fmt.Errorf("%s must be an absolute HTTPS URL without user information", field)
	}
	if _, ok := origins[originOf(parsed)]; !ok {
		return nil, fmt.Errorf("artifact origin %s is not allowed", originOf(parsed))
	}
	return parsed, nil
}

func originOf(parsed *url.URL) string {
	return strings.ToLower(parsed.Scheme + "://" + parsed.Host)
}

func restrictedHTTPClient(base *http.Client, origins map[string]struct{}) *http.Client {
	var client http.Client
	if base != nil {
		client = *base
	} else {
		client.Timeout = 30 * time.Second
	}
	previousRedirect := client.CheckRedirect
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		if request.URL.Scheme != "https" {
			return errors.New("redirect changed to a non-HTTPS URL")
		}
		if _, ok := origins[originOf(request.URL)]; !ok {
			return fmt.Errorf("redirect origin %s is not allowed", originOf(request.URL))
		}
		if previousRedirect != nil {
			return previousRedirect(request, via)
		}
		return nil
	}
	return &client
}

func loadRegistryIndex(ctx context.Context, client *http.Client, registryURL *url.URL, cacheDir, packageID, requestedVersion string, offline bool) ([]byte, bool, error) {
	if offline {
		data, err := loadLatestRegistrySnapshot(cacheDir, registryURL, packageID, requestedVersion)
		if err != nil {
			return nil, false, fmt.Errorf("offline registry lookup failed: no valid cached index for %s; run once online without --offline", registryURL)
		}
		return data, false, nil
	}
	data, err := fetchBounded(ctx, client, registryURL, maxRegistryBytes)
	if err != nil {
		return nil, false, fmt.Errorf("registry download failed: %w; retry online or use --offline after a successful fetch", err)
	}
	if _, err := registry.ParseIndex(data, registryURL.String()); err != nil {
		return nil, false, fmt.Errorf("registry validation failed: %w; cached registry was not replaced", err)
	}
	return data, true, nil
}

func normalizedVersionSelector(requested string) string {
	selector := strings.TrimSpace(requested)
	if selector == "" {
		return "latest"
	}
	return selector
}

func registrySnapshotDir(cacheDir string, registryURL *url.URL, packageID, requestedVersion string) string {
	selectionKey := cacheKey(packageID + "@" + normalizedVersionSelector(requestedVersion))
	return filepath.Join(cacheDir, "registries", cacheKey(registryURL.String()), selectionKey)
}

func loadLatestRegistrySnapshot(cacheDir string, registryURL *url.URL, packageID, requestedVersion string) ([]byte, error) {
	directory := registrySnapshotDir(cacheDir, registryURL, packageID, requestedVersion)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	for index := len(entries) - 1; index >= 0; index-- {
		entry := entries[index]
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			continue
		}
		if _, err := registry.ParseIndex(data, registryURL.String()); err == nil {
			return data, nil
		}
	}
	return nil, errors.New("no valid registry snapshot")
}

func writeRegistrySnapshots(cacheDir string, registryURL *url.URL, packageID string, selectors []string, data []byte) error {
	seen := make(map[string]struct{}, len(selectors))
	for _, selector := range selectors {
		normalized := normalizedVersionSelector(selector)
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		if err := writeRegistrySnapshot(cacheDir, registryURL, packageID, normalized, data); err != nil {
			return err
		}
	}
	return nil
}

func writeRegistrySnapshot(cacheDir string, registryURL *url.URL, packageID, requestedVersion string, data []byte) error {
	directory := registrySnapshotDir(cacheDir, registryURL, packageID, requestedVersion)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".candidate-")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	name := fmt.Sprintf("%020d-%s.json", time.Now().UnixNano(), strings.TrimPrefix(filepath.Base(temporaryPath), ".candidate-"))
	if err := os.Rename(temporaryPath, filepath.Join(directory, name)); err != nil {
		return err
	}
	pruneRegistrySnapshots(directory, 5)
	return nil
}

func pruneRegistrySnapshots(directory string, keep int) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return
	}
	remaining := keep
	for index := len(entries) - 1; index >= 0; index-- {
		entry := entries[index]
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if remaining > 0 {
			remaining--
			continue
		}
		_ = os.Remove(filepath.Join(directory, entry.Name()))
	}
}

func loadArtifact(ctx context.Context, client *http.Client, artifactURL *url.URL, cacheDir, expectedSHA string, offline bool) (string, bool, error) {
	cachePath := filepath.Join(cacheDir, "artifacts", expectedSHA+".tar.gz")
	if err := verifyFileSHA256(cachePath, expectedSHA); err == nil {
		return cachePath, true, nil
	}
	if offline {
		return "", false, fmt.Errorf("offline artifact lookup failed: verified cache entry %s is unavailable; run once online without --offline", expectedSHA)
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return "", false, fmt.Errorf("artifact cache preparation failed: %w; choose a writable --cache-dir", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(cachePath), ".download-")
	if err != nil {
		return "", false, fmt.Errorf("artifact download preparation failed: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, artifactURL.String(), nil)
	if err != nil {
		temporary.Close()
		return "", false, fmt.Errorf("artifact download failed: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		temporary.Close()
		return "", false, fmt.Errorf("artifact download failed: %w; retry when the registry is reachable", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		temporary.Close()
		return "", false, fmt.Errorf("artifact download failed: unexpected HTTP status %s; retry or select another version", response.Status)
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(temporary, hash), io.LimitReader(response.Body, maxCompressedArchiveBytes+1))
	closeErr := temporary.Close()
	if copyErr != nil || closeErr != nil {
		return "", false, fmt.Errorf("artifact download failed: partial download was discarded; retry when the connection is stable")
	}
	if written > maxCompressedArchiveBytes {
		return "", false, fmt.Errorf("artifact download failed: archive exceeds %d bytes; publish a smaller release", maxCompressedArchiveBytes)
	}
	actualSHA := hex.EncodeToString(hash.Sum(nil))
	if actualSHA != expectedSHA {
		return "", false, fmt.Errorf("checksum validation failed: expected %s but downloaded %s; cached and installed copies were not changed", expectedSHA, actualSHA)
	}
	// The cache is content-addressed. Any existing file at this path already
	// failed verification above, so removing it cannot discard trusted data and
	// makes replacement portable to platforms where Rename cannot overwrite.
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return "", false, fmt.Errorf("artifact cache repair failed: %w; remove the corrupt cache entry and retry", err)
	}
	if err := os.Rename(temporaryPath, cachePath); err != nil {
		return "", false, fmt.Errorf("artifact cache commit failed: %w; retry with a writable --cache-dir", err)
	}
	return cachePath, false, nil
}

func fetchBounded(ctx context.Context, client *http.Client, source *url.URL, limit int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source.String(), nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %s", response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return data, nil
}

func verifyFileSHA256(filePath, expected string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(file, maxCompressedArchiveBytes+1))
	if err != nil || written > maxCompressedArchiveBytes {
		return errors.New("cached artifact is unreadable or oversized")
	}
	if hex.EncodeToString(hash.Sum(nil)) != expected {
		return errors.New("cached artifact checksum does not match")
	}
	return nil
}

func extractVerifiedTarGz(archivePath, destination string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("open gzip stream: %w", err)
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	seen := make(map[string]struct{})
	var total int64
	files := 0
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}
		cleanName, err := safeArchivePath(header.Name)
		if err != nil {
			return err
		}
		if _, exists := seen[cleanName]; exists {
			return fmt.Errorf("duplicate archive path %q", header.Name)
		}
		seen[cleanName] = struct{}{}
		files++
		if files > maxArchiveFiles {
			return fmt.Errorf("archive exceeds %d entries", maxArchiveFiles)
		}
		target := filepath.Join(destination, filepath.FromSlash(cleanName))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if header.Size < 0 || header.Size > maxExtractedArchiveBytes-total {
				return fmt.Errorf("archive expands beyond %d bytes", maxExtractedArchiveBytes)
			}
			total += header.Size
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(0o644)
			if header.FileInfo().Mode()&0o111 != 0 {
				mode = 0o755
			}
			output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			copied, copyErr := io.CopyN(output, tarReader, header.Size)
			closeErr := output.Close()
			if copyErr != nil || copied != header.Size || closeErr != nil {
				return fmt.Errorf("extract %q: truncated or unwritable content", header.Name)
			}
		default:
			return fmt.Errorf("archive path %q uses unsupported entry type %d", header.Name, header.Typeflag)
		}
	}
	return nil
}

// ValidateRetainedRelease applies the remote installer's bounded archive and
// package-admission checks to an immutable publisher release without committing
// it into a runtime directory.
func ValidateRetainedRelease(archivePath, storedManifestPath, module, expectedID, expectedVersion string) (registry.Manifest, error) {
	archiveInfo, err := os.Lstat(archivePath)
	if err != nil || !archiveInfo.Mode().IsRegular() {
		return registry.Manifest{}, fmt.Errorf("release archive must be a regular file: %s", archivePath)
	}
	if archiveInfo.Size() > maxCompressedArchiveBytes {
		return registry.Manifest{}, fmt.Errorf("release archive exceeds %d bytes: %s", maxCompressedArchiveBytes, archivePath)
	}
	manifestInfo, err := os.Lstat(storedManifestPath)
	if err != nil || !manifestInfo.Mode().IsRegular() {
		return registry.Manifest{}, fmt.Errorf("release manifest must be a regular file: %s", storedManifestPath)
	}
	extractRoot, err := os.MkdirTemp("", ".release-validation-")
	if err != nil {
		return registry.Manifest{}, fmt.Errorf("create release validation directory: %w", err)
	}
	defer os.RemoveAll(extractRoot)
	if err := extractVerifiedTarGz(archivePath, extractRoot); err != nil {
		return registry.Manifest{}, fmt.Errorf("validate retained archive: %w", err)
	}
	manifestName, err := manifestNameForModule(module)
	if err != nil {
		return registry.Manifest{}, err
	}
	archivedManifestPath := filepath.Join(extractRoot, manifestName)
	storedManifest, err := os.ReadFile(storedManifestPath)
	if err != nil {
		return registry.Manifest{}, fmt.Errorf("read retained manifest: %w", err)
	}
	archivedManifest, err := os.ReadFile(archivedManifestPath)
	if err != nil {
		return registry.Manifest{}, fmt.Errorf("read archived manifest: %w", err)
	}
	if !bytes.Equal(storedManifest, archivedManifest) {
		return registry.Manifest{}, errors.New("retained manifest does not equal the archive manifest")
	}
	manifest, err := registry.ValidateHistoricalPackageManifest(archivedManifestPath)
	if err != nil {
		return registry.Manifest{}, fmt.Errorf("validate retained package: %w", err)
	}
	if manifest.ID != expectedID || manifest.Version != expectedVersion {
		return registry.Manifest{}, fmt.Errorf("retained package declares %s@%s, expected %s@%s", manifest.ID, manifest.Version, expectedID, expectedVersion)
	}
	return manifest, nil
}

func safeArchivePath(name string) (string, error) {
	if name == "" || strings.Contains(name, "\\") || strings.ContainsRune(name, 0) || path.IsAbs(name) {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != strings.TrimSuffix(name, "/") {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	return clean, nil
}

func manifestNameForModule(module string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(module)) {
	case "skills", "skill", "":
		return "skill.yaml", nil
	case "agents", "agent":
		return "agent.yaml", nil
	case "tools", "tool", "tools-mcp":
		return "tool.yaml", nil
	case "plugins", "plugin":
		return "plugin.yaml", nil
	default:
		return "", fmt.Errorf("unsupported module %q", module)
	}
}

func supportsRuntime(runtimes []string, runtime string) bool {
	for _, candidate := range runtimes {
		if strings.EqualFold(strings.TrimSpace(candidate), strings.TrimSpace(runtime)) {
			return true
		}
	}
	return false
}

func validatePackageID(id string) error {
	if id == "" || strings.Contains(id, "\\") || path.IsAbs(id) || path.Clean(id) != id || strings.Count(id, "/") != 1 {
		return fmt.Errorf("invalid package ID %q", id)
	}
	return nil
}

func cacheKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func remotePluginInstallPlans(extractRoot, pluginTargetRoot, runtimeName string, executionRuntimes []string, manifest registry.Manifest) ([]installTreePlan, error) {
	plans := make([]installTreePlan, 0, 1+len(manifest.Includes.Skills)+len(manifest.Includes.Agents)+len(manifest.Includes.Tools))
	// The plugin is transaction operation zero. Upgrades deactivate that path
	// before changing dependencies, and recovery restores it only after every
	// dependency has returned to its previous version.
	plans = append(plans, installTreePlan{SourceDir: extractRoot, TargetRoot: pluginTargetRoot, ID: manifest.ID})
	sets := []struct {
		moduleDir  string
		moduleName string
		manifest   string
		ids        []string
	}{
		{moduleDir: "skills", moduleName: "skills", manifest: "skill.yaml", ids: manifest.Includes.Skills},
		{moduleDir: "agents", moduleName: "agents", manifest: "agent.yaml", ids: manifest.Includes.Agents},
		{moduleDir: "tools-mcp", moduleName: "tools", manifest: "tool.yaml", ids: manifest.Includes.Tools},
	}
	for _, set := range sets {
		if len(set.ids) == 0 {
			continue
		}
		target, err := ResolvePluginDependencyTarget(runtimeName, set.moduleName, pluginTargetRoot)
		if err != nil {
			return nil, err
		}
		for _, id := range set.ids {
			if err := validatePackageID(id); err != nil {
				return nil, fmt.Errorf("%s dependency: %w", set.moduleName, err)
			}
			sourceDir := filepath.Join(extractRoot, "bundled", set.moduleDir, filepath.FromSlash(id))
			dependency, err := registry.ValidatePackageManifest(filepath.Join(sourceDir, set.manifest))
			if err != nil {
				return nil, fmt.Errorf("validate %s dependency %s: %w", set.moduleName, id, err)
			}
			if dependency.ID != id {
				return nil, fmt.Errorf("%s dependency path %s contains id %s", set.moduleName, id, dependency.ID)
			}
			if !supportsRuntime(dependency.Runtimes, runtimeName) {
				return nil, fmt.Errorf("%s dependency %s does not declare runtime %s", set.moduleName, id, runtimeName)
			}
			if err := validateExecutionCompatibility(dependency, runtimeName, executionRuntimes); err != nil {
				return nil, fmt.Errorf("%s dependency %s: %w", set.moduleName, id, err)
			}
			if err := ValidateOperationalInstall(registry.ProjectManifest(dependency)); err != nil {
				return nil, fmt.Errorf("%s dependency %s: %w", set.moduleName, id, err)
			}
			plans = append(plans, installTreePlan{SourceDir: sourceDir, TargetRoot: target.TargetPath, ID: id})
		}
	}
	return plans, nil
}

func validateExecutionCompatibility(manifest registry.Manifest, catalogRuntime string, explicitRuntimes []string) error {
	if !strings.HasPrefix(manifest.SchemaVersion, "2.") || !requiresExecutionCompatibility(manifest) {
		return nil
	}
	hostPlatform, err := currentHostPlatform()
	if err != nil {
		return err
	}
	if !containsNormalized(manifest.Execution.SupportedPlatforms, hostPlatform) {
		return fmt.Errorf("%s execution kind %s does not support host platform %s (declared: %s)", manifest.ID, manifest.Execution.Kind, hostPlatform, strings.Join(manifest.Execution.SupportedPlatforms, ", "))
	}

	available := make([]string, 0, 2+len(explicitRuntimes))
	available = append(available, strings.TrimSpace(catalogRuntime), "native")
	available = append(available, explicitRuntimes...)
	for _, declared := range manifest.Execution.SupportedRuntimes {
		if containsNormalized(available, declared) {
			return nil
		}
	}
	return fmt.Errorf("%s execution kind %s requires one of execution runtimes [%s], but selected environment provides [%s]; add a matching --execution-runtime", manifest.ID, manifest.Execution.Kind, strings.Join(manifest.Execution.SupportedRuntimes, ", "), strings.Join(normalizedUnique(available), ", "))
}

func requiresExecutionCompatibility(manifest registry.Manifest) bool {
	if len(manifest.Usability.ExecutableHelpers) > 0 {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(manifest.Execution.Kind)) {
	case "instructions", "integration-template", "":
		return false
	default:
		return true
	}
}

func currentHostPlatform() (string, error) {
	switch goruntime.GOOS {
	case "darwin":
		return "macos", nil
	case "linux", "windows":
		return goruntime.GOOS, nil
	default:
		return "", fmt.Errorf("unsupported host operating system %s", goruntime.GOOS)
	}
}

func containsNormalized(values []string, wanted string) bool {
	wanted = strings.ToLower(strings.TrimSpace(wanted))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == wanted {
			return true
		}
	}
	return false
}

func normalizedUnique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	sort.Strings(result)
	return result
}

func atomicInstallClosure(plans []installTreePlan, pluginTargetRoot, pluginID string, force bool) (string, error) {
	if len(plans) == 0 {
		return "", errors.New("empty plugin install closure")
	}
	runtimeRoot, err := filepath.Abs(filepath.Dir(pluginTargetRoot))
	if err != nil {
		return "", fmt.Errorf("resolve runtime root: %w", err)
	}
	markerPath := closureTransactionMarker(runtimeRoot, pluginID)

	type preparedPlan struct {
		plan        installTreePlan
		destination string
		parent      string
		hadPrevious bool
	}
	prepared := make([]preparedPlan, 0, len(plans))
	seenDestinations := make(map[string]struct{}, len(plans))
	for _, plan := range plans {
		if err := validatePackageID(plan.ID); err != nil {
			return "", err
		}
		targetRoot, err := filepath.Abs(plan.TargetRoot)
		if err != nil {
			return "", fmt.Errorf("resolve dependency target: %w", err)
		}
		if filepath.Dir(targetRoot) != runtimeRoot {
			return "", fmt.Errorf("dependency target %s is outside runtime root %s", targetRoot, runtimeRoot)
		}
		destination := filepath.Join(targetRoot, filepath.FromSlash(plan.ID))
		if _, exists := seenDestinations[destination]; exists {
			return "", fmt.Errorf("duplicate plugin closure destination %s", destination)
		}
		seenDestinations[destination] = struct{}{}
		prepared = append(prepared, preparedPlan{plan: plan, destination: destination, parent: filepath.Dir(destination)})
	}
	lockOrder := append([]preparedPlan(nil), prepared...)
	sort.Slice(lockOrder, func(left, right int) bool { return lockOrder[left].destination < lockOrder[right].destination })

	locks := make([]*flock.Flock, 0, len(prepared))
	defer func() {
		for index := len(locks) - 1; index >= 0; index-- {
			_ = locks[index].Unlock()
		}
	}()
	for _, item := range lockOrder {
		if err := os.MkdirAll(item.parent, 0o755); err != nil {
			return "", fmt.Errorf("create destination parent: %w", err)
		}
		lock := flock.New(filepath.Join(item.parent, "."+filepath.Base(item.destination)+".skills-hub.lock"))
		locked, err := lock.TryLock()
		if err != nil {
			return "", fmt.Errorf("acquire closure transaction lock: %w", err)
		}
		if !locked {
			return "", fmt.Errorf("another installation is updating %s; retry after it completes", item.destination)
		}
		locks = append(locks, lock)
	}
	if err := recoverClosureInstallTransaction(markerPath, runtimeRoot); err != nil {
		return "", fmt.Errorf("recover interrupted plugin closure: %w", err)
	}
	for index := range prepared {
		exists, err := pathExists(prepared[index].destination)
		if err != nil {
			return "", err
		}
		if exists && !force {
			return "", fmt.Errorf("destination already exists: %s (use --force to reinstall or upgrade)", prepared[index].destination)
		}
		prepared[index].hadPrevious = exists
	}
	if len(prepared) > 1 && prepared[0].hadPrevious {
		return "", errors.New("in-place upgrade of an installed plugin dependency closure is not supported without a runtime-wide atomic activation pointer; install into a clean runtime or keep the current closure")
	}

	transaction := closureInstallTransaction{Phase: "prepared", Operations: make([]closureInstallOperation, 0, len(prepared))}
	defer func() {
		for _, operation := range transaction.Operations {
			_ = os.RemoveAll(operation.Stage)
		}
	}()
	for _, item := range prepared {
		stage, err := os.MkdirTemp(item.parent, ".skills-hub-stage-")
		if err != nil {
			return "", fmt.Errorf("create closure staging directory: %w", err)
		}
		if err := copyTree(item.plan.SourceDir, stage); err != nil {
			_ = os.RemoveAll(stage)
			return "", fmt.Errorf("stage verified closure member %s: %w", item.plan.ID, err)
		}
		operation := closureInstallOperation{Destination: item.destination, Stage: stage, HadPrevious: item.hadPrevious}
		if item.hadPrevious {
			backup, err := os.MkdirTemp(item.parent, ".skills-hub-backup-")
			if err != nil {
				_ = os.RemoveAll(stage)
				return "", fmt.Errorf("create closure rollback path: %w", err)
			}
			if err := os.Remove(backup); err != nil {
				_ = os.RemoveAll(stage)
				return "", fmt.Errorf("prepare closure rollback path: %w", err)
			}
			operation.Backup = backup
		}
		transaction.Operations = append(transaction.Operations, operation)
	}
	if err := writeClosureInstallTransaction(markerPath, transaction); err != nil {
		return "", fmt.Errorf("persist plugin closure transaction: %w", err)
	}
	rollback := func(commitErr error) (string, error) {
		if recoveryErr := recoverClosureInstallTransaction(markerPath, runtimeRoot); recoveryErr != nil {
			return "", fmt.Errorf("%w; closure rollback failed: %v", commitErr, recoveryErr)
		}
		return "", commitErr
	}
	pluginOperation := transaction.Operations[0]
	if pluginOperation.HadPrevious {
		if err := os.Rename(pluginOperation.Destination, pluginOperation.Backup); err != nil {
			return rollback(fmt.Errorf("deactivate existing plugin %s: %w", pluginOperation.Destination, err))
		}
		if err := syncDirectory(filepath.Dir(pluginOperation.Destination)); err != nil {
			return rollback(fmt.Errorf("persist plugin deactivation %s: %w", pluginOperation.Destination, err))
		}
		if closureInstallTransactionFault != nil {
			if err := closureInstallTransactionFault("after-plugin-deactivation", 0); err != nil {
				return rollback(err)
			}
		}
	}
	activationOrder := make([]int, 0, len(transaction.Operations))
	for index := 1; index < len(transaction.Operations); index++ {
		activationOrder = append(activationOrder, index)
	}
	activationOrder = append(activationOrder, 0)
	for _, index := range activationOrder {
		operation := transaction.Operations[index]
		parent := filepath.Dir(operation.Destination)
		if operation.HadPrevious && index != 0 {
			if err := os.Rename(operation.Destination, operation.Backup); err != nil {
				return rollback(fmt.Errorf("preserve existing closure member %s: %w", operation.Destination, err))
			}
			if err := syncDirectory(parent); err != nil {
				return rollback(fmt.Errorf("persist preserved closure member %s: %w", operation.Destination, err))
			}
		}
		if closureInstallTransactionFault != nil {
			if err := closureInstallTransactionFault("before-activation", index); err != nil {
				return rollback(err)
			}
		}
		if err := os.Rename(operation.Stage, operation.Destination); err != nil {
			return rollback(fmt.Errorf("activate closure member %s: %w", operation.Destination, err))
		}
		if err := syncDirectory(parent); err != nil {
			return rollback(fmt.Errorf("persist activated closure member %s: %w", operation.Destination, err))
		}
		if closureInstallTransactionFault != nil {
			if err := closureInstallTransactionFault("after-activation", index); err != nil {
				return rollback(err)
			}
		}
	}
	transaction.Phase = "committed"
	if err := writeClosureInstallTransaction(markerPath, transaction); err != nil {
		return rollback(fmt.Errorf("persist committed plugin closure: %w", err))
	}
	if err := recoverClosureInstallTransaction(markerPath, runtimeRoot); err != nil {
		return "", fmt.Errorf("finalize plugin closure transaction: %w", err)
	}
	return filepath.Join(pluginTargetRoot, filepath.FromSlash(pluginID)), nil
}

func closureTransactionMarker(runtimeRoot, pluginID string) string {
	return filepath.Join(runtimeRoot, ".skills-hub-closure-"+cacheKey(pluginID)[:16]+".json")
}

func writeClosureInstallTransaction(markerPath string, transaction closureInstallTransaction) error {
	parent := filepath.Dir(markerPath)
	temporary, err := os.CreateTemp(parent, ".closure-transaction-")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := json.NewEncoder(temporary).Encode(transaction); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, markerPath); err != nil {
		return err
	}
	return syncDirectory(parent)
}

func recoverClosureInstallTransaction(markerPath, runtimeRoot string) error {
	data, err := os.ReadFile(markerPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var transaction closureInstallTransaction
	if err := json.Unmarshal(data, &transaction); err != nil {
		return fmt.Errorf("decode closure transaction marker %s: %w", markerPath, err)
	}
	if transaction.Phase != "prepared" && transaction.Phase != "committed" {
		return fmt.Errorf("closure transaction marker %s has invalid phase %q", markerPath, transaction.Phase)
	}
	if len(transaction.Operations) == 0 {
		return fmt.Errorf("closure transaction marker %s has no operations", markerPath)
	}
	for _, operation := range transaction.Operations {
		if err := validateClosureOperation(runtimeRoot, operation); err != nil {
			return fmt.Errorf("closure transaction marker %s: %w", markerPath, err)
		}
	}
	if transaction.Phase == "prepared" {
		plugin := transaction.Operations[0]
		pluginExists, err := pathExists(plugin.Destination)
		if err != nil {
			return err
		}
		backupExists := false
		if plugin.Backup != "" {
			backupExists, err = pathExists(plugin.Backup)
			if err != nil {
				return err
			}
		}
		// If a new plugin was activated, remove it before restoring any old
		// dependency. If the old plugin has not yet been deactivated, its backup
		// does not exist and it is already consistent with the old dependencies.
		if pluginExists && (!plugin.HadPrevious || backupExists) {
			if err := os.RemoveAll(plugin.Destination); err != nil {
				return fmt.Errorf("deactivate plugin before closure rollback: %w", err)
			}
			if err := syncDirectory(filepath.Dir(plugin.Destination)); err != nil {
				return err
			}
		}
	}
	for index := len(transaction.Operations) - 1; index >= 0; index-- {
		operation := transaction.Operations[index]
		destinationExists, err := pathExists(operation.Destination)
		if err != nil {
			return err
		}
		backupExists := false
		if operation.Backup != "" {
			backupExists, err = pathExists(operation.Backup)
			if err != nil {
				return err
			}
		}
		if transaction.Phase == "prepared" {
			if operation.HadPrevious && backupExists {
				if destinationExists {
					if err := os.RemoveAll(operation.Destination); err != nil {
						return err
					}
				}
				if err := os.Rename(operation.Backup, operation.Destination); err != nil {
					return fmt.Errorf("restore closure member %s: %w", operation.Destination, err)
				}
			} else if !operation.HadPrevious && destinationExists {
				if err := os.RemoveAll(operation.Destination); err != nil {
					return err
				}
			} else if operation.HadPrevious && !destinationExists {
				return fmt.Errorf("closure member %s has neither active content nor a rollback copy", operation.Destination)
			}
		} else {
			if !destinationExists {
				return fmt.Errorf("committed closure member %s is missing", operation.Destination)
			}
			if backupExists {
				if err := os.RemoveAll(operation.Backup); err != nil {
					return err
				}
			}
		}
		if err := os.RemoveAll(operation.Stage); err != nil {
			return err
		}
		if err := syncDirectory(filepath.Dir(operation.Destination)); err != nil {
			return err
		}
	}
	if err := os.Remove(markerPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return syncDirectory(runtimeRoot)
}

func validateClosureOperation(runtimeRoot string, operation closureInstallOperation) error {
	relative, err := filepath.Rel(runtimeRoot, operation.Destination)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("destination %s is outside runtime root", operation.Destination)
	}
	parent := filepath.Dir(operation.Destination)
	if filepath.Dir(operation.Stage) != parent || !strings.HasPrefix(filepath.Base(operation.Stage), ".skills-hub-stage-") {
		return fmt.Errorf("stage %s is outside its package transaction", operation.Stage)
	}
	if operation.HadPrevious {
		if filepath.Dir(operation.Backup) != parent || !strings.HasPrefix(filepath.Base(operation.Backup), ".skills-hub-backup-") {
			return fmt.Errorf("backup %s is outside its package transaction", operation.Backup)
		}
	} else if operation.Backup != "" {
		return fmt.Errorf("new closure member unexpectedly declares backup %s", operation.Backup)
	}
	return nil
}

func recoverInterruptedClosureInstalls(runtimeRoot string) error {
	entries, err := os.ReadDir(runtimeRoot)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), ".skills-hub-closure-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		markerPath := filepath.Join(runtimeRoot, entry.Name())
		data, err := os.ReadFile(markerPath)
		if err != nil {
			return err
		}
		var transaction closureInstallTransaction
		if err := json.Unmarshal(data, &transaction); err != nil {
			return fmt.Errorf("decode closure transaction marker %s: %w", markerPath, err)
		}
		locks := make([]*flock.Flock, 0, len(transaction.Operations))
		var recoveryErr error
		for _, operation := range transaction.Operations {
			if err := validateClosureOperation(runtimeRoot, operation); err != nil {
				return fmt.Errorf("closure transaction marker %s: %w", markerPath, err)
			}
			lock := flock.New(filepath.Join(filepath.Dir(operation.Destination), "."+filepath.Base(operation.Destination)+".skills-hub.lock"))
			locked, err := lock.TryLock()
			if err != nil {
				recoveryErr = err
				break
			}
			if !locked {
				recoveryErr = fmt.Errorf("closure transaction %s is busy; retry after the active installer releases all recorded package locks", markerPath)
				break
			}
			locks = append(locks, lock)
		}
		if recoveryErr == nil {
			recoveryErr = recoverClosureInstallTransaction(markerPath, runtimeRoot)
		}
		for index := len(locks) - 1; index >= 0; index-- {
			_ = locks[index].Unlock()
		}
		if recoveryErr != nil {
			return recoveryErr
		}
	}
	return nil
}

func atomicInstallTree(sourceDir, targetRoot, id string, force bool) (string, error) {
	destination := filepath.Join(targetRoot, filepath.FromSlash(id))
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", fmt.Errorf("create destination parent: %w", err)
	}
	lock := flock.New(filepath.Join(parent, "."+filepath.Base(destination)+".skills-hub.lock"))
	locked, err := lock.TryLock()
	if err != nil {
		return "", fmt.Errorf("acquire install transaction lock: %w", err)
	}
	if !locked {
		return "", fmt.Errorf("another installation is updating %s; retry after it completes", destination)
	}
	defer lock.Unlock()
	markerPath := filepath.Join(parent, "."+filepath.Base(destination)+".skills-hub-transaction.json")
	if err := recoverInstallTransaction(markerPath, destination); err != nil {
		return "", fmt.Errorf("recover interrupted installation: %w", err)
	}
	if _, err := os.Stat(destination); err == nil && !force {
		return "", fmt.Errorf("destination already exists: %s (use --force to reinstall or upgrade)", destination)
	} else if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect destination %s: %w", destination, err)
	}
	stage, err := os.MkdirTemp(parent, ".skills-hub-stage-")
	if err != nil {
		return "", fmt.Errorf("create install staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	if err := copyTree(sourceDir, stage); err != nil {
		return "", fmt.Errorf("stage verified package: %w", err)
	}

	if _, err := os.Stat(destination); os.IsNotExist(err) {
		if err := os.Rename(stage, destination); err != nil {
			return "", fmt.Errorf("activate staged package: %w", err)
		}
		return destination, nil
	}
	backup, err := os.MkdirTemp(parent, ".skills-hub-backup-")
	if err != nil {
		return "", fmt.Errorf("create rollback path: %w", err)
	}
	if err := os.Remove(backup); err != nil {
		return "", fmt.Errorf("prepare rollback path: %w", err)
	}
	transaction := installTransaction{Destination: destination, Stage: stage, Backup: backup}
	if err := writeInstallTransaction(markerPath, transaction); err != nil {
		return "", fmt.Errorf("persist install transaction: %w", err)
	}
	if err := os.Rename(destination, backup); err != nil {
		_ = os.Remove(markerPath)
		return "", fmt.Errorf("preserve existing installation: %w", err)
	}
	if err := syncDirectory(parent); err != nil {
		return "", fmt.Errorf("persist preserved installation: %w", err)
	}
	if installTransactionFault != nil {
		if err := installTransactionFault("after-backup-rename"); err != nil {
			return "", err
		}
	}
	if err := os.Rename(stage, destination); err != nil {
		rollbackErr := os.Rename(backup, destination)
		if rollbackErr != nil {
			return "", fmt.Errorf("activate staged package: %w; rollback also failed: %v; recover %s", err, rollbackErr, backup)
		}
		_ = os.Remove(markerPath)
		_ = syncDirectory(parent)
		return "", fmt.Errorf("activate staged package: %w; previous installation restored", err)
	}
	if installTransactionFault != nil {
		if err := installTransactionFault("after-activation-rename"); err != nil {
			return "", err
		}
	}
	if err := syncDirectory(parent); err != nil {
		return "", fmt.Errorf("persist activated package: %w", err)
	}
	if err := os.RemoveAll(backup); err == nil {
		_ = os.Remove(markerPath)
		_ = syncDirectory(parent)
	}
	return destination, nil
}

func writeInstallTransaction(markerPath string, transaction installTransaction) error {
	parent := filepath.Dir(markerPath)
	temporary, err := os.CreateTemp(parent, ".install-transaction-")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	encoder := json.NewEncoder(temporary)
	if err := encoder.Encode(transaction); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, markerPath); err != nil {
		return err
	}
	return syncDirectory(parent)
}

func recoverInstallTransaction(markerPath, destination string) error {
	data, err := os.ReadFile(markerPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var transaction installTransaction
	if err := json.Unmarshal(data, &transaction); err != nil {
		return fmt.Errorf("decode transaction marker %s: %w", markerPath, err)
	}
	parent := filepath.Dir(destination)
	if transaction.Destination != destination || filepath.Dir(transaction.Stage) != parent || filepath.Dir(transaction.Backup) != parent ||
		!strings.HasPrefix(filepath.Base(transaction.Stage), ".skills-hub-stage-") ||
		!strings.HasPrefix(filepath.Base(transaction.Backup), ".skills-hub-backup-") {
		return fmt.Errorf("transaction marker %s contains paths outside the package transaction", markerPath)
	}
	destinationExists, err := pathExists(destination)
	if err != nil {
		return err
	}
	backupExists, err := pathExists(transaction.Backup)
	if err != nil {
		return err
	}
	if !destinationExists {
		if !backupExists {
			return fmt.Errorf("transaction marker %s has neither an active package nor a recoverable backup", markerPath)
		}
		if err := os.Rename(transaction.Backup, destination); err != nil {
			return fmt.Errorf("restore %s: %w", transaction.Backup, err)
		}
	} else if backupExists {
		if err := os.RemoveAll(transaction.Backup); err != nil {
			return fmt.Errorf("remove completed transaction backup %s: %w", transaction.Backup, err)
		}
	}
	_ = os.RemoveAll(transaction.Stage)
	if err := os.Remove(markerPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return syncDirectory(parent)
}

func recoverInterruptedInstalls(targetRoot string) error {
	if _, err := os.Stat(targetRoot); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	markers := make([]string, 0)
	if err := filepath.WalkDir(targetRoot, func(markerPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), ".") || !strings.HasSuffix(entry.Name(), ".skills-hub-transaction.json") {
			return nil
		}
		markers = append(markers, markerPath)
		return nil
	}); err != nil {
		return err
	}
	for _, markerPath := range markers {
		name := filepath.Base(markerPath)
		packageName := strings.TrimSuffix(strings.TrimPrefix(name, "."), ".skills-hub-transaction.json")
		if packageName == "" || filepath.Base(packageName) != packageName {
			return fmt.Errorf("invalid install transaction marker %s", markerPath)
		}
		destination := filepath.Join(filepath.Dir(markerPath), packageName)
		lock := flock.New(filepath.Join(filepath.Dir(markerPath), "."+packageName+".skills-hub.lock"))
		locked, err := lock.TryLock()
		if err != nil {
			return err
		}
		if !locked {
			continue
		}
		recoveryErr := recoverInstallTransaction(markerPath, destination)
		unlockErr := lock.Unlock()
		if recoveryErr != nil {
			return recoveryErr
		}
		if unlockErr != nil {
			return unlockErr
		}
	}
	return nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("inspect transaction path %s: %w", path, err)
}
