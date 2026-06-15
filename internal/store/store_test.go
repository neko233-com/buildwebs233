package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neko233-com/buildwebs233/internal/config"
)

func TestUpsertPageGeneratesUniqueSlugAndPersists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	repo, err := NewDiskStore(config.StorageConfig{
		DataDir:       dir,
		PagesFile:     "pages.json",
		RevisionsFile: "revisions.json",
		SitesFile:     "sites.json",
		TemplatesFile: "templates.json",
		UploadsDir:    "uploads",
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	first, err := repo.UpsertPage(Page{
		SiteID: "site-default",
		Name:   "Landing Page",
		Title:  "Landing Page",
		Blocks: []Block{
			{ID: "b1", Type: "text", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("upsert first: %v", err)
	}

	second, err := repo.UpsertPage(Page{
		SiteID: "site-default",
		Name:   "Landing Page",
		Title:  "Landing Page 2",
	})
	if err != nil {
		t.Fatalf("upsert second: %v", err)
	}

	if first.Slug == "" || second.Slug == "" {
		t.Fatal("expected generated slugs")
	}
	if first.Slug == second.Slug {
		t.Fatalf("expected unique slugs, got %q", first.Slug)
	}
	if len(first.Sections) == 0 {
		t.Fatal("expected blocks to be normalized into sections")
	}
	if first.SchemaVersion != 2 {
		t.Fatalf("expected schema version 2, got %d", first.SchemaVersion)
	}

	pagesPath := filepath.Join(dir, "pages.json")
	if _, err := os.Stat(pagesPath); err != nil {
		t.Fatalf("expected pages file: %v", err)
	}

	templates := repo.ListTemplates()
	if len(templates) < 2 {
		t.Fatalf("expected default templates, got %d", len(templates))
	}

	sites := repo.ListSites()
	if len(sites) == 0 {
		t.Fatal("expected default site")
	}
	if got := repo.ListPagesBySite("site-default"); len(got) != 2 {
		t.Fatalf("expected 2 pages for site-default, got %d", len(got))
	}
	revisions := repo.ListPageRevisions(first.ID)
	if len(revisions) == 0 {
		t.Fatal("expected at least one revision after save")
	}
}

func TestUpsertSitePersistsSiteRecord(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	repo, err := NewDiskStore(config.StorageConfig{
		DataDir:       dir,
		PagesFile:     "pages.json",
		RevisionsFile: "revisions.json",
		SitesFile:     "sites.json",
		TemplatesFile: "templates.json",
		UploadsDir:    "uploads",
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	saved, err := repo.UpsertSite(Site{
		Name:       "华东企业站",
		Domain:     "corp.example.cn",
		Region:     "CN-SH",
		TemplateID: "tpl-product",
		Status:     "draft",
	})
	if err != nil {
		t.Fatalf("upsert site: %v", err)
	}

	if saved.ID == "" {
		t.Fatal("expected generated site id")
	}
	if saved.Region != "CN-SH" {
		t.Fatalf("expected region CN-SH, got %q", saved.Region)
	}
	if saved.Compliance.ICPStatus == "" || saved.Compliance.PSBStatus == "" {
		t.Fatal("expected compliance defaults")
	}
	if _, err := os.Stat(filepath.Join(dir, "sites.json")); err != nil {
		t.Fatalf("expected sites file: %v", err)
	}
}

func TestTemplateHomepagePublishAndComplianceFlow(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	repo, err := NewDiskStore(config.StorageConfig{
		DataDir:       dir,
		PagesFile:     "pages.json",
		RevisionsFile: "revisions.json",
		SitesFile:     "sites.json",
		TemplatesFile: "templates.json",
		UploadsDir:    "uploads",
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	page, site, err := repo.ApplyTemplateToSite("site-default", "tpl-product")
	if err != nil {
		t.Fatalf("apply template: %v", err)
	}
	if page.TemplateID != "tpl-product" || site.TemplateID != "tpl-product" {
		t.Fatal("expected template applied to site and page")
	}

	published, err := repo.PublishPage(page.ID)
	if err != nil {
		t.Fatalf("publish page: %v", err)
	}
	if published.Status != "published" {
		t.Fatalf("expected published status, got %q", published.Status)
	}

	updatedSite, err := repo.SetSitePrimaryPage("site-default", page.ID)
	if err != nil {
		t.Fatalf("set homepage: %v", err)
	}
	if updatedSite.PrimaryPageID != page.ID {
		t.Fatalf("expected primary page id %q, got %q", page.ID, updatedSite.PrimaryPageID)
	}

	material, reviewedSite, err := repo.SaveComplianceMaterial("site-default", "business-license", "license.pdf", []byte("file"))
	if err != nil {
		t.Fatalf("save compliance material: %v", err)
	}
	if material.PublicURL == "" || len(reviewedSite.Compliance.Materials) == 0 {
		t.Fatal("expected stored material metadata")
	}

	reviewedSite, err = repo.ReviewCompliance("site-default", "mark_material_verified", "ok", material.ID)
	if err != nil {
		t.Fatalf("review material: %v", err)
	}
	if reviewedSite.Compliance.Materials[0].Status != "verified" {
		t.Fatalf("expected verified material, got %q", reviewedSite.Compliance.Materials[0].Status)
	}
}
