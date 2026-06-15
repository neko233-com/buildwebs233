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
	ReviewHistory   []ComplianceEvent    `json:"review_history,omitempty"`
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

type ComplianceEvent struct {
	ID         string    `json:"id"`
	Action     string    `json:"action"`
	Actor      string    `json:"actor"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id,omitempty"`
	Note       string    `json:"note,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
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
	site.Compliance.ReviewHistory = append([]ComplianceEvent{{
		ID:         randomID(),
		Action:     "set_homepage",
		Actor:      "system",
		TargetType: "page",
		TargetID:   pageID,
		CreatedAt:  time.Now(),
	}}, site.Compliance.ReviewHistory...)
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
	site.Compliance.ReviewHistory = append([]ComplianceEvent{{
		ID:         randomID(),
		Action:     "apply_template",
		Actor:      "system",
		TargetType: "template",
		TargetID:   templateID,
		CreatedAt:  time.Now(),
	}}, site.Compliance.ReviewHistory...)
	_ = s.saveSites()
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
	site.Compliance.ReviewHistory = append([]ComplianceEvent{{
		ID:         randomID(),
		Action:     "upload_material",
		Actor:      "user",
		TargetType: "material",
		TargetID:   material.ID,
		Note:       material.Type + ":" + material.FileName,
		CreatedAt:  now,
	}}, site.Compliance.ReviewHistory...)
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
	site.Compliance.ReviewHistory = append([]ComplianceEvent{{
		ID:         randomID(),
		Action:     action,
		Actor:      "reviewer",
		TargetType: "site",
		TargetID:   siteID,
		Note:       note,
		CreatedAt:  now,
	}}, site.Compliance.ReviewHistory...)
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
		for _, tpl := range defaultTemplates() {
			template := tpl
			s.templates[template.ID] = &template
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
	headline, intro, cta := templateCopy(templateID, siteName)
	switch templateDomain(templateID) {
	case "ecommerce":
		return []Section{
			{
				ID:     "hero-commerce",
				Name:   "商城主视觉",
				Layout: "stack",
				Blocks: []Block{
					{Type: "hero", Props: map[string]string{"headline": headline}},
					{Type: "text", Props: map[string]string{"text": intro}},
					{Type: "button", Props: map[string]string{"label": cta}},
				},
			},
			{
				ID:     "offers-commerce",
				Name:   "核心卖点",
				Layout: "grid",
				Blocks: []Block{
					{Type: "text", Props: map[string]string{"text": "热销商品专区"}},
					{Type: "text", Props: map[string]string{"text": "支付与物流说明"}},
					{Type: "text", Props: map[string]string{"text": "售后与备案信息公示"}},
				},
			},
		}
	case "game":
		return []Section{
			{
				ID:     "hero-game",
				Name:   "游戏主视觉",
				Layout: "stack",
				Blocks: []Block{
					{Type: "hero", Props: map[string]string{"headline": headline}},
					{Type: "text", Props: map[string]string{"text": intro}},
					{Type: "button", Props: map[string]string{"label": cta}},
				},
			},
			{
				ID:     "highlights-game",
				Name:   "内容亮点",
				Layout: "grid",
				Blocks: []Block{
					{Type: "text", Props: map[string]string{"text": "版本介绍与运营日历"}},
					{Type: "text", Props: map[string]string{"text": "下载入口与玩家社区"}},
					{Type: "text", Props: map[string]string{"text": "防沉迷与合规说明"}},
				},
			},
		}
	case "blog":
		return []Section{
			{
				ID:     "hero-blog",
				Name:   "内容头图",
				Layout: "stack",
				Blocks: []Block{
					{Type: "hero", Props: map[string]string{"headline": headline}},
					{Type: "text", Props: map[string]string{"text": intro}},
					{Type: "button", Props: map[string]string{"label": cta}},
				},
			},
			{
				ID:     "columns-blog",
				Name:   "栏目区域",
				Layout: "grid",
				Blocks: []Block{
					{Type: "text", Props: map[string]string{"text": "精选文章列表"}},
					{Type: "text", Props: map[string]string{"text": "作者介绍与订阅引导"}},
					{Type: "text", Props: map[string]string{"text": "版权与备案信息"}},
				},
			},
		}
	case "technology":
		return []Section{
			{
				ID:     "hero-tech",
				Name:   "技术主视觉",
				Layout: "stack",
				Blocks: []Block{
					{Type: "hero", Props: map[string]string{"headline": headline}},
					{Type: "text", Props: map[string]string{"text": intro}},
					{Type: "button", Props: map[string]string{"label": cta}},
				},
			},
			{
				ID:     "modules-tech",
				Name:   "解决方案模块",
				Layout: "grid",
				Blocks: []Block{
					{Type: "text", Props: map[string]string{"text": "产品模块介绍"}},
					{Type: "text", Props: map[string]string{"text": "客户案例与接入流程"}},
					{Type: "text", Props: map[string]string{"text": "部署架构与合规承诺"}},
				},
			},
		}
	case "science":
		return []Section{
			{
				ID:     "hero-science",
				Name:   "科研主视觉",
				Layout: "stack",
				Blocks: []Block{
					{Type: "hero", Props: map[string]string{"headline": headline}},
					{Type: "text", Props: map[string]string{"text": intro}},
					{Type: "button", Props: map[string]string{"label": cta}},
				},
			},
			{
				ID:     "research-science",
				Name:   "研究栏目",
				Layout: "grid",
				Blocks: []Block{
					{Type: "text", Props: map[string]string{"text": "研究方向与团队介绍"}},
					{Type: "text", Props: map[string]string{"text": "项目成果与论文摘要"}},
					{Type: "text", Props: map[string]string{"text": "实验安全与机构信息"}},
				},
			},
		}
	case "outsourcing":
		return []Section{
			{
				ID:     "hero-outsourcing",
				Name:   "服务主视觉",
				Layout: "stack",
				Blocks: []Block{
					{Type: "hero", Props: map[string]string{"headline": headline}},
					{Type: "text", Props: map[string]string{"text": intro}},
					{Type: "button", Props: map[string]string{"label": cta}},
				},
			},
			{
				ID:     "delivery-outsourcing",
				Name:   "交付能力",
				Layout: "grid",
				Blocks: []Block{
					{Type: "text", Props: map[string]string{"text": "服务套餐与交付周期"}},
					{Type: "text", Props: map[string]string{"text": "团队履历与案例展示"}},
					{Type: "text", Props: map[string]string{"text": "合同流程与开票信息"}},
				},
			},
		}
	case "music":
		return []Section{
			{
				ID:     "hero-music",
				Name:   "音乐主视觉",
				Layout: "stack",
				Blocks: []Block{
					{Type: "hero", Props: map[string]string{"headline": headline}},
					{Type: "text", Props: map[string]string{"text": intro}},
					{Type: "button", Props: map[string]string{"label": cta}},
				},
			},
			{
				ID:     "tracks-music",
				Name:   "内容展示",
				Layout: "grid",
				Blocks: []Block{
					{Type: "text", Props: map[string]string{"text": "专辑与作品入口"}},
					{Type: "text", Props: map[string]string{"text": "演出日程与票务说明"}},
					{Type: "text", Props: map[string]string{"text": "版权与合作联系"}},
				},
			},
		}
	case "culture":
		return []Section{
			{
				ID:     "hero-culture",
				Name:   "文化主视觉",
				Layout: "stack",
				Blocks: []Block{
					{Type: "hero", Props: map[string]string{"headline": headline}},
					{Type: "text", Props: map[string]string{"text": intro}},
					{Type: "button", Props: map[string]string{"label": cta}},
				},
			},
			{
				ID:     "program-culture",
				Name:   "活动栏目",
				Layout: "grid",
				Blocks: []Block{
					{Type: "text", Props: map[string]string{"text": "展览活动与排期"}},
					{Type: "text", Props: map[string]string{"text": "机构介绍与馆藏内容"}},
					{Type: "text", Props: map[string]string{"text": "开放信息与预约规则"}},
				},
			},
		}
	case "news":
		return []Section{
			{
				ID:     "hero-news",
				Name:   "资讯头条",
				Layout: "stack",
				Blocks: []Block{
					{Type: "hero", Props: map[string]string{"headline": headline}},
					{Type: "text", Props: map[string]string{"text": intro}},
					{Type: "button", Props: map[string]string{"label": cta}},
				},
			},
			{
				ID:     "streams-news",
				Name:   "频道结构",
				Layout: "grid",
				Blocks: []Block{
					{Type: "text", Props: map[string]string{"text": "要闻推荐位"}},
					{Type: "text", Props: map[string]string{"text": "专题频道与记者署名"}},
					{Type: "text", Props: map[string]string{"text": "版权声明与举报入口"}},
				},
			},
		}
	case "medical":
		return []Section{
			{
				ID:     "hero-medical",
				Name:   "医疗主视觉",
				Layout: "stack",
				Blocks: []Block{
					{Type: "hero", Props: map[string]string{"headline": headline}},
					{Type: "text", Props: map[string]string{"text": intro}},
					{Type: "button", Props: map[string]string{"label": cta}},
				},
			},
			{
				ID:     "services-medical",
				Name:   "诊疗信息",
				Layout: "grid",
				Blocks: []Block{
					{Type: "text", Props: map[string]string{"text": "科室与服务介绍"}},
					{Type: "text", Props: map[string]string{"text": "医生团队与出诊安排"}},
					{Type: "text", Props: map[string]string{"text": "执业资质与就诊须知"}},
				},
			},
		}
	case "education":
		return []Section{
			{
				ID:     "hero-education",
				Name:   "教育主视觉",
				Layout: "stack",
				Blocks: []Block{
					{Type: "hero", Props: map[string]string{"headline": headline}},
					{Type: "text", Props: map[string]string{"text": intro}},
					{Type: "button", Props: map[string]string{"label": cta}},
				},
			},
			{
				ID:     "courses-education",
				Name:   "课程栏目",
				Layout: "grid",
				Blocks: []Block{
					{Type: "text", Props: map[string]string{"text": "课程体系与班型介绍"}},
					{Type: "text", Props: map[string]string{"text": "师资与校区信息"}},
					{Type: "text", Props: map[string]string{"text": "报名流程与监管信息"}},
				},
			},
		}
	case "finance":
		return []Section{
			{
				ID:     "hero-finance",
				Name:   "金融主视觉",
				Layout: "stack",
				Blocks: []Block{
					{Type: "hero", Props: map[string]string{"headline": headline}},
					{Type: "text", Props: map[string]string{"text": intro}},
					{Type: "button", Props: map[string]string{"label": cta}},
				},
			},
			{
				ID:     "services-finance",
				Name:   "服务栏目",
				Layout: "grid",
				Blocks: []Block{
					{Type: "text", Props: map[string]string{"text": "产品服务与费率说明"}},
					{Type: "text", Props: map[string]string{"text": "风险揭示与资质公示"}},
					{Type: "text", Props: map[string]string{"text": "客户支持与业务流程"}},
				},
			},
		}
	default:
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
}

func defaultTemplates() []Template {
	return []Template{
		{ID: "tpl-hero", Name: "企业官网模板", Description: "通用企业官网首页，适合公司介绍、联系入口与备案展示。", Thumbnail: "/assets/templates/hero.png"},
		{ID: "tpl-product", Name: "产品展示模板", Description: "适合 SaaS、工具与服务型产品的功能展示首页。", Thumbnail: "/assets/templates/product.png"},
		{ID: "tpl-enterprise-corporate", Name: "企业形象模板", Description: "适合品牌官网、企业形象和资质展示。", Thumbnail: "/assets/templates/enterprise-corporate.png"},
		{ID: "tpl-enterprise-factory", Name: "制造工厂模板", Description: "适合工厂、制造、设备类企业备案和官网展示。", Thumbnail: "/assets/templates/enterprise-factory.png"},
		{ID: "tpl-enterprise-service", Name: "专业服务模板", Description: "适合法务、咨询、会计、顾问类服务官网。", Thumbnail: "/assets/templates/enterprise-service.png"},
		{ID: "tpl-game-arcade", Name: "游戏发行模板", Description: "适合游戏官网、版本活动页与下载入口。", Thumbnail: "/assets/templates/game-arcade.png"},
		{ID: "tpl-game-esports", Name: "电竞赛事模板", Description: "适合电竞战队、赛事专题和直播聚合。", Thumbnail: "/assets/templates/game-esports.png"},
		{ID: "tpl-game-indie", Name: "独立游戏模板", Description: "适合独游工作室、试玩页和玩家社区引流。", Thumbnail: "/assets/templates/game-indie.png"},
		{ID: "tpl-ecommerce-fashion", Name: "电商服饰模板", Description: "适合服饰、美妆、潮流品牌电商首页。", Thumbnail: "/assets/templates/ecommerce-fashion.png"},
		{ID: "tpl-ecommerce-digital", Name: "电商数码模板", Description: "适合数码、家电和硬件销售站点。", Thumbnail: "/assets/templates/ecommerce-digital.png"},
		{ID: "tpl-ecommerce-local", Name: "本地商城模板", Description: "适合同城团购、本地零售和配送展示。", Thumbnail: "/assets/templates/ecommerce-local.png"},
		{ID: "tpl-blog-personal", Name: "个人博客模板", Description: "适合个人博客、品牌专栏和知识沉淀。", Thumbnail: "/assets/templates/blog-personal.png"},
		{ID: "tpl-blog-tech", Name: "技术博客模板", Description: "适合技术分享、开发教程与产品日志。", Thumbnail: "/assets/templates/blog-tech.png"},
		{ID: "tpl-blog-media", Name: "媒体专栏模板", Description: "适合媒体评论、专栏作者和订阅型内容站。", Thumbnail: "/assets/templates/blog-media.png"},
		{ID: "tpl-technology-saas", Name: "SaaS 科技模板", Description: "适合 SaaS 平台、云服务和数字产品官网。", Thumbnail: "/assets/templates/technology-saas.png"},
		{ID: "tpl-technology-ai", Name: "AI 科技模板", Description: "适合 AI 产品、自动化服务和模型能力展示。", Thumbnail: "/assets/templates/technology-ai.png"},
		{ID: "tpl-technology-devtool", Name: "开发工具模板", Description: "适合开发者平台、API 服务和基础设施产品。", Thumbnail: "/assets/templates/technology-devtool.png"},
		{ID: "tpl-science-lab", Name: "实验室模板", Description: "适合实验室、研究机构与课题组官网。", Thumbnail: "/assets/templates/science-lab.png"},
		{ID: "tpl-science-research", Name: "科研项目模板", Description: "适合科研专题、成果展示和学术合作。", Thumbnail: "/assets/templates/science-research.png"},
		{ID: "tpl-science-education", Name: "科普教育模板", Description: "适合科普平台、青少年教育和机构展示。", Thumbnail: "/assets/templates/science-education.png"},
		{ID: "tpl-outsourcing-agency", Name: "外包机构模板", Description: "适合综合外包、项目承接和商务服务。", Thumbnail: "/assets/templates/outsourcing-agency.png"},
		{ID: "tpl-outsourcing-software", Name: "软件外包模板", Description: "适合软件开发、定制系统和交付团队展示。", Thumbnail: "/assets/templates/outsourcing-software.png"},
		{ID: "tpl-outsourcing-design", Name: "设计外包模板", Description: "适合设计工作室、视觉服务和案例展示。", Thumbnail: "/assets/templates/outsourcing-design.png"},
		{ID: "tpl-music-artist", Name: "音乐人模板", Description: "适合歌手、乐队和作品推广页面。", Thumbnail: "/assets/templates/music-artist.png"},
		{ID: "tpl-music-label", Name: "厂牌模板", Description: "适合音乐厂牌、发行机构和版权合作。", Thumbnail: "/assets/templates/music-label.png"},
		{ID: "tpl-music-festival", Name: "音乐节模板", Description: "适合音乐节、演出档期和票务说明。", Thumbnail: "/assets/templates/music-festival.png"},
		{ID: "tpl-culture-museum", Name: "文博馆模板", Description: "适合博物馆、美术馆和文博机构官网。", Thumbnail: "/assets/templates/culture-museum.png"},
		{ID: "tpl-culture-brand", Name: "文化品牌模板", Description: "适合文化品牌、文创产品和品牌故事。", Thumbnail: "/assets/templates/culture-brand.png"},
		{ID: "tpl-culture-event", Name: "文化活动模板", Description: "适合展演活动、论坛和公共文化项目。", Thumbnail: "/assets/templates/culture-event.png"},
		{ID: "tpl-news-local", Name: "地方资讯模板", Description: "适合地方新闻、政务资讯和民生信息展示。", Thumbnail: "/assets/templates/news-local.png"},
		{ID: "tpl-news-tech", Name: "科技资讯模板", Description: "适合科技媒体、行业快讯和专题报道。", Thumbnail: "/assets/templates/news-tech.png"},
		{ID: "tpl-news-finance", Name: "财经资讯模板", Description: "适合财经媒体、市场观察和研究简报。", Thumbnail: "/assets/templates/news-finance.png"},
		{ID: "tpl-medical-clinic", Name: "门诊医疗模板", Description: "适合诊所、门诊、专科机构站点。", Thumbnail: "/assets/templates/medical-clinic.png"},
		{ID: "tpl-medical-hospital", Name: "医院机构模板", Description: "适合综合医院、专科医院和医疗集团。", Thumbnail: "/assets/templates/medical-hospital.png"},
		{ID: "tpl-medical-telehealth", Name: "互联网医疗模板", Description: "适合在线问诊、健康平台和远程服务。", Thumbnail: "/assets/templates/medical-telehealth.png"},
		{ID: "tpl-education-school", Name: "学校机构模板", Description: "适合学校、培训机构和校区介绍。", Thumbnail: "/assets/templates/education-school.png"},
		{ID: "tpl-education-course", Name: "课程招生模板", Description: "适合招生页、课程介绍和在线报名。", Thumbnail: "/assets/templates/education-course.png"},
		{ID: "tpl-education-vocational", Name: "职业教育模板", Description: "适合职业培训、认证机构和技能提升服务。", Thumbnail: "/assets/templates/education-vocational.png"},
		{ID: "tpl-finance-bank", Name: "金融服务模板", Description: "适合银行、理财、保险与持牌业务官网。", Thumbnail: "/assets/templates/finance-bank.png"},
		{ID: "tpl-finance-advisory", Name: "投顾咨询模板", Description: "适合财务顾问、基金顾问和企业金融服务。", Thumbnail: "/assets/templates/finance-advisory.png"},
		{ID: "tpl-finance-fintech", Name: "金融科技模板", Description: "适合支付、风控、账务和 fintech 产品站。", Thumbnail: "/assets/templates/finance-fintech.png"},
	}
}

func templateDomain(templateID string) string {
	parts := strings.Split(templateID, "-")
	if len(parts) >= 3 {
		return parts[1]
	}
	if templateID == "tpl-product" {
		return "technology"
	}
	return "enterprise"
}

func templateCopy(templateID, siteName string) (string, string, string) {
	switch templateDomain(templateID) {
	case "game":
		return siteName + " 官方站点", "快速搭建游戏下载、版本公告、活动专题与未成年人保护说明页面。", "查看版本动态"
	case "ecommerce":
		return siteName + " 精选商城", "适合商品展示、支付配送说明、售后与备案信息统一公示。", "立即选购"
	case "blog":
		return siteName + " 内容主页", "适合专栏、文章归档、作者介绍与订阅引导。", "查看最新文章"
	case "technology":
		return siteName + " 科技方案", "展示产品能力、行业解决方案、客户案例与接入流程。", "申请演示"
	case "science":
		return siteName + " 研究平台", "用于科研机构、实验室、课题组和成果项目展示。", "查看研究方向"
	case "outsourcing":
		return siteName + " 服务中心", "快速建立外包服务官网，展示团队能力、交付流程与案例。", "获取报价"
	case "music":
		return siteName + " 音乐主页", "适合作品发布、演出安排、版权合作与艺人介绍。", "查看作品"
	case "culture":
		return siteName + " 文化专题", "适合文化品牌、文博机构、展演活动和预约公示。", "查看活动安排"
	case "news":
		return siteName + " 资讯频道", "适合新闻门户、专题报道、栏目频道与版权信息展示。", "阅读头条"
	case "medical":
		return siteName + " 医疗服务", "适合医疗机构展示资质、科室、医生与就诊流程。", "预约咨询"
	case "education":
		return siteName + " 教育主页", "适合课程介绍、招生信息、校区展示与监管信息披露。", "查看课程"
	case "finance":
		return siteName + " 金融服务", "适合金融业务说明、持牌信息公示和客户服务入口。", "咨询方案"
	default:
		return siteName + " 欢迎页", "这是自动套用模板生成的首页，可直接用于公司介绍与备案验证。", "立即联系"
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
	if compliance.ReviewHistory == nil {
		compliance.ReviewHistory = []ComplianceEvent{}
	}
	return compliance
}
