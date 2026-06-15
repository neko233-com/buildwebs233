package server

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/neko233-com/buildwebs233/internal/config"
	"github.com/neko233-com/buildwebs233/internal/hotreload"
	"github.com/neko233-com/buildwebs233/internal/platform"
	"github.com/neko233-com/buildwebs233/internal/store"
)

type App struct {
	cfgManager *config.Manager
	store      *store.DiskStore
	reloadHub  *hotreload.Hub
	logger     *log.Logger

	sessions sync.Map
}

type jsonError struct {
	Error string `json:"error"`
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type applyTemplateReq struct {
	TemplateID string `json:"template_id"`
}

type setHomepageReq struct {
	PageID string `json:"page_id"`
}

type reviewComplianceReq struct {
	Action     string `json:"action"`
	Note       string `json:"note"`
	MaterialID string `json:"material_id,omitempty"`
}

func NewApp(cfg *config.Manager, s *store.DiskStore, reload *hotreload.Hub, logger *log.Logger) *App {
	return &App{
		cfgManager: cfg,
		store:      s,
		reloadHub:  reload,
		logger:     logger,
	}
}

func (a *App) RegisterRoutes(r chi.Router) {
	r.Get("/api/health", a.handleHealth)
	r.Get("/api/config", a.handleConfig)
	r.Get("/api/reload", a.handleReloadSSE)
	r.Get("/api/pages", a.handleListPages)
	r.Get("/api/pages/{id}", a.handleGetPage)
	r.Get("/api/pages/{id}/revisions", a.handleListPageRevisions)
	r.Get("/api/sites", a.handleListSites)
	r.Get("/api/sites/{id}", a.handleGetSite)
	r.Get("/api/sites/{id}/pages", a.handleListSitePages)
	r.Get("/api/templates", a.handleListTemplates)
	r.Get("/api/platform/roadmap", a.handleRoadmap)
	r.Get("/__reload-client.js", a.handleReloadClient)
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir(a.currentConfig().UploadsPath()))))

	r.Post("/api/login", a.handleLogin)
	r.Post("/api/logout", a.handleLogout)

	r.Route("/api/admin", func(r chi.Router) {
		r.Use(a.authMiddleware)
		r.Post("/pages", a.handleSavePage)
		r.Delete("/pages/{id}", a.handleDeletePage)
		r.Post("/pages/{id}/publish", a.handlePublishPage)
		r.Post("/sites", a.handleSaveSite)
		r.Post("/sites/{id}/apply-template", a.handleApplyTemplate)
		r.Post("/sites/{id}/homepage", a.handleSetHomepage)
		r.Post("/sites/{id}/compliance/materials", a.handleUploadComplianceMaterial)
		r.Post("/sites/{id}/compliance/review", a.handleReviewCompliance)
	})

	r.Get("/admin", a.handleAdminStatic)
	r.Get("/admin/*", a.handleAdminStatic)
	r.Get("/p/{slug}", a.handleRenderPage)
	r.Get("/", a.handleAdminStatic)
}

func (a *App) currentConfig() config.Config {
	return a.cfgManager.Config()
}

func (a *App) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a.currentConfig())
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	cfg := a.currentConfig()
	res := map[string]any{
		"ok":      true,
		"name":    "buildwebs233-server",
		"port":    cfg.Server.Port,
		"time":    time.Now().Format(time.RFC3339),
		"version": "0.1.0",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in loginReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid login payload")
		return
	}
	cfg := a.currentConfig()
	if in.Username != cfg.Auth.Username || in.Password != cfg.Auth.Password {
		a.writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token := randomToken()
	a.sessions.Store(token, in.Username)
	http.SetCookie(w, &http.Cookie{
		Name:     "bw_admin",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok": true,
	})
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("bw_admin")
	if err == nil {
		a.sessions.Delete(c.Value)
		http.SetCookie(w, &http.Cookie{
			Name:     "bw_admin",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Expires:  time.Unix(0, 0),
		})
	}
	a.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleListPages(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimSpace(r.URL.Query().Get("site_id"))
	if siteID != "" {
		a.writeJSON(w, http.StatusOK, a.store.ListPagesBySite(siteID))
		return
	}
	pages := a.store.ListPages()
	a.writeJSON(w, http.StatusOK, pages)
}

func (a *App) handleGetPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, ok := a.store.GetPageByID(id)
	if !ok {
		a.writeError(w, http.StatusNotFound, "page not found")
		return
	}
	a.writeJSON(w, http.StatusOK, p)
}

func (a *App) handleListPageRevisions(w http.ResponseWriter, r *http.Request) {
	pageID := chi.URLParam(r, "id")
	if _, ok := a.store.GetPageByID(pageID); !ok {
		a.writeError(w, http.StatusNotFound, "page not found")
		return
	}
	a.writeJSON(w, http.StatusOK, a.store.ListPageRevisions(pageID))
}

func (a *App) handleListSites(w http.ResponseWriter, r *http.Request) {
	sites := a.store.ListSites()
	a.writeJSON(w, http.StatusOK, sites)
}

func (a *App) handleGetSite(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	site, ok := a.store.GetSiteByID(id)
	if !ok {
		a.writeError(w, http.StatusNotFound, "site not found")
		return
	}
	a.writeJSON(w, http.StatusOK, site)
}

func (a *App) handleListSitePages(w http.ResponseWriter, r *http.Request) {
	siteID := chi.URLParam(r, "id")
	if _, ok := a.store.GetSiteByID(siteID); !ok {
		a.writeError(w, http.StatusNotFound, "site not found")
		return
	}
	a.writeJSON(w, http.StatusOK, a.store.ListPagesBySite(siteID))
}

func (a *App) handleSavePage(w http.ResponseWriter, r *http.Request) {
	var in store.Page
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid page payload")
		return
	}
	saved, err := a.store.UpsertPage(in)
	if err != nil {
		if err == os.ErrNotExist {
			a.writeError(w, http.StatusBadRequest, "site not found for page")
			return
		}
		a.writeError(w, http.StatusInternalServerError, "save page failed")
		return
	}
	a.reloadHub.Broadcast(hotreload.Event{Type: "html", File: "/p/" + saved.Slug})
	a.writeJSON(w, http.StatusOK, saved)
}

func (a *App) handleSaveSite(w http.ResponseWriter, r *http.Request) {
	var in store.Site
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid site payload")
		return
	}
	saved, err := a.store.UpsertSite(in)
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "save site failed")
		return
	}
	a.writeJSON(w, http.StatusOK, saved)
}

func (a *App) handlePublishPage(w http.ResponseWriter, r *http.Request) {
	pageID := chi.URLParam(r, "id")
	saved, err := a.store.PublishPage(pageID)
	if err != nil {
		if err == os.ErrNotExist {
			a.writeError(w, http.StatusNotFound, "page not found")
			return
		}
		a.writeError(w, http.StatusInternalServerError, "publish page failed")
		return
	}
	a.reloadHub.Broadcast(hotreload.Event{Type: "html", File: "/p/" + saved.Slug})
	a.writeJSON(w, http.StatusOK, saved)
}

func (a *App) handleApplyTemplate(w http.ResponseWriter, r *http.Request) {
	siteID := chi.URLParam(r, "id")
	var in applyTemplateReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid template payload")
		return
	}
	page, site, err := a.store.ApplyTemplateToSite(siteID, in.TemplateID)
	if err != nil {
		if err == os.ErrNotExist {
			a.writeError(w, http.StatusNotFound, "site or template not found")
			return
		}
		a.writeError(w, http.StatusInternalServerError, "apply template failed")
		return
	}
	a.reloadHub.Broadcast(hotreload.Event{Type: "html", File: "/p/" + page.Slug})
	a.writeJSON(w, http.StatusOK, map[string]any{
		"site": site,
		"page": page,
	})
}

func (a *App) handleSetHomepage(w http.ResponseWriter, r *http.Request) {
	siteID := chi.URLParam(r, "id")
	var in setHomepageReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid homepage payload")
		return
	}
	site, err := a.store.SetSitePrimaryPage(siteID, in.PageID)
	if err != nil {
		if err == os.ErrNotExist {
			a.writeError(w, http.StatusNotFound, "site or page not found")
			return
		}
		a.writeError(w, http.StatusInternalServerError, "set homepage failed")
		return
	}
	a.writeJSON(w, http.StatusOK, site)
}

func (a *App) handleUploadComplianceMaterial(w http.ResponseWriter, r *http.Request) {
	siteID := chi.URLParam(r, "id")
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		a.writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()
	body, err := io.ReadAll(file)
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "read upload failed")
		return
	}
	materialType := r.FormValue("type")
	material, site, err := a.store.SaveComplianceMaterial(siteID, materialType, header.Filename, body)
	if err != nil {
		if err == os.ErrNotExist {
			a.writeError(w, http.StatusNotFound, "site not found")
			return
		}
		a.writeError(w, http.StatusInternalServerError, "save material failed")
		return
	}
	a.writeJSON(w, http.StatusOK, map[string]any{
		"material": material,
		"site":     site,
	})
}

func (a *App) handleReviewCompliance(w http.ResponseWriter, r *http.Request) {
	siteID := chi.URLParam(r, "id")
	var in reviewComplianceReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid review payload")
		return
	}
	site, err := a.store.ReviewCompliance(siteID, in.Action, in.Note, in.MaterialID)
	if err != nil {
		if err == os.ErrNotExist {
			a.writeError(w, http.StatusNotFound, "site not found")
			return
		}
		a.writeError(w, http.StatusInternalServerError, "review compliance failed")
		return
	}
	a.writeJSON(w, http.StatusOK, site)
}

func (a *App) handleDeletePage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := a.store.DeletePage(id); err != nil {
		a.writeError(w, http.StatusInternalServerError, "delete page failed")
		return
	}
	a.reloadHub.Broadcast(hotreload.Event{Type: "html", File: "/pages/" + id})
	a.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	templates := a.store.ListTemplates()
	a.writeJSON(w, http.StatusOK, templates)
}

func (a *App) handleRoadmap(w http.ResponseWriter, r *http.Request) {
	a.writeJSON(w, http.StatusOK, platform.DefaultRoadmap())
}

func (a *App) handleRenderPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	p, ok := a.store.GetPageBySlug(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	html := renderHTML(*p)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func (a *App) handleAdminStatic(w http.ResponseWriter, r *http.Request) {
	cfg := a.currentConfig()
	if err := os.MkdirAll(cfg.Server.StaticDir, 0o755); err != nil {
		http.Error(w, "static dir unavailable", http.StatusServiceUnavailable)
		return
	}

	indexPath := filepath.Join(cfg.Server.StaticDir, "index.html")
	path := strings.TrimPrefix(r.URL.Path, "/admin")
	if strings.TrimSpace(path) == "" || path == "/" {
		if _, err := os.Stat(indexPath); err == nil {
			http.ServeFile(w, r, indexPath)
			return
		}
		a.renderFallbackAdmin(w)
		return
	}

	filePath := filepath.Join(cfg.Server.StaticDir, path)
	if _, err := os.Stat(filePath); err == nil {
		http.ServeFile(w, r, filePath)
		return
	}
	a.renderFallbackAdmin(w)
}

func (a *App) renderFallbackAdmin(w http.ResponseWriter) {
	fallback := `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>buildwebs233 管理后台</title>
  <style>
    body { margin: 0; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #0f172a; color: #e2e8f0; }
    .box { max-width: 860px; margin: 12vh auto; background: #111827; border: 1px solid #334155; padding: 24px; border-radius: 12px; }
    .ok { color: #22d3ee; }
    a { color: #93c5fd; }
  </style>
</head>
<body>
  <div class="box">
    <h1>buildwebs233 管理后台</h1>
    <p>当前为 fallback 页面，建议运行前端构建后将产物放到 <code>web/</code>。</p>
    <p>管理员账号: <strong>root</strong> / <strong>root</strong></p>
    <p>API: <a href="/api/health">/api/health</a>、<a href="/api/pages">/api/pages</a>、<a href="/api/templates">/api/templates</a></p>
    <p class="ok">热重载通道: <code>/api/reload</code></p>
  </div>
  <script src="/__reload-client.js"></script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(fallback))
}

func (a *App) handleReloadClient(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	_, _ = w.Write([]byte(hotreload.ReloadClientScript()))
}

func (a *App) handleReloadSSE(w http.ResponseWriter, r *http.Request) {
	a.reloadHub.ServeSSE(w, r)
}

func (a *App) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.isAuthenticated(r) {
			next.ServeHTTP(w, r)
			return
		}
		a.writeError(w, http.StatusUnauthorized, "unauthorized")
	})
}

func (a *App) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("bw_admin")
	if err != nil {
		return false
	}
	_, ok := a.sessions.Load(cookie.Value)
	return ok
}

func (a *App) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (a *App) writeError(w http.ResponseWriter, status int, msg string) {
	a.writeJSON(w, status, jsonError{Error: msg})
}

func renderHTML(p store.Page) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset='utf-8'/>")
	b.WriteString("<meta name='viewport' content='width=device-width, initial-scale=1' />")
	b.WriteString("<title>" + html.EscapeString(orDefault(p.SEO.Title, p.Title)) + "</title>")
	if strings.TrimSpace(p.SEO.Description) != "" {
		b.WriteString("<meta name='description' content='" + html.EscapeString(p.SEO.Description) + "' />")
	}
	b.WriteString("<style>body{margin:0;font-family:Arial,\"Noto Sans SC\";background:#f8fafc;color:#111827;} .hero{padding:40px;text-align:center;background:#0f172a;color:white;} .section{padding:24px;max-width:980px;margin:0 auto;} .block{margin:12px 0;padding:16px;border:1px solid #e2e8f0;border-radius:10px;background:white;box-shadow:0 6px 20px rgba(0,0,0,.08);} .btn{display:inline-block;padding:10px 14px;border-radius:8px;background:#0ea5e9;color:white;text-decoration:none;}</style>")
	b.WriteString("<script src='/__reload-client.js'></script>")
	b.WriteString("</head><body>")
	b.WriteString("<div class='hero'><h1>" + html.EscapeString(orDefault(p.Title, p.Name)) + "</h1></div>")
	for _, section := range p.Sections {
		b.WriteString("<section class='section'>")
		for _, block := range section.Blocks {
			switch strings.ToLower(block.Type) {
			case "text":
				b.WriteString("<div class='block'>" + html.EscapeString(orDefault(block.Props["text"], block.Content)) + "</div>")
			case "button":
				b.WriteString("<div><a class='btn' href='#'>" + html.EscapeString(orDefault(block.Props["label"], block.Content)) + "</a></div>")
			case "hero":
				b.WriteString("<div class='block'><strong>" + html.EscapeString(orDefault(block.Props["headline"], block.Content)) + "</strong></div>")
			default:
				b.WriteString("<div class='block'>" + html.EscapeString(orDefault(block.Props["text"], block.Content)) + "</div>")
			}
		}
		b.WriteString("</section>")
	}
	b.WriteString("</body></html>")
	return b.String()
}

var slugRE = regexp.MustCompile(`[^a-z0-9-]`)

func orDefault(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func sanitizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = slugRE.ReplaceAllString(s, "")
	return strings.Trim(s, "-")
}

func randomToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}
