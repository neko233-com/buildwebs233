package store

import (
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
	CompanyName     string           `json:"company_name,omitempty"`
	ContactName     string           `json:"contact_name,omitempty"`
	ContactPhone    string           `json:"contact_phone,omitempty"`
	RegionRule      string           `json:"region_rule,omitempty"`
	ICPStatus       string           `json:"icp_status,omitempty"`
	PSBStatus       string           `json:"psb_status,omitempty"`
	ReviewStatus    string           `json:"review_status,omitempty"`
	ReviewNotes     string           `json:"review_notes,omitempty"`
	Checklist       []ComplianceItem `json:"checklist,omitempty"`
	LastReviewedAt  *time.Time       `json:"last_reviewed_at,omitempty"`
	LastSubmittedAt *time.Time       `json:"last_submitted_at,omitempty"`
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

type DiskStore struct {
	mu            sync.RWMutex
	pagesPath     string
	sitesPath     string
	templatesPath string
	pages         map[string]*Page
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

func NewDiskStore(cfg config.StorageConfig) (*DiskStore, error) {
	s := &DiskStore{
		pagesPath:     filepath.Join(cfg.DataDir, cfg.PagesFile),
		sitesPath:     filepath.Join(cfg.DataDir, cfg.SitesFile),
		templatesPath: filepath.Join(cfg.DataDir, cfg.TemplatesFile),
		pages:         map[string]*Page{},
		sites:         map[string]*Site{},
		templates:     map[string]*Template{},
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, err
	}
	if err := s.loadPages(); err != nil {
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
