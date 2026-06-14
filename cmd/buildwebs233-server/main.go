package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	buildconfig "github.com/neko233-com/buildwebs233/internal/config"
	"github.com/neko233-com/buildwebs233/internal/hotreload"
	"github.com/neko233-com/buildwebs233/internal/server"
	"github.com/neko233-com/buildwebs233/internal/store"
)

func main() {
	cfgPath := flag.String("config", "server.yaml", "path to server.yaml")
	flag.Parse()

	logger := log.New(os.Stdout, "[buildwebs233] ", log.LstdFlags)
	cfgManager, err := buildconfig.NewManager(*cfgPath)
	if err != nil {
		logger.Fatalf("load config failed: %v", err)
	}

	storeRepo, err := store.NewDiskStore(cfgManager.Config().Storage)
	if err != nil {
		logger.Fatalf("init store failed: %v", err)
	}

	hub := hotreload.NewHub()
	app := server.NewApp(cfgManager, storeRepo, hub, logger)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	app.RegisterRoutes(r)

	cfg := cfgManager.Config()
	srv := &http.Server{
		Addr:         cfg.Server.Host + ":" + itoa(cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout(),
		WriteTimeout: cfg.WriteTimeout(),
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if cfg.HotReload.Enabled {
		paths := make([]string, 0, len(cfg.HotReload.WatchPath)+2)
		paths = append(paths, cfg.HotReload.WatchPath...)
		paths = append(paths, cfg.PagesPath(), cfg.TemplatesPath())
		startWatcher(ctx, cfgManager, hub, paths)
	}

	go func() {
		logger.Printf("buildwebs233 server start on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("http server exit: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	logger.Printf("shutdown requested")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Printf("shutdown failed: %v", err)
	}
}

func startWatcher(ctx context.Context, cfgManager *buildconfig.Manager, hub *hotreload.Hub, paths []string) {
	absPaths := normalizeWatchPaths(paths)
	cfgManager.StartWatch(ctx, absPaths, func(_ buildconfig.Config, changed string, err error) {
		if strings.HasSuffix(changed, filepath.Base("server.yaml")) && err != nil {
			loggerWarn("config reload failed", map[string]any{"path": changed, "err": err.Error()})
			hub.Broadcast(hotreload.Event{
				Type:    "config",
				File:    changed,
				Message: "config reload failed: " + err.Error(),
			})
			return
		}
		eventType := "html"
		if strings.HasSuffix(changed, filepath.Base("server.yaml")) || strings.Contains(changed, "server.yaml") {
			eventType = "config"
		}
		hub.Broadcast(hotreload.Event{
			Type:    eventType,
			File:    changed,
			Message: "reloading",
		})
	})
}

func normalizeWatchPaths(paths []string) []string {
	dedup := map[string]struct{}{}
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if p == "" {
			continue
		}
		clean := filepath.Clean(p)
		if _, ok := dedup[clean]; ok {
			continue
		}
		dedup[clean] = struct{}{}
		result = append(result, clean)
	}
	return result
}

func itoa(v int) string {
	return strconv.Itoa(v)
}

func loggerWarn(msg string, kv map[string]any) {
	var b strings.Builder
	b.WriteString(msg)
	for k, v := range kv {
		b.WriteString(" ")
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(toString(v))
	}
	log.Print(b.String())
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case error:
		return t.Error()
	default:
		return "unknown"
	}
}
