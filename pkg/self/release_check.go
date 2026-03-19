package self

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	selfOutput "github.com/joeblew999/utm-dev/pkg/self/output"
)

// CheckReleaseResult represents the result of checking a release
type CheckReleaseResult struct {
	Tag           string   `json:"tag"`
	Exists        bool     `json:"exists"`
	Published     bool     `json:"published"`
	PublishedAt   string   `json:"published_at,omitempty"`
	Assets        []string `json:"assets,omitempty"`
	ReleaseURL    string   `json:"release_url,omitempty"`
	WorkflowURL   string   `json:"workflow_url"`
	Message       string   `json:"message"`
}

// CheckRelease checks if a GitHub release exists and is ready
func CheckRelease(tag string) error {
	selfOutput.Run("self check-release", func() (*CheckReleaseResult, error) {
		result := &CheckReleaseResult{
			Tag:         tag,
			Exists:      false,
			Published:   false,
			WorkflowURL: fmt.Sprintf("https://github.com/%s/actions", FullRepoName),
		}

		// Try to get the release
		releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", FullRepoName, tag)
		
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(releaseURL)
		if err != nil {
			result.Message = fmt.Sprintf("Failed to check release: %v", err)
			return result, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode == 404 {
			result.Message = fmt.Sprintf("Release %s not found - workflow may still be running", tag)
			return result, nil
		}

		if resp.StatusCode != 200 {
			result.Message = fmt.Sprintf("GitHub API returned status %d", resp.StatusCode)
			return result, nil
		}

		// Parse release info
		var release struct {
			TagName     string    `json:"tag_name"`
			PublishedAt time.Time `json:"published_at"`
			HTMLURL     string    `json:"html_url"`
			Assets      []struct {
				Name string `json:"name"`
			} `json:"assets"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			result.Message = fmt.Sprintf("Failed to parse release info: %v", err)
			return result, nil
		}

		result.Exists = true
		result.Published = true
		result.PublishedAt = release.PublishedAt.Format(time.RFC3339)
		result.ReleaseURL = release.HTMLURL

		// List assets
		for _, asset := range release.Assets {
			result.Assets = append(result.Assets, asset.Name)
		}

		if len(result.Assets) == 0 {
			result.Message = "Release exists but has no assets yet - workflow may still be building"
		} else {
			result.Message = fmt.Sprintf("Release is ready with %d assets", len(result.Assets))
		}

		return result, nil
	})
	return nil
}
