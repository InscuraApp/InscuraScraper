package route

import (
	"encoding/json"
	"inscurascraper/engine"
	"inscurascraper/route/auth"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type denyAll struct{}

func (denyAll) Valid(string) bool { return false }

func newTestRouter(t *testing.T, v auth.Validator) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	app := engine.Default()
	return New(app, v)
}

func doRequest(t *testing.T, r http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestIndex(t *testing.T) {
	r := newTestRouter(t, denyAll{})
	w := doRequest(t, r, http.MethodGet, "/")
	if w.Code != http.StatusOK {
		t.Fatalf("index status = %d, want %d", w.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Errorf("response missing data field: %s", w.Body.String())
	}
}

func TestHealthz(t *testing.T) {
	r := newTestRouter(t, denyAll{})
	w := doRequest(t, r, http.MethodGet, "/healthz")
	if w.Code != http.StatusOK {
		t.Fatalf("/healthz status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"ok"`) {
		t.Errorf("/healthz body missing ok marker: %s", w.Body.String())
	}
}

func TestReadyz(t *testing.T) {
	r := newTestRouter(t, denyAll{})
	w := doRequest(t, r, http.MethodGet, "/readyz")
	// Default() uses in-memory SQLite; readiness must succeed.
	if w.Code != http.StatusOK {
		t.Fatalf("/readyz status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestNoRoute(t *testing.T) {
	r := newTestRouter(t, denyAll{})
	w := doRequest(t, r, http.MethodGet, "/does-not-exist")
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing route status = %d, want 404", w.Code)
	}
}

func TestNoMethod(t *testing.T) {
	r := newTestRouter(t, denyAll{})
	w := doRequest(t, r, http.MethodDelete, "/")
	if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
		// Gin returns 404 unless HandleMethodNotAllowed is explicitly enabled.
		t.Fatalf("unexpected status = %d for DELETE /", w.Code)
	}
}

func TestModulesEndpoint(t *testing.T) {
	r := newTestRouter(t, denyAll{})
	w := doRequest(t, r, http.MethodGet, "/v1/modules")
	if w.Code != http.StatusOK {
		t.Fatalf("/v1/modules status = %d, want 200", w.Code)
	}
}

func TestProvidersEndpoint(t *testing.T) {
	r := newTestRouter(t, denyAll{})
	w := doRequest(t, r, http.MethodGet, "/v1/providers")
	if w.Code != http.StatusOK {
		t.Fatalf("/v1/providers status = %d, want 200", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("response missing data object: %s", w.Body.String())
	}
	if _, ok := data["actor_providers"]; !ok {
		t.Errorf("data missing actor_providers: %v", data)
	}
	if _, ok := data["movie_providers"]; !ok {
		t.Errorf("data missing movie_providers: %v", data)
	}
}

func TestPrivateEndpointRequiresAuth(t *testing.T) {
	r := newTestRouter(t, denyAll{})
	w := doRequest(t, r, http.MethodGet, "/v1/db/version")
	if w.Code != http.StatusUnauthorized && w.Code != http.StatusForbidden {
		t.Fatalf("unauthenticated /v1/db/version status = %d, want 401/403; body=%s",
			w.Code, w.Body.String())
	}
}
