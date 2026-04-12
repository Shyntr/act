package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ── ShouldResolve ────────────────────────────────────────────────────────────

func TestMarketplaceResolver_ShouldResolve(t *testing.T) {
	t.Run("nil resolver returns false", func(t *testing.T) {
		var r *MarketplaceResolver
		assert.False(t, r.ShouldResolve("marketplace", "docker-build"))
	})

	t.Run("empty baseURL returns false", func(t *testing.T) {
		r := &MarketplaceResolver{baseURL: ""}
		assert.False(t, r.ShouldResolve("marketplace", "docker-build"))
	})

	t.Run("empty namespace list resolves everything", func(t *testing.T) {
		r := NewMarketplaceResolver("https://marketplace.shyntr.com", "", nil)
		assert.True(t, r.ShouldResolve("any-org", "any-repo"))
		assert.True(t, r.ShouldResolve("marketplace", "docker-build"))
	})

	t.Run("namespace match (case-insensitive) returns true", func(t *testing.T) {
		r := NewMarketplaceResolver("https://marketplace.shyntr.com", "",
			[]string{"marketplace", "shyntr-actions"})
		assert.True(t, r.ShouldResolve("Marketplace", "docker-build"))
		assert.True(t, r.ShouldResolve("SHYNTR-ACTIONS", "something"))
	})

	t.Run("no namespace match returns false", func(t *testing.T) {
		r := NewMarketplaceResolver("https://marketplace.shyntr.com", "",
			[]string{"marketplace"})
		assert.False(t, r.ShouldResolve("actions", "checkout"))
	})
}

// ── Resolve ──────────────────────────────────────────────────────────────────

func TestMarketplaceResolver_Resolve_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	r := NewMarketplaceResolver(srv.URL, "", nil)
	meta, err := r.Resolve(context.Background(), "marketplace", "docker-build", "v1")
	assert.NoError(t, err)
	assert.Nil(t, meta, "404 should return nil meta for silent GitHub fallback")
}

func TestMarketplaceResolver_Resolve_Unreachable(t *testing.T) {
	// Point at a port that is not listening.
	r := NewMarketplaceResolver("http://127.0.0.1:1", "", nil)
	meta, err := r.Resolve(context.Background(), "marketplace", "docker-build", "v1")
	assert.Error(t, err, "unreachable server should return an error")
	assert.Nil(t, meta)
}

func TestMarketplaceResolver_Resolve_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	r := NewMarketplaceResolver(srv.URL, "", nil)
	_, err := r.Resolve(context.Background(), "marketplace", "docker-build", "v1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestMarketplaceResolver_Resolve_Success(t *testing.T) {
	want := MarketplaceActionMeta{
		Namespace:   "marketplace",
		Name:        "docker-build",
		Version:     "v1",
		StorageType: "git",
		CloneURL:    "https://git.shyntr.com/marketplace/docker-build",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/marketplace/actions/marketplace/docker-build/versions/v1", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	resolver := NewMarketplaceResolver(srv.URL, "test-token", nil)
	got, err := resolver.Resolve(context.Background(), "marketplace", "docker-build", "v1")
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, want.Namespace, got.Namespace)
	assert.Equal(t, want.Name, got.Name)
	assert.Equal(t, want.StorageType, got.StorageType)
	assert.Equal(t, want.CloneURL, got.CloneURL)
}

func TestMarketplaceResolver_Resolve_NoAuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"), "no token should mean no auth header")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	r := NewMarketplaceResolver(srv.URL, "", nil)
	_, _ = r.Resolve(context.Background(), "org", "repo", "ref")
}

// ── Config lazy-init ─────────────────────────────────────────────────────────

func TestConfig_marketplaceResolver_disabled(t *testing.T) {
	c := &Config{}
	assert.Nil(t, c.marketplaceResolver(), "empty URL should return nil")
}

func TestConfig_marketplaceResolver_lazyInit(t *testing.T) {
	c := &Config{
		ShyntrMarketplaceURL:        "https://marketplace.shyntr.com",
		ShyntrMarketplaceToken:      "tok",
		ShyntrMarketplaceNamespaces: []string{"marketplace"},
	}
	r1 := c.marketplaceResolver()
	r2 := c.marketplaceResolver()
	assert.NotNil(t, r1)
	assert.Same(t, r1, r2, "should return the same instance on repeated calls")
}
