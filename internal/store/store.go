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
	ID      string            `json:"id"`
	Type    string            `json:"type"`
	Content string            `json:"content"`
	Style   map[string]string `json:"style,omitempty"`
}

type Page struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Blocks    []Block   `json:"blocks"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Template struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Thumbnail   string `json:"thumbnail"`
}

type DiskStore struct {
	mu          sync.RWMutex
	pagesPath   string
	templatesPath string
	pages       map[string]*Page
	templates   map[string]*Template
}

type pagesFile struct {
	Items []Page `json:"items"`
}

type templatesFile struct {
	Items []Template `json:"items"`
}

func NewDiskStore(cfg config.StorageConfig) (*DiskStore, error) {
	s := &DiskStore{
		pagesPath:    filepath.Join(cfg.DataDir, cfg.PagesFile),
		templatesPath: filepath.Join(cfg.DataDir, cfg.TemplatesFile),
		pages:        map[string]*Page{},
		templates:    map[string]*Template{},
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, err
	}
	if err := s.loadPages(); err != nil {
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
	p.Slug = strings.TrimSpace(p.Slug)
	if p.Slug == "" {
		p.Slug = slugify(p.Name)
	}
	p.Slug = ensureUniqueSlug(p.Slug, s.pages, p.ID)
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = time.Now()
	}
	s.pages[p.ID] = &p
	if err := s.savePages(); err != nil {
		return nil, err
	}
	return &p, nil
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
