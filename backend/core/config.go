package core

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const Version = "2.0.0-go"

type Settings struct {
	Port                          int
	AdminKey                      string
	BrowserPoolSize               int
	MaxInflightPerAccount         int
	BrowserStreamTimeoutSeconds   int
	AccountMinIntervalMS          int
	RequestJitterMinMS            int
	RequestJitterMaxMS            int
	RateLimitBaseCooldown         int
	RateLimitMaxCooldown          int
	AccountReadySetThreshold      int
	ChatIDPrewarmTargetPerAccount int
	ChatIDPrewarmTTLSeconds       int
	ChatIDPrewarmMaxConcurrency   int
	LogLevel                      string
	BaseDir                       string
	DataDir                       string
	LogsDir                       string
	AccountsFile                  string
	UsersFile                     string
	APIKeysFile                   string
	FrontendDist                  string
}

func LoadSettings(baseDir string) Settings {
	base := strings.TrimSpace(baseDir)
	if base == "" {
		base = "."
	}
	if v := strings.TrimSpace(os.Getenv("BASE_DIR")); v != "" {
		base = v
	}
	if abs, err := filepath.Abs(base); err == nil {
		base = abs
	}
	data := EnvString("DATA_DIR", filepath.Join(base, "data"))
	logs := EnvString("LOGS_DIR", filepath.Join(base, "logs"))
	return Settings{
		Port:                          EnvInt("PORT", 7860),
		AdminKey:                      EnvString("ADMIN_KEY", ""),
		BrowserPoolSize:               EnvInt("BROWSER_POOL_SIZE", 1),
		MaxInflightPerAccount:         EnvIntAlias("MAX_INFLIGHT_PER_ACCOUNT", "MAX_INFLIGHT", 2),
		BrowserStreamTimeoutSeconds:   EnvInt("BROWSER_STREAM_TIMEOUT_SECONDS", 1800),
		AccountMinIntervalMS:          EnvInt("ACCOUNT_MIN_INTERVAL_MS", 0),
		RequestJitterMinMS:            EnvInt("REQUEST_JITTER_MIN_MS", 0),
		RequestJitterMaxMS:            EnvInt("REQUEST_JITTER_MAX_MS", 0),
		RateLimitBaseCooldown:         EnvInt("RATE_LIMIT_BASE_COOLDOWN", 600),
		RateLimitMaxCooldown:          EnvInt("RATE_LIMIT_MAX_COOLDOWN", 3600),
		AccountReadySetThreshold:      EnvInt("ACCOUNT_READY_SET_THRESHOLD", 128),
		ChatIDPrewarmTargetPerAccount: EnvInt("CHAT_ID_PREWARM_TARGET_PER_ACCOUNT", 0),
		ChatIDPrewarmTTLSeconds:       EnvInt("CHAT_ID_PREWARM_TTL_SECONDS", 120),
		ChatIDPrewarmMaxConcurrency:   EnvInt("CHAT_ID_PREWARM_MAX_CONCURRENCY", 16),
		LogLevel:                      EnvString("LOG_LEVEL", "INFO"),
		BaseDir:                       base,
		DataDir:                       data,
		LogsDir:                       logs,
		AccountsFile:                  EnvString("ACCOUNTS_FILE", filepath.Join(data, "accounts.json")),
		UsersFile:                     EnvString("USERS_FILE", filepath.Join(data, "users.json")),
		APIKeysFile:                   EnvString("API_KEYS_FILE", filepath.Join(data, "api_keys.json")),
		FrontendDist:                  filepath.Join(base, "frontend", "dist"),
	}
}

func EnvString(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func EnvInt(key string, fallback int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func EnvIntAlias(key, alias string, fallback int) int {
	if v := EnvInt(key, fallback); v != fallback {
		return v
	}
	return EnvInt(alias, fallback)
}
