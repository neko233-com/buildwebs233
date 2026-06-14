package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Auth      AuthConfig      `yaml:"auth"`
	Storage   StorageConfig   `yaml:"storage"`
	HotReload HotReloadConfig `yaml:"hot_reload"`
}

type ServerConfig struct {
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	StaticDir          string `yaml:"static_dir"`
	ViteDevURL         string `yaml:"vite_dev_url"`
	ReadTimeoutSeconds  int    `yaml:"read_timeout_seconds"`
	WriteTimeoutSeconds int    `yaml:"write_timeout_seconds"`
}

type AuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type StorageConfig struct {
	DataDir       string `yaml:"data_dir"`
	PagesFile     string `yaml:"pages_file"`
	TemplatesFile string `yaml:"templates_file"`
}

type HotReloadConfig struct {
	Enabled   bool     `yaml:"enabled"`
	WatchPath []string `yaml:"watch_paths"`
}

func Load(path string) (Config, error) {
	cfg := defaultConfig()
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	cfg.fillDefaults()
	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Host:               "0.0.0.0",
			Port:               6640,
			StaticDir:          "web",
			ViteDevURL:         "http://127.0.0.1:5173",
			ReadTimeoutSeconds:  30,
			WriteTimeoutSeconds: 30,
		},
		Auth: AuthConfig{
			Username: "root",
			Password: "root",
		},
		Storage: StorageConfig{
			DataDir:       "data",
			PagesFile:     "pages.json",
			TemplatesFile: "templates.json",
		},
		HotReload: HotReloadConfig{
			Enabled: true,
			WatchPath: []string{
				"server.yaml",
			},
		},
	}
}

func (c *Config) fillDefaults() {
	d := defaultConfig()
	if c.Server.Host == "" {
		c.Server.Host = d.Server.Host
	}
	if c.Server.Port == 0 {
		c.Server.Port = d.Server.Port
	}
	if c.Server.StaticDir == "" {
		c.Server.StaticDir = d.Server.StaticDir
	}
	if c.Server.ReadTimeoutSeconds == 0 {
		c.Server.ReadTimeoutSeconds = d.Server.ReadTimeoutSeconds
	}
	if c.Server.WriteTimeoutSeconds == 0 {
		c.Server.WriteTimeoutSeconds = d.Server.WriteTimeoutSeconds
	}
	if c.Auth.Username == "" {
		c.Auth.Username = d.Auth.Username
	}
	if c.Auth.Password == "" {
		c.Auth.Password = d.Auth.Password
	}
	if c.Storage.DataDir == "" {
		c.Storage.DataDir = d.Storage.DataDir
	}
	if c.Storage.PagesFile == "" {
		c.Storage.PagesFile = d.Storage.PagesFile
	}
	if c.Storage.TemplatesFile == "" {
		c.Storage.TemplatesFile = d.Storage.TemplatesFile
	}
	if len(c.HotReload.WatchPath) == 0 {
		c.HotReload.WatchPath = d.HotReload.WatchPath
	}
}

func (c *Config) ReadTimeout() time.Duration {
	return time.Duration(c.Server.ReadTimeoutSeconds) * time.Second
}

func (c *Config) WriteTimeout() time.Duration {
	return time.Duration(c.Server.WriteTimeoutSeconds) * time.Second
}

func (c *Config) PagesPath() string {
	return c.Storage.DataDir + "/" + c.Storage.PagesFile
}

func (c *Config) TemplatesPath() string {
	return c.Storage.DataDir + "/" + c.Storage.TemplatesFile
}

