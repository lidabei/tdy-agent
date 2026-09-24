package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port               string
	DSN                string
	JWTSecret          string
	Mode               string
	CORSOrigins        []string
	UploadDir          string
	StaticDir          string
	PasswordMinLen     int
	WithdrawMaxFen     int64
	WithdrawDailyMaxFen int64
	MaxOpenConns       int
	MaxIdleConns       int
}

type fileConfig struct {
	Database struct {
		DBType       string `yaml:"dbtype"`
		Host         string `yaml:"host"`
		Name         string `yaml:"name"`
		Password     string `yaml:"password"`
		Port         int    `yaml:"port"`
		Username     string `yaml:"username"`
		MaxOpenConns int    `yaml:"max_open_conns"`
		MaxIdleConns int    `yaml:"max_idle_conns"`
	} `yaml:"database"`
	Server struct {
		Port           interface{} `yaml:"port"`
		JWTSecret      string      `yaml:"jwt_secret"`
		Mode           string      `yaml:"mode"`
		CORSOrigins    []string    `yaml:"cors_origins"`
		UploadDir      string      `yaml:"upload_dir"`
		StaticDir      string      `yaml:"static_dir"`
		PasswordMinLen int         `yaml:"password_min_len"`
	} `yaml:"server"`
	Business struct {
		WithdrawMaxYuan      float64 `yaml:"withdraw_max_yuan"`
		WithdrawDailyMaxYuan float64 `yaml:"withdraw_daily_max_yuan"`
	} `yaml:"business"`
}

func Load() Config {
	fc := loadYAML()

	host := orDefault(fc.Database.Host, "127.0.0.1")
	user := orDefault(fc.Database.Username, "postgres")
	pass := fc.Database.Password
	dbName := orDefault(fc.Database.Name, "tdy_manager")
	port := fc.Database.Port
	if port == 0 {
		port = 5432
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
		host, user, pass, dbName, port,
	)
	if v := firstNonEmpty(os.Getenv("PG_DSN"), os.Getenv("DATABASE_DSN")); v != "" {
		dsn = v
	}

	srvPort := "8080"
	if p := fmt.Sprint(fc.Server.Port); p != "" && p != "<nil>" {
		srvPort = p
	}
	if v := os.Getenv("PORT"); v != "" {
		srvPort = v
	}

	secret := orDefault(fc.Server.JWTSecret, "tdy-manager-dev-secret-change-me")
	if v := os.Getenv("JWT_SECRET"); v != "" {
		secret = v
	}

	mode := strings.ToLower(orDefault(fc.Server.Mode, "debug"))
	if v := os.Getenv("GIN_MODE"); v != "" {
		mode = v
	}

	origins := fc.Server.CORSOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:5173", "http://127.0.0.1:5173", "http://localhost:8080", "http://127.0.0.1:8080"}
	}
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		origins = splitCSV(v)
	}

	uploadDir := orDefault(fc.Server.UploadDir, "uploads")
	staticDir := orDefault(fc.Server.StaticDir, "../frontend/dist")
	pwdMin := fc.Server.PasswordMinLen
	if pwdMin < 6 {
		pwdMin = 6
	}

	wMax := fc.Business.WithdrawMaxYuan
	if wMax <= 0 {
		wMax = 50000
	}
	wDaily := fc.Business.WithdrawDailyMaxYuan
	if wDaily <= 0 {
		wDaily = 100000
	}

	maxOpen := fc.Database.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 30
	}
	maxIdle := fc.Database.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 10
	}

	return Config{
		Port:                srvPort,
		DSN:                 dsn,
		JWTSecret:           secret,
		Mode:                mode,
		CORSOrigins:         origins,
		UploadDir:           uploadDir,
		StaticDir:           staticDir,
		PasswordMinLen:      pwdMin,
		WithdrawMaxFen:      int64(wMax*100 + 0.5),
		WithdrawDailyMaxFen: int64(wDaily*100 + 0.5),
		MaxOpenConns:        maxOpen,
		MaxIdleConns:        maxIdle,
	}
}

func loadYAML() fileConfig {
	var fc fileConfig
	paths := []string{os.Getenv("CONFIG_FILE"), "config.yaml", "backend/config.yaml"}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "config.yaml"))
	}
	for _, p := range paths {
		if p == "" {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if yaml.Unmarshal(b, &fc) == nil {
			return fc
		}
	}
	return fc
}

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func GetenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
