package skills

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

const (
	defaultClawHubTimeout  = 30 * time.Second
	defaultMaxZipSize      = 50 * 1024 * 1024 // 50 MB
	defaultMaxResponseSize = 2 * 1024 * 1024  // 2 MB
)

// skillIdentifierRegex validates skill slugs/names
var skillIdentifierRegex = regexp.MustCompile(`^[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*$`)

// ValidateSkillIdentifier validates a skill identifier (slug/name)
func ValidateSkillIdentifier(slug string) error {
	if slug == "" {
		return fmt.Errorf("skill identifier cannot be empty")
	}
	if len(slug) > 64 {
		return fmt.Errorf("skill identifier exceeds 64 characters")
	}
	if !skillIdentifierRegex.MatchString(slug) {
		return fmt.Errorf("skill identifier must be alphanumeric with optional hyphens")
	}
	return nil
}

// ClawHubRegistry implements SkillRegistry for the ClawHub platform.
type ClawHubRegistry struct {
	baseURL         string
	authToken       string
	searchPath      string
	skillsPath      string
	downloadPath    string
	maxZipSize      int
	maxResponseSize int
	client          *http.Client
}

// NewClawHubRegistry creates a new ClawHub registry client from config.
func NewClawHubRegistry(cfg ClawHubConfig) *ClawHubRegistry {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://clawhub.ai"
	}
	searchPath := cfg.SearchPath
	if searchPath == "" {
		searchPath = "/api/v1/search"
	}
	skillsPath := cfg.SkillsPath
	if skillsPath == "" {
		skillsPath = "/api/v1/skills"
	}
	downloadPath := cfg.DownloadPath
	if downloadPath == "" {
		downloadPath = "/api/v1/download"
	}

	timeout := defaultClawHubTimeout
	if cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Second
	}

	maxZip := defaultMaxZipSize
	if cfg.MaxZipSize > 0 {
		maxZip = cfg.MaxZipSize
	}

	maxResp := defaultMaxResponseSize
	if cfg.MaxResponseSize > 0 {
		maxResp = cfg.MaxResponseSize
	}

	return &ClawHubRegistry{
		baseURL:         baseURL,
		authToken:       cfg.AuthToken,
		searchPath:      searchPath,
		skillsPath:      skillsPath,
		downloadPath:    downloadPath,
		maxZipSize:      maxZip,
		maxResponseSize: maxResp,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        5,
				IdleConnTimeout:     30 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
	}
}

func (c *ClawHubRegistry) Name() string {
	return "clawhub"
}

// --- Search ---

type clawhubSearchResponse struct {
	Results []clawhubSearchResult `json:"results"`
}

type clawhubSearchResult struct {
	Score       float64 `json:"score"`
	Slug        *string `json:"slug"`
	DisplayName *string `json:"displayName"`
	Summary     *string `json:"summary"`
	Version     *string `json:"version"`
}

func derefStr(ptr *string, def string) string {
	if ptr == nil {
		return def
	}
	return *ptr
}

func (c *ClawHubRegistry) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	u, err := url.Parse(c.baseURL + c.searchPath)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	q.Set("q", query)
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	u.RawQuery = q.Encode()

	body, err := c.doGet(ctx, u.String())
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	var resp clawhubSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	results := make([]SearchResult, 0, len(resp.Results))
	for _, r := range resp.Results {
		slug := derefStr(r.Slug, "")
		if slug == "" {
			continue
		}

		summary := derefStr(r.Summary, "")
		if summary == "" {
			continue
		}

		displayName := derefStr(r.DisplayName, "")
		if displayName == "" {
			displayName = slug
		}

		results = append(results, SearchResult{
			Score:        r.Score,
			Slug:         slug,
			DisplayName:  displayName,
			Summary:      summary,
			Version:      derefStr(r.Version, ""),
			RegistryName: c.Name(),
			Name:         slug,
			Description:  summary,
			Source:       c.Name(),
		})
	}

	return results, nil
}

// --- GetSkillMeta ---

type clawhubSkillResponse struct {
	Slug          string                 `json:"slug"`
	DisplayName   string                 `json:"displayName"`
	Summary       string                 `json:"summary"`
	LatestVersion *clawhubVersionInfo    `json:"latestVersion"`
	Moderation    *clawhubModerationInfo `json:"moderation"`
}

type clawhubVersionInfo struct {
	Version string `json:"version"`
}

type clawhubModerationInfo struct {
	IsMalwareBlocked bool `json:"isMalwareBlocked"`
	IsSuspicious     bool `json:"isSuspicious"`
}

func (c *ClawHubRegistry) GetSkillMeta(ctx context.Context, slug string) (*SkillMeta, error) {
	if err := ValidateSkillIdentifier(slug); err != nil {
		return nil, fmt.Errorf("invalid slug %q: %w", slug, err)
	}

	u := c.baseURL + c.skillsPath + "/" + url.PathEscape(slug)

	body, err := c.doGet(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("skill metadata request failed: %w", err)
	}

	var resp clawhubSkillResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse skill metadata: %w", err)
	}

	meta := &SkillMeta{
		Slug:         resp.Slug,
		DisplayName:  resp.DisplayName,
		Summary:      resp.Summary,
		RegistryName: c.Name(),
	}

	if resp.LatestVersion != nil {
		meta.LatestVersion = resp.LatestVersion.Version
	}
	if resp.Moderation != nil {
		meta.IsMalwareBlocked = resp.Moderation.IsMalwareBlocked
		meta.IsSuspicious = resp.Moderation.IsSuspicious
	}

	return meta, nil
}

// --- DownloadAndInstall ---

// DownloadAndInstall fetches metadata (with fallback), resolves version,
// downloads the skill ZIP, and extracts it to targetDir.
func (c *ClawHubRegistry) DownloadAndInstall(ctx context.Context, slug, version, targetDir string) (*InstallResult, error) {
	if err := ValidateSkillIdentifier(slug); err != nil {
		return nil, fmt.Errorf("invalid slug %q: %w", slug, err)
	}

	// Step 1: Fetch metadata (with fallback).
	result := &InstallResult{}
	meta, err := c.GetSkillMeta(ctx, slug)
	if err != nil {
		// Fallback: proceed without metadata.
		meta = nil
	}

	if meta != nil {
		result.IsMalwareBlocked = meta.IsMalwareBlocked
		result.IsSuspicious = meta.IsSuspicious
		result.Summary = meta.Summary
	}

	// Step 2: Resolve version.
	installVersion := version
	if installVersion == "" && meta != nil {
		installVersion = meta.LatestVersion
	}
	if installVersion == "" {
		installVersion = "latest"
	}
	result.Version = installVersion

	// Step 3: Download ZIP to temp file.
	u, err := url.Parse(c.baseURL + c.downloadPath)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	q.Set("slug", slug)
	if installVersion != "latest" {
		q.Set("version", installVersion)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	tmpPath, err := c.downloadToFile(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpPath)

	// Step 4: Extract from file on disk.
	if err := extractZipFile(tmpPath, targetDir); err != nil {
		return nil, err
	}

	return result, nil
}

// downloadToFile downloads a file to a temp location, respecting maxZipSize.
func (c *ClawHubRegistry) downloadToFile(ctx context.Context, req *http.Request) (string, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "wunderpus-skill-*.zip")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()

	// Limit download size
	limitedReader := io.LimitReader(resp.Body, int64(c.maxZipSize))
	written, err := io.Copy(tmpFile, limitedReader)
	tmpFile.Close()

	if err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	if written >= int64(c.maxZipSize) {
		os.Remove(tmpPath)
		return "", fmt.Errorf("download exceeds maximum size of %d bytes", c.maxZipSize)
	}

	return tmpPath, nil
}

// extractZipFile extracts a ZIP file to targetDir.
func extractZipFile(zipPath, targetDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		// Prevent zip slip vulnerability
		relPath := filepath.Clean(f.Name)
		if filepath.IsAbs(relPath) {
			return fmt.Errorf("invalid file path in ZIP: %s", f.Name)
		}
		if relPath == ".." || len(relPath) > 200 {
			return fmt.Errorf("suspicious file path in ZIP: %s", f.Name)
		}

		targetPath := filepath.Join(targetDir, relPath)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		// Create parent directory if needed
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode())
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", f.Name, err)
		}

		inFile, err := f.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to open file in ZIP %s: %w", f.Name, err)
		}

		_, err = io.Copy(outFile, inFile)
		inFile.Close()
		outFile.Close()

		if err != nil {
			return fmt.Errorf("failed to extract file %s: %w", f.Name, err)
		}
	}

	return nil
}

// --- HTTP helper ---

func (c *ClawHubRegistry) doGet(ctx context.Context, urlStr string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Limit response body read to prevent memory issues.
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(c.maxResponseSize)))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
