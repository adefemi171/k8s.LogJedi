package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds operator configuration from environment.
type Config struct {
	LLMServiceURL         string
	LogBackendType        string
	LogBackendURL         string
	ApplyMode             string   // "auto" | "manual"
	SlackWebhookURL       string
	SlackBotToken         string
	TeamsWebhookURL       string
	LogTailLines          int
	HistoricalLogSince    time.Duration
	LLMClientTimeout      time.Duration
	LLMClientMaxRetries   int
	CooldownDuration      time.Duration // do not re-analyze same resource within this window
	DryRunBeforeApply     bool          // run server-side dry-run before applying patch
	MaxRecentLogLines     int           // cap recent_logs sent to LLM (0 = no cap)
	MaxHistoricalLogLines int           // cap historical_logs (0 = no cap)
	WatchNamespaces       []string      // empty = all; otherwise only these namespaces
	ExcludeNamespaces     []string      // never watch these namespaces
	AutoApplyNamespaces   []string      // when auto, only apply in these namespaces (empty = all)
	LLMServiceAuthHeader  string        // optional header value for operator->LLM auth (e.g. Bearer token)
}

// Load reads configuration from environment variables.
func Load() *Config {
	c := &Config{
		LLMServiceURL:        getEnv("LLM_SERVICE_URL", "http://llm-service:8000"),
		LogBackendType:        getEnv("LOG_BACKEND_TYPE", "kubernetes"),
		LogBackendURL:         getEnv("LOG_BACKEND_URL", ""),
		ApplyMode:             getEnv("APPLY_MODE", "manual"),
		SlackWebhookURL:       getEnv("SLACK_WEBHOOK_URL", ""),
		SlackBotToken:         getEnv("SLACK_BOT_TOKEN", ""),
		TeamsWebhookURL:       getEnv("TEAMS_WEBHOOK_URL", ""),
		LogTailLines:          getEnvInt("LOG_TAIL_LINES", 100),
		LLMClientTimeout:      getEnvDuration("LLM_CLIENT_TIMEOUT", 30*time.Second),
		LLMClientMaxRetries:    getEnvInt("LLM_CLIENT_MAX_RETRIES", 3),
		DryRunBeforeApply:     getEnvBool("DRY_RUN_BEFORE_APPLY", false),
		MaxRecentLogLines:     getEnvInt("MAX_RECENT_LOG_LINES", 0),
		MaxHistoricalLogLines: getEnvInt("MAX_HISTORICAL_LOG_LINES", 0),
		LLMServiceAuthHeader:  getEnv("LLM_SERVICE_AUTH_HEADER", ""),
	}
	sinceMin := getEnvInt("HISTORICAL_LOG_SINCE_MINUTES", 60)
	c.HistoricalLogSince = time.Duration(sinceMin) * time.Minute
	cooldownMin := getEnvInt("ANALYZE_COOLDOWN_MINUTES", 15)
	c.CooldownDuration = time.Duration(cooldownMin) * time.Minute
	c.WatchNamespaces = getEnvSlice("WATCH_NAMESPACES", ",")
	c.ExcludeNamespaces = getEnvSlice("EXCLUDE_NAMESPACES", ",")
	c.AutoApplyNamespaces = getEnvSlice("AUTO_APPLY_NAMESPACES", ",")
	return c
}

func getEnvBool(key string, def bool) bool {
	v := getEnv(key, "")
	if v == "" {
		return def
	}
	return strings.EqualFold(v, "true") || v == "1"
}

func getEnvSlice(key, sep string) []string {
	v := getEnv(key, "")
	if v == "" {
		return nil
	}
	parts := strings.Split(v, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return strings.TrimSpace(v)
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(strings.TrimSpace(v)); err == nil {
			return d
		}
	}
	return def
}
