package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// MarketplaceActionMeta holds the metadata returned by the Shyntr Marketplace API.
type MarketplaceActionMeta struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	// StorageType: "git", "s3", "oci"
	StorageType string `json:"storage_type"`
	// CloneURL — for git storage
	CloneURL string `json:"clone_url"`
	// DownloadURL — for s3/oci storage (action bundle URL)
	DownloadURL string `json:"download_url"`
	// Token — auth token for private registry (omitted when empty)
	Token string `json:"token,omitempty"`
}

// MarketplaceResolver resolves actions from Shyntr Marketplace before falling
// back to GitHub.
type MarketplaceResolver struct {
	baseURL    string
	token      string
	namespaces []string
	httpClient *http.Client
}

// NewMarketplaceResolver creates a MarketplaceResolver.
func NewMarketplaceResolver(baseURL, token string, namespaces []string) *MarketplaceResolver {
	return &MarketplaceResolver{
		baseURL:    baseURL,
		token:      token,
		namespaces: namespaces,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// ShouldResolve reports whether this action should be looked up in the
// marketplace.  If the namespace list is empty every action is attempted;
// otherwise only those whose org matches a configured namespace.
func (r *MarketplaceResolver) ShouldResolve(org, _ string) bool {
	if r == nil || r.baseURL == "" {
		return false
	}
	if len(r.namespaces) == 0 {
		return true
	}
	for _, ns := range r.namespaces {
		if strings.EqualFold(org, ns) {
			return true
		}
	}
	return false
}

// Resolve fetches action metadata from the marketplace API.
//
//	GET {baseURL}/api/v1/marketplace/actions/{namespace}/{name}/versions/{version}
//
// Returns (nil, nil) when the action is not found (HTTP 404) — the caller
// should silently fall back to GitHub.  Any other non-200 status or network
// error is returned as an error — the caller should log a warning and fall
// back to GitHub.
func (r *MarketplaceResolver) Resolve(ctx context.Context, org, repo, ref string) (*MarketplaceActionMeta, error) {
	url := fmt.Sprintf("%s/api/v1/marketplace/actions/%s/%s/versions/%s",
		r.baseURL, org, repo, ref)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("marketplace unreachable: %w", err)
	}
	defer resp.Body.Close()

	// 404 → not in marketplace; caller should fall back to GitHub silently.
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace returned %d", resp.StatusCode)
	}

	var meta MarketplaceActionMeta
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("failed to decode marketplace response: %w", err)
	}

	return &meta, nil
}
