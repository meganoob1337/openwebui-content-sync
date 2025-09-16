package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	LogLevel   string           `yaml:"log_level"`
	Schedule   ScheduleConfig   `yaml:"schedule"`
	Storage    StorageConfig    `yaml:"storage"`
	OpenWebUI  OpenWebUIConfig  `yaml:"openwebui"`
	GitHub     GitHubConfig     `yaml:"github"`
	Confluence ConfluenceConfig `yaml:"confluence"`
}

// ScheduleConfig defines the sync schedule
type ScheduleConfig struct {
	Interval time.Duration `yaml:"interval"`
}

// StorageConfig defines local storage settings
type StorageConfig struct {
	Path string `yaml:"path"`
}

// OpenWebUIConfig defines OpenWebUI API settings
type OpenWebUIConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
}

// GitHubConfig defines GitHub adapter settings
type GitHubConfig struct {
	Enabled      bool     `yaml:"enabled"`
	Token        string   `yaml:"token"`
	Repositories []string `yaml:"repositories"`
	KnowledgeID  string   `yaml:"knowledge_id"`
}

// ConfluenceConfig defines Confluence adapter settings
type ConfluenceConfig struct {
	Enabled            bool     `yaml:"enabled"`
	BaseURL            string   `yaml:"base_url"`
	Username           string   `yaml:"username"`
	APIKey             string   `yaml:"api_key"`
	Spaces             []string `yaml:"spaces"`
	ParentPageIDs      []string `yaml:"parent_page_ids"` // Optional: specific parent pages to process sub-pages
	KnowledgeID        string   `yaml:"knowledge_id"`
	PageLimit          int      `yaml:"page_limit"`
	IncludeAttachments bool     `yaml:"include_attachments"`
}

// Load loads configuration from file and environment variables
func Load(path string) (*Config, error) {
	fmt.Printf("Loading configuration from: %s\n", path)

	cfg := &Config{
		LogLevel: "info",
		Schedule: ScheduleConfig{
			Interval: 1 * time.Hour,
		},
		Storage: StorageConfig{
			Path: "/data",
		},
		OpenWebUI: OpenWebUIConfig{
			BaseURL: getEnv("OPENWEBUI_BASE_URL", "http://localhost:8080"),
			APIKey:  getEnv("OPENWEBUI_API_KEY", ""),
		},
		GitHub: GitHubConfig{
			Enabled:      false,
			Token:        getEnv("GITHUB_TOKEN", ""),
			Repositories: []string{},
			KnowledgeID:  getEnv("GITHUB_KNOWLEDGE_ID", ""),
		},
		Confluence: ConfluenceConfig{
			Enabled:            false,
			BaseURL:            "",
			Username:           "",
			APIKey:             getEnv("CONFLUENCE_API_KEY", ""),
			Spaces:             []string{},
			ParentPageIDs:      []string{},
			KnowledgeID:        "",
			PageLimit:          100,
			IncludeAttachments: true,
		},
	}

	fmt.Printf("Default OpenWebUI BaseURL: %s\n", cfg.OpenWebUI.BaseURL)
	fmt.Printf("Confluence API Key loaded: %s\n", func() string {
		if cfg.Confluence.APIKey != "" {
			return "***" + cfg.Confluence.APIKey[len(cfg.Confluence.APIKey)-4:] // Show last 4 chars
		}
		return "NOT SET"
	}())

	// Load from file if it exists
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("Config file exists, loading from: %s\n", path)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		// fmt.Printf("Config file content:\n%s\n", string(data))

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}

		fmt.Printf("After loading config file - OpenWebUI BaseURL: %s\n", cfg.OpenWebUI.BaseURL)
	} else {
		fmt.Printf("Config file does not exist at: %s (error: %v)\n", path, err)
	}

	// Override with environment variables
	cfg.OpenWebUI.BaseURL = getEnv("OPENWEBUI_BASE_URL", cfg.OpenWebUI.BaseURL)
	cfg.OpenWebUI.APIKey = getEnv("OPENWEBUI_API_KEY", cfg.OpenWebUI.APIKey)
	cfg.GitHub.Token = getEnv("GITHUB_TOKEN", cfg.GitHub.Token)
	cfg.GitHub.KnowledgeID = getEnv("GITHUB_KNOWLEDGE_ID", cfg.GitHub.KnowledgeID)
	cfg.Confluence.APIKey = getEnv("CONFLUENCE_API_KEY", cfg.Confluence.APIKey)
	cfg.Storage.Path = getEnv("STORAGE_PATH", cfg.Storage.Path)

	fmt.Printf("Final OpenWebUI BaseURL: %s\n", cfg.OpenWebUI.BaseURL)
	fmt.Printf("Environment OPENWEBUI_BASE_URL: %s\n", os.Getenv("OPENWEBUI_BASE_URL"))
	fmt.Printf("Final Confluence API Key: %s\n", func() string {
		if cfg.Confluence.APIKey != "" {
			return "***" + cfg.Confluence.APIKey[len(cfg.Confluence.APIKey)-4:] // Show last 4 chars
		}
		return "NOT SET"
	}())
	fmt.Printf("Environment CONFLUENCE_API_KEY: %s\n", func() string {
		env := os.Getenv("CONFLUENCE_API_KEY")
		if env != "" {
			return "***" + env[len(env)-4:] // Show last 4 chars
		}
		return "NOT SET"
	}())

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
