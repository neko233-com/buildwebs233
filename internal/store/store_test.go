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
		TemplatesFile: "templates.json",
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	first, err := repo.UpsertPage(Page{
		Name:  "Landing Page",
		Title: "Landing Page",
		Blocks: []Block{
			{ID: "b1", Type: "text", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("upsert first: %v", err)
	}

	second, err := repo.UpsertPage(Page{
		Name:  "Landing Page",
		Title: "Landing Page 2",
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

	pagesPath := filepath.Join(dir, "pages.json")
	if _, err := os.Stat(pagesPath); err != nil {
		t.Fatalf("expected pages file: %v", err)
	}

	templates := repo.ListTemplates()
	if len(templates) < 2 {
		t.Fatalf("expected default templates, got %d", len(templates))
	}
}
