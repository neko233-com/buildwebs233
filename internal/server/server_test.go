package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/neko233-com/buildwebs233/internal/config"
	"github.com/neko233-com/buildwebs233/internal/hotreload"
	"github.com/neko233-com/buildwebs233/internal/store"
)

func TestLoginAndSavePageFlow(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	router := chi.NewRouter()
	app.RegisterRoutes(router)

	savePayload := `{"site_id":"site-default","name":"Landing","title":"Landing","sections":[{"id":"s1","name":"主区域","layout":"stack","blocks":[{"id":"b1","type":"text","props":{"text":"hello"}}]}]}`
	unauthReq := httptest.NewRequest(http.MethodPost, "/api/admin/pages", strings.NewReader(savePayload))
	unauthReq.Header.Set("Content-Type", "application/json")
	unauthRec := httptest.NewRecorder()
	router.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", unauthRec.Code)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"username":"root","password":"root"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected 200 login, got %d", loginRec.Code)
	}
	cookies := loginRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected auth cookie")
	}

	saveReq := httptest.NewRequest(http.MethodPost, "/api/admin/pages", bytes.NewBufferString(savePayload))
	saveReq.Header.Set("Content-Type", "application/json")
	saveReq.AddCookie(cookies[0])
	saveRec := httptest.NewRecorder()
	router.ServeHTTP(saveRec, saveReq)
	if saveRec.Code != http.StatusOK {
		t.Fatalf("expected 200 save, got %d", saveRec.Code)
	}

	var saved store.Page
	if err := json.Unmarshal(saveRec.Body.Bytes(), &saved); err != nil {
		t.Fatalf("decode save response: %v", err)
	}
	if saved.Slug != "landing" {
		t.Fatalf("expected landing slug, got %q", saved.Slug)
	}

	pageReq := httptest.NewRequest(http.MethodGet, "/p/"+saved.Slug, nil)
	pageRec := httptest.NewRecorder()
	router.ServeHTTP(pageRec, pageReq)
	if pageRec.Code != http.StatusOK {
		t.Fatalf("expected 200 render, got %d", pageRec.Code)
	}
	if !strings.Contains(pageRec.Body.String(), "hello") {
		t.Fatalf("expected rendered content, got %q", pageRec.Body.String())
	}
}

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	router := chi.NewRouter()
	app.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("expected ok health response, got %q", rec.Body.String())
	}
}

func TestRoadmapAndSitesEndpoints(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	router := chi.NewRouter()
	app.RegisterRoutes(router)

	roadmapReq := httptest.NewRequest(http.MethodGet, "/api/platform/roadmap", nil)
	roadmapRec := httptest.NewRecorder()
	router.ServeHTTP(roadmapRec, roadmapReq)
	if roadmapRec.Code != http.StatusOK {
		t.Fatalf("expected roadmap 200, got %d", roadmapRec.Code)
	}
	if !strings.Contains(roadmapRec.Body.String(), `"recommended"`) {
		t.Fatalf("expected roadmap payload, got %q", roadmapRec.Body.String())
	}

	sitesReq := httptest.NewRequest(http.MethodGet, "/api/sites", nil)
	sitesRec := httptest.NewRecorder()
	router.ServeHTTP(sitesRec, sitesReq)
	if sitesRec.Code != http.StatusOK {
		t.Fatalf("expected sites 200, got %d", sitesRec.Code)
	}
	if !strings.Contains(sitesRec.Body.String(), "默认企业站") {
		t.Fatalf("expected default site, got %q", sitesRec.Body.String())
	}

	sitePagesReq := httptest.NewRequest(http.MethodGet, "/api/sites/site-default/pages", nil)
	sitePagesRec := httptest.NewRecorder()
	router.ServeHTTP(sitePagesRec, sitePagesReq)
	if sitePagesRec.Code != http.StatusOK {
		t.Fatalf("expected site pages 200, got %d", sitePagesRec.Code)
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()

	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	cfgPath := filepath.Join(dir, "server.yaml")
	cfgBody := []byte("server:\n  static_dir: \"" + filepath.ToSlash(filepath.Join(dir, "web")) + "\"\nauth:\n  username: \"root\"\n  password: \"root\"\nstorage:\n  data_dir: \"" + filepath.ToSlash(dataDir) + "\"\n  pages_file: \"pages.json\"\n  sites_file: \"sites.json\"\n  templates_file: \"templates.json\"\n")
	if err := os.WriteFile(cfgPath, cfgBody, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfgManager, err := config.NewManager(cfgPath)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	repo, err := store.NewDiskStore(cfgManager.Config().Storage)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	return NewApp(cfgManager, repo, hotreload.NewHub(), nil)
}
