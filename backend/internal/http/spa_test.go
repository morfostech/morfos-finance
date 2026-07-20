package http

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSPAHandlerServesAssetsAndRouteFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<main>app</main>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("window.app = true"), 0o644); err != nil {
		t.Fatal(err)
	}

	handler := newSPAHandler(dir)

	assetResponse := httptest.NewRecorder()
	handler.ServeHTTP(assetResponse, httptest.NewRequest("GET", "/assets/app.js", nil))
	if got := assetResponse.Body.String(); got != "window.app = true" {
		t.Fatalf("asset body = %q", got)
	}

	routeResponse := httptest.NewRecorder()
	handler.ServeHTTP(routeResponse, httptest.NewRequest("GET", "/projects/42", nil))
	if got := routeResponse.Body.String(); got != "<main>app</main>" {
		t.Fatalf("fallback body = %q", got)
	}
}
