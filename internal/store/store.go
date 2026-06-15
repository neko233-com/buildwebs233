package store

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/neko233-com/buildwebs233/internal/config"
)

type Block struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Label    string            `json:"label,omitempty"`
	Content  string            `json:"content,omitempty"`
	Props    map[string]string `json:"props,omitempty"`
	Style    map[string]string `json:"style,omitempty"`
	Children []Block           `json:"children,omitempty"`
}

type Section struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Layout string            `json:"layout"`
	Style  map[string]string `json:"style,omitempty"`
	Blocks []Block           `json:"blocks"`
}

type PageSEO struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Keywords    string `json:"keywords,omitempty"`
}

type ComplianceItem struct {
	Code   string `json:"code"`
	Label  string `json:"label"`
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
}

type ComplianceProfile struct {
	CompanyName     string               `json:"company_name,omitempty"`
	ContactName     string               `json:"contact_name,omitempty"`
	ContactPhone    string               `json:"contact_phone,omitempty"`
	RegionRule      string               `json:"region_rule,omitempty"`
	ICPStatus       string               `json:"icp_status,omitempty"`
	PSBStatus       string               `json:"psb_status,omitempty"`
	ReviewStatus    string               `json:"review_status,omitempty"`
	ReviewNotes     string               `json:"review_notes,omitempty"`
	Checklist       []ComplianceItem     `json:"checklist,omitempty"`
	Materials       []ComplianceMaterial `json:"materials,omitempty"`
	LastReviewedAt  *time.Time           `json:"last_reviewed_at,omitempty"`
	LastSubmittedAt *time.Time           `json:"last_submitted_at,omitempty"`
}

type ComplianceMaterial struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	FileName   string     `json:"file_name"`
	FilePath   string     `json:"file_path"`
	PublicURL  string     `json:"public_url"`
	Status     string     `json:"status"`
	Note       string     `json:"note,omitempty"`
	UploadedAt time.Time  `json:"uploaded_at"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
}

type Page struct {
	ID            string    `json:"id"`
	SiteID        string    `json:"site_id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Title         string    `json:"title"`
	TemplateID    string    `json:"template_id,omitempty"`
	Status        string    `json:"status"`
	SchemaVersion int       `json:"schema_version"`
	SEO           PageSEO   `json:"seo,omitempty"`
	Blocks        []Block   `json:"blocks,omitempty"`
	Sections      []Section `json:"sections,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Site struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Domain        string            `json:"domain"`
	Region        string            `json:"region"`
	TemplateID    string            `json:"template_id"`
	PrimaryPageID string            `json:"primary_page_id,omitempty"`
	Status        string            `json:"status"`
	ICPNumber     string            `json:"icp_number,omitempty"`
	PSBNumber     string            `json:"psb_number,omitempty"`
	Theme         map[string]string `json:"theme,omitempty"`
	Compliance    ComplianceProfile `json:"compliance"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type Template struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Thumbnail   string `json:"thumbnail"`
}

type PageRevision struct {
	ID        string    `json:"id"`
	PageID    string    `json:"page_id"`
	SiteID    string    `json:"site_id"`
	Version   int       `json:"version"`
	Status    string    `json:"status"`
	Source    string    `json:"source"`
	Snapshot  Page      `json:"snapshot"`
	CreatedAt time.Time `json:"created_at"`
}

type DiskStore struct {
	mu            sync.RWMutex
	pagesPath     string
	revisionsPath string
	sitesPath     string
	templatesPath string
	uploadsDir    string
	pages         map[string]*Page
	revisions     []PageRevision
	sites         map[string]*Site
	templates     map[string]*Template
}

type pagesFile struct {
	Items []Page `json:"items"`
}

type templatesFile struct {
	Items []Template `json:"items"`
}

type sitesFile struct {
	Items []Site `json:"items"`
}

type revisionsFile struct {
	Items []PageRevision `json:"items"`
}

func NewDiskStore(cfg config.StorageConfig) (*DiskStore, error) {
	s := &DiskStore{
		pagesPath:     filepath.Join(cfg.DataDir, cfg.PagesFile),
		revisionsPath: filepath.Join(cfg.DataDir, cfg.RevisionsFile),
		sitesPath:     filepath.Join(cfg.DataDir, cfg.SitesFile),
		templatesPath: filepath.Join(cfg.DataDir, cfg.TemplatesFile),
		uploadsDir:    filepath.Join(cfg.DataDir, cfg.UploadsDir),
		pages:         map[string]*Page{},
		revisions:     []PageRevision{},
		sites:         map[string]*Site{},
		templates:     map[string]*Template{},
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.uploadsDir, 0o755); err != nil {
		return nil, err
	}
	if err := s.loadPages(); err != nil {
		return nil, err
	}
	if err := s.loadRevisions(); err != nil {
		return nil, err
	}
	if err := s.loadSites(); err != nil {
		return nil, err
	}
	if err := s.loadTemplates(); err != nil {
		return nil, err
	}
	s.ensureDefaults()
	return s, nil
}

func (s *DiskStore) ListPages() []Page {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Page, 0, len(s.pages))
	for _, p := range s.pages {
		result = append(result, *p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result
}

func (s *DiskStore) ListPagesBySite(siteID string) []Page {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Page, 0)
	for _, p := range s.pages {
		if p.SiteID == siteID {
			result = append(result, *p)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result
}

func (s *DiskStore) ListPageRevisions(pageID string) []PageRevision {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]PageRevision, 0)
	for _, rev := range s.revisions {
		if rev.PageID == pageID {
			result = append(result, rev)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Version > result[j].Version
	})
	return result
}

func (s *DiskStore) ListSites() []Site {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Site, 0, len(s.sites))
	for _, site := range s.sites {
		result = append(result, *site)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result
}

func (s *DiskStore) ListTemplates() []Template {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Template, 0, len(s.templates))
	for _, t := range s.templates {
		result = append(result, *t)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

func (s *DiskStore) GetPageBySlug(slug string) (*Page, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.pages {
		if p.Slug == slug {
			cp := *p
			return &cp, true
		}
	}
	return nil, false
}

func (s *DiskStore) GetPageByID(id string) (*Page, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.pages[id]
	if !ok {
		return nil, false
	}
	cp := *p
	return &cp, true
}

func (s *DiskStore) GetSiteByID(id string) (*Site, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	site, ok := s.sites[id]
	if !ok {
		return nil, false
	}
	cp := *site
	return &cp, true
}

func (s *DiskStore) UpsertPage(p Page) (*Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if p.ID == "" {
		p.ID = randomID()
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		p.Name = "页面 " + p.ID[:6]
	}
	p.SiteID = strings.TrimSpace(p.SiteID)
	if p.SiteID == "" {
		p.SiteID = "site-default"
	}
	if _, ok := s.sites[p.SiteID]; !ok {
		return nil, os.ErrNotExist
	}
	p.Slug = strings.TrimSpace(p.Slug)
	if p.Slug == "" {
		p.Slug = slugify(p.Name)
	}
	p.Slug = ensureUniqueSlug(p.Slug, s.pages, p.ID)
	p.Status = strings.TrimSpace(p.Status)
	if p.Status == "" {
		p.Status = "draft"
	}
	if p.SchemaVersion == 0 {
		p.SchemaVersion = 2
	}
	p.TemplateID = strings.TrimSpace(p.TemplateID)
	if p.TemplateID == "" {
		p.TemplateID = s.sites[p.SiteID].TemplateID
	}
	p.normalizeSections()
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = time.Now()
	}
	p.UpdatedAt = time.Now()
	s.pages[p.ID] = &p
	site := s.sites[p.SiteID]
	if site.PrimaryPageID == "" {
		site.PrimaryPageID = p.ID
		site.UpdatedAt = time.Now()
		_ = s.saveSites()
	}
	if err := s.savePages(); err != nil {
		return nil, err
	}
	if err := s.appendRevision(p, "save"); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *DiskStore) UpsertSite(site Site) (*Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if site.ID == "" {
		site.ID = randomID()
	}
	site.Name = strings.TrimSpace(site.Name)
	if site.Name == "" {
		site.Name = "站点 " + site.ID[:6]
	}
	site.Region = strings.TrimSpace(site.Region)
	if site.Region == "" {
		site.Region = "CN"
	}
	site.TemplateID = strings.TrimSpace(site.TemplateID)
	if site.TemplateID == "" {
		site.TemplateID = "tpl-hero"
	}
	site.Status = strings.TrimSpace(site.Status)
	if site.Status == "" {
		site.Status = "planning"
	}
	site.Compliance = normalizeCompliance(site.Region, site.Compliance)
	if site.Theme == nil {
		site.Theme = map[string]string{
			"accent":  "#0284c7",
			"surface": "#0f172a",
		}
	}
	site.UpdatedAt = time.Now()
	s.sites[site.ID] = &site
	if err := s.saveSites(); err != nil {
		return nil, err
	}
	return &site, nil
}

func (s *DiskStore) DeletePage(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pages, id)
	return s.savePages()
}

func (s *DiskStore) PublishPage(pageID string) (*Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	page, ok := s.pages[pageID]
	if !ok {
		return nil, os.ErrNotExist
	}
	page.Status = "published"
	page.UpdatedAt = time.Now()
	if err := s.savePages(); err != nil {
		return nil, err
	}
	if err := s.appendRevision(*page, "publish"); err != nil {
		return nil, err
	}
	cp := *page
	return &cp, nil
}

func (s *DiskStore) SetSitePrimaryPage(siteID, pageID string) (*Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	site, ok := s.sites[siteID]
	if !ok {
		return nil, os.ErrNotExist
	}
	page, ok := s.pages[pageID]
	if !ok || page.SiteID != siteID {
		return nil, os.ErrNotExist
	}
	site.PrimaryPageID = pageID
	site.UpdatedAt = time.Now()
	if err := s.saveSites(); err != nil {
		return nil, err
	}
	cp := *site
	return &cp, nil
}

func (s *DiskStore) ApplyTemplateToSite(siteID, templateID string) (*Page, *Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	site, ok := s.sites[siteID]
	if !ok {
		return nil, nil, os.ErrNotExist
	}
	if _, ok := s.templates[templateID]; !ok {
		return nil, nil, os.ErrNotExist
	}
	site.TemplateID = templateID
	site.UpdatedAt = time.Now()

	pageID := site.PrimaryPageID
	if pageID == "" {
		pageID = randomID()
		site.PrimaryPageID = pageID
	}
	page := Page{
		ID:            pageID,
		SiteID:        siteID,
		Name:          site.Name + " 首页",
		Slug:          ensureUniqueSlug(slugify(site.Name), s.pages, pageID),
		Title:         site.Name,
		TemplateID:    templateID,
		Status:        "draft",
		SchemaVersion: 2,
		SEO: PageSEO{
			Title:       site.Name,
			Description: site.Name + " 官网首页",
		},
		Sections:  buildTemplateSections(templateID, site.Name),
		UpdatedAt: time.Now(),
	}
	page.normalizeSections()
	s.pages[page.ID] = &page

	if err := s.saveSites(); err != nil {
		return nil, nil, err
	}
	if err := s.savePages(); err != nil {
		return nil, nil, err
	}
	if err := s.appendRevision(page, "apply_template"); err != nil {
		return nil, nil, err
	}
	pageCopy := page
	siteCopy := *site
	return &pageCopy, &siteCopy, nil
}

func (s *DiskStore) SaveComplianceMaterial(siteID, materialType, fileName string, body []byte) (*ComplianceMaterial, *Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	site, ok := s.sites[siteID]
	if !ok {
		return nil, nil, os.ErrNotExist
	}
	safeName := sanitizeFileName(fileName)
	storedName := siteID + "-" + randomID()[:6] + "-" + safeName
	storedPath := filepath.Join(s.uploadsDir, storedName)
	if err := os.WriteFile(storedPath, body, 0o644); err != nil {
		return nil, nil, err
	}
	material := ComplianceMaterial{
		ID:         randomID(),
		Type:       strings.TrimSpace(materialType),
		FileName:   fileName,
		FilePath:   storedPath,
		PublicURL:  "/uploads/" + storedName,
		Status:     "uploaded",
		UploadedAt: time.Now(),
	}
	if material.Type == "" {
		material.Type = "general"
	}
	site.Compliance.Materials = append([]ComplianceMaterial{material}, site.Compliance.Materials...)
	site.Compliance.ReviewStatus = "materials_uploaded"
	now := time.Now()
	site.Compliance.LastSubmittedAt = &now
	site.UpdatedAt = time.Now()
	if err := s.saveSites(); err != nil {
		return nil, nil, err
	}
	matCopy := material
	siteCopy := *site
	return &matCopy, &siteCopy, nil
}

func (s *DiskStore) ReviewCompliance(siteID, action, note, materialID string) (*Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	site, ok := s.sites[siteID]
	if !ok {
		return nil, os.ErrNotExist
	}
	now := time.Now()
	switch strings.TrimSpace(action) {
	case "submit":
		site.Compliance.ReviewStatus = "submitted"
		site.Compliance.LastSubmittedAt = &now
	case "approve":
		site.Compliance.ReviewStatus = "approved"
		site.Compliance.ICPStatus = "approved"
		site.Compliance.PSBStatus = "approved"
	case "reject":
		site.Compliance.ReviewStatus = "rejected"
		site.Compliance.ReviewNotes = note
	case "mark_material_verified":
		for i := range site.Compliance.Materials {
			if site.Compliance.Materials[i].ID == materialID {
				site.Compliance.Materials[i].Status = "verified"
				site.Compliance.Materials[i].ReviewedAt = &now
				site.Compliance.Materials[i].Note = note
			}
		}
	default:
		site.Compliance.ReviewStatus = action
	}
	site.Compliance.LastReviewedAt = &now
	site.UpdatedAt = time.Now()
	if err := s.saveSites(); err != nil {
		return nil, err
	}
	cp := *site
	return &cp, nil
}

func (s *DiskStore) loadPages() error {
	b, err := os.ReadFile(s.pagesPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var data pagesFile
	if err := json.Unmarshal(b, &data); err != nil {
		log.Printf("[store] read pages failed, fallback empty: %v", err)
		return nil
	}
	for i := range data.Items {
		item := data.Items[i]
		copied := item
		s.pages[copied.ID] = &copied
	}
	return nil
}

func (s *DiskStore) loadSites() error {
	b, err := os.ReadFile(s.sitesPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var data sitesFile
	if err := json.Unmarshal(b, &data); err != nil {
		log.Printf("[store] read sites failed, fallback empty: %v", err)
		return nil
	}
	for i := range data.Items {
		item := data.Items[i]
		copied := item
		s.sites[copied.ID] = &copied
	}
	return nil
}

func (s *DiskStore) loadRevisions() error {
	b, err := os.ReadFile(s.revisionsPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var data revisionsFile
	if err := json.Unmarshal(b, &data); err != nil {
		log.Printf("[store] read revisions failed, fallback empty: %v", err)
		return nil
	}
	s.revisions = data.Items
	return nil
}

func (s *DiskStore) loadTemplates() error {
	b, err := os.ReadFile(s.templatesPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var data templatesFile
	if err := json.Unmarshal(b, &data); err != nil {
		return nil
	}
	for i := range data.Items {
		item := data.Items[i]
		c := item
		s.templates[c.ID] = &c
	}
	return nil
}

func (s *DiskStore) savePages() error {
	list := make([]Page, 0, len(s.pages))
	for _, p := range s.pages {
		list = append(list, *p)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].UpdatedAt.After(list[j].UpdatedAt)
	})
	payload := pagesFile{Items: list}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.pagesPath, b, 0o644)
}

func (s *DiskStore) saveRevisions() error {
	payload := revisionsFile{Items: s.revisions}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.revisionsPath, b, 0o644)
}

func (s *DiskStore) saveSites() error {
	list := make([]Site, 0, len(s.sites))
	for _, site := range s.sites {
		list = append(list, *site)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].UpdatedAt.After(list[j].UpdatedAt)
	})
	payload := sitesFile{Items: list}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.sitesPath, b, 0o644)
}

func (s *DiskStore) saveTemplates() error {
	list := make([]Template, 0, len(s.templates))
	for _, t := range s.templates {
		list = append(list, *t)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	payload := templatesFile{Items: list}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.templatesPath, b, 0o644)
}

func (s *DiskStore) ensureDefaults() {
	if len(s.sites) == 0 {
		s.sites["site-default"] = &Site{
			ID:         "site-default",
			Name:       "默认企业站",
			Domain:     "example.com",
			Region:     "CN",
			TemplateID: "tpl-hero",
			Status:     "planning",
			Theme: map[string]string{
				"accent":  "#0284c7",
				"surface": "#0f172a",
			},
			Compliance: normalizeCompliance("CN", ComplianceProfile{}),
			UpdatedAt:  time.Now(),
		}
		_ = s.saveSites()
	}
	if len(s.templates) == 0 {
		s.templates["tpl-hero"] = &Template{
			ID:          "tpl-hero",
			Name:        "企业官网模板",
			Description: "含标题、简介、按钮与图文区块，适合快速建站。",
			Thumbnail:   "/assets/templates/hero.png",
		}
		s.templates["tpl-product"] = &Template{
			ID:          "tpl-product",
			Name:        "产品展示模板",
			Description: "多卡片+功能列表，适合 SaaS/工具/服务。",
			Thumbnail:   "/assets/templates/product.png",
		}
		_ = s.saveTemplates()
	}
}

func randomID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "id"
	}
	return hex.EncodeToString(b)
}

func slugify(src string) string {
	s := strings.ToLower(src)
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, c := range s {
		switch {
		case (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-':
			b.WriteRune(c)
		case c == '-' || c == '_':
			b.WriteRune('-')
		}
	}
	result := b.String()
	result = strings.Trim(result, "-")
	if result == "" {
		result = "page"
	}
	return result
}

func ensureUniqueSlug(slug string, all map[string]*Page, selfID string) string {
	if slug == "" {
		slug = randomID()
	}
	exists := false
	for _, p := range all {
		if p.ID != selfID && p.Slug == slug {
			exists = true
			break
		}
	}
	if !exists {
		return slug
	}
	for i := 1; ; i++ {
		candidate := slug + "-" + time.Now().Format("150405") + "-" + randomID()[:4]
		dup := false
		for _, p := range all {
			if p.ID != selfID && p.Slug == candidate {
				dup = true
				break
			}
		}
		if !dup {
			return candidate
		}
	}
}

func (s *DiskStore) appendRevision(page Page, source string) error {
	version := 1
	for _, rev := range s.revisions {
		if rev.PageID == page.ID && rev.Version >= version {
			version = rev.Version + 1
		}
	}
	revision := PageRevision{
		ID:        randomID(),
		PageID:    page.ID,
		SiteID:    page.SiteID,
		Version:   version,
		Status:    page.Status,
		Source:    source,
		Snapshot:  page,
		CreatedAt: time.Now(),
	}
	s.revisions = append([]PageRevision{revision}, s.revisions...)
	return s.saveRevisions()
}

func (p *Page) normalizeSections() {
	if len(p.Sections) == 0 && len(p.Blocks) > 0 {
		p.Sections = []Section{
			{
				ID:     "section-main",
				Name:   "主区域",
				Layout: "stack",
				Blocks: p.Blocks,
			},
		}
	}
	for si := range p.Sections {
		section := &p.Sections[si]
		if section.ID == "" {
			section.ID = "section-" + randomID()[:6]
		}
		if section.Name == "" {
			section.Name = "区块区域"
		}
		if section.Layout == "" {
			section.Layout = "stack"
		}
		for bi := range section.Blocks {
			normalizeBlock(&section.Blocks[bi])
		}
	}
}

func buildTemplateSections(templateID, siteName string) []Section {
	switch templateID {
	case "tpl-product":
		return []Section{
			{
				ID:     "hero-product",
				Name:   "产品主视觉",
				Layout: "stack",
				Blocks: []Block{
					{Type: "hero", Props: map[string]string{"headline": siteName + " 产品方案"}},
					{Type: "text", Props: map[string]string{"text": "为企业提供高效、可信、可发布的网站解决方案。"}},
					{Type: "button", Props: map[string]string{"label": "申请演示"}},
				},
			},
			{
				ID:     "features",
				Name:   "产品特性",
				Layout: "grid",
				Blocks: []Block{
					{Type: "text", Props: map[string]string{"text": "低代码拖拽"}},
					{Type: "text", Props: map[string]string{"text": "模板快速套用"}},
					{Type: "text", Props: map[string]string{"text": "备案流程支持"}},
				},
			},
		}
	default:
		return []Section{
			{
				ID:     "hero-site",
				Name:   "官网主视觉",
				Layout: "stack",
				Blocks: []Block{
					{Type: "hero", Props: map[string]string{"headline": siteName + " 欢迎页"}},
					{Type: "text", Props: map[string]string{"text": "这是自动套用模板生成的首页。"}},
					{Type: "button", Props: map[string]string{"label": "立即联系"}},
				},
			},
			{
				ID:     "intro",
				Name:   "介绍区域",
				Layout: "stack",
				Blocks: []Block{
					{Type: "text", Props: map[string]string{"text": siteName + " 提供值得信赖的服务与交付能力。"}},
				},
			},
		}
	}
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "file.bin"
	}
	var out bytes.Buffer
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			out.WriteRune(r)
		case r == '.', r == '-', r == '_':
			out.WriteRune(r)
		default:
			out.WriteRune('-')
		}
	}
	result := out.String()
	result = strings.Trim(result, "-")
	if result == "" {
		return "file.bin"
	}
	return result
}

func normalizeBlock(b *Block) {
	if b.ID == "" {
		b.ID = "block-" + randomID()[:6]
	}
	b.Type = strings.TrimSpace(b.Type)
	if b.Type == "" {
		b.Type = "text"
	}
	if b.Props == nil {
		b.Props = map[string]string{}
	}
	if b.Content != "" {
		switch b.Type {
		case "text":
			if b.Props["text"] == "" {
				b.Props["text"] = b.Content
			}
		case "button":
			if b.Props["label"] == "" {
				b.Props["label"] = b.Content
			}
		case "hero":
			if b.Props["headline"] == "" {
				b.Props["headline"] = b.Content
			}
		}
	}
	for i := range b.Children {
		normalizeBlock(&b.Children[i])
	}
}

func normalizeCompliance(region string, compliance ComplianceProfile) ComplianceProfile {
	if compliance.RegionRule == "" {
		compliance.RegionRule = region
	}
	if compliance.ICPStatus == "" {
		compliance.ICPStatus = "not_started"
	}
	if compliance.PSBStatus == "" {
		compliance.PSBStatus = "not_started"
	}
	if compliance.ReviewStatus == "" {
		compliance.ReviewStatus = "draft"
	}
	if len(compliance.Checklist) == 0 {
		compliance.Checklist = []ComplianceItem{
			{Code: "business-license", Label: "营业执照", Status: "missing"},
			{Code: "legal-identity", Label: "法人身份证明", Status: "missing"},
			{Code: "domain-proof", Label: "域名持有证明", Status: "missing"},
			{Code: "hosting-proof", Label: "接入/服务器证明", Status: "missing"},
		}
	}
	return compliance
}
