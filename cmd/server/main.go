// Package main is the entry point for the DeguBase server.
//
//	@title                      DeguBase API
//	@version                    1.0
//	@description                DeguBase – workspace and table management API.
//	@BasePath                   /api
//	@securityDefinitions.apikey BearerAuth
//	@in                         header
//	@name                       Authorization

//go:generate swag init -g cmd/server/main.go -o docs

package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-co-op/gocron/v2"

	docs "github.com/saintedlama/degubase/docs"
	"github.com/saintedlama/degubase/internal/api"
	"github.com/saintedlama/degubase/internal/infrastructure/storage"
	"github.com/saintedlama/degubase/internal/infrastructure/store"
	"github.com/saintedlama/degubase/internal/jobs"
	"github.com/saintedlama/degubase/internal/records"
	"github.com/saintedlama/degubase/internal/snapshots"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	viper.SetEnvPrefix("DEGUBASE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("host", "localhost")
	viper.SetDefault("port", "8080")
	viper.SetDefault("data_dir", "data")
	viper.SetDefault("config_dir", ".")
	viper.SetDefault("disable_auth", false)
	viper.SetDefault("upload.image.max_bytes", int64(10<<20))
	viper.SetDefault("upload.file.max_bytes", int64(10<<20))
	viper.SetDefault("upload.image.allowed_mimes", []string{"image/jpeg", "image/png", "image/webp", "image/gif"})
	viper.SetDefault("upload.image.thumbnail_width", 256)
	viper.SetDefault("snapshots.max", snapshots.DefaultMaxSnapshots)
	viper.SetDefault("snapshots.schedule", "0 2 * * *")

	if cfgFile := viper.GetString("config"); cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("degubase")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(viper.GetString("config_dir"))
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(fmt.Sprintf("read config: %v", err))
		}
	} else {
		slog.Info("using config file", "path", viper.ConfigFileUsed())
	}

	disableAuth := viper.GetBool("disable_auth")
	jwtSecret := ensureJWTSecret(disableAuth)

	dataDir := viper.GetString("data_dir")
	host := viper.GetString("host")
	port := viper.GetString("port")
	if err := os.MkdirAll(filepath.Join(dataDir, "db"), 0o755); err != nil {
		panic(fmt.Sprintf("create db dir: %v", err))
	}

	dbPath := filepath.Join(dataDir, "db", "degubase.db")
	snapshotDir := filepath.Join(dataDir, "db", "snapshots")

	if err := snapshots.RestoreOnStartup(dbPath, snapshotDir); err != nil {
		slog.Error("snapshot restore FAILED", "error", err)
		slog.Error("=== RECOVERY ===")
		slog.Error("The database may be in an inconsistent state.")
		slog.Error("1. Check for pre-restore backup files in " + filepath.Dir(dbPath))
		slog.Error("2. To roll back: cp <backup-path>* " + dbPath + "* && restart")
		slog.Error("3. Or delete the restore marker and restart to try again")
		slog.Error("================")
		os.Exit(1)
	}

	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", dbPath)

	str, err := store.New(dsn)
	if err != nil {
		panic(fmt.Sprintf("init store: %v", err))
	}

	storagePath := filepath.Join(dataDir, "files")
	absPath, err := filepath.Abs(storagePath)
	if err != nil {
		panic(fmt.Sprintf("resolve storage path: %v", err))
	}
	fileStorage, err := storage.NewLocal(absPath)
	if err != nil {
		panic(fmt.Sprintf("init file storage: %v", err))
	}

	if h := viper.GetString("public_host"); h != "" {
		docs.SwaggerInfo.Host = h
	}

	uploadCfg := records.UploadConfig{
		ImageMaxBytes:     viper.GetInt64("upload.image.max_bytes"),
		FileMaxBytes:      viper.GetInt64("upload.file.max_bytes"),
		AllowedImageMIMEs: viper.GetStringSlice("upload.image.allowed_mimes"),
		ThumbnailWidth:    viper.GetInt("upload.image.thumbnail_width"),
	}

	snapHandler := &snapshots.Handler{
		DB:           str.DB(),
		DBPath:       dbPath,
		SnapshotDir:  snapshotDir,
		MaxSnapshots: viper.GetInt("snapshots.max"),
	}

	jobStore := jobs.NewSQLite(str.DB())
	jobHandler := &jobs.Handler{Store: jobStore}

	// Scheduled snapshots with job run tracking.
	if schedule := viper.GetString("snapshots.schedule"); schedule != "" {
		s, err := gocron.NewScheduler()
		if err != nil {
			panic(fmt.Sprintf("start snapshot scheduler: %v", err))
		}
		if _, err := s.NewJob(
			gocron.CronJob(schedule, false),
			gocron.NewTask(
				snapshots.ScheduledJob(str.DB(), snapshotDir, viper.GetInt("snapshots.max"), jobStore),
			),
		); err != nil {
			panic(fmt.Sprintf("register snapshot job: %v", err))
		}
		s.Start()
		slog.Info("scheduled snapshots enabled", "schedule", schedule)
	}

	router := api.NewRouter(str.DB(), fileStorage, uploadCfg, "ui/dist", jwtSecret, disableAuth, snapHandler, jobHandler)

	addr := fmt.Sprintf("%s:%s", host, port)
	slog.Info("degubase listening", "addr", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		panic(fmt.Sprintf("server: %v", err))
	}
}

// ensureJWTSecret reads jwt_secret from config/env/secret-file, or generates and persists one.
func ensureJWTSecret(disableAuth bool) []byte {
	decode := func(s string) []byte {
		if decoded, err := base64.StdEncoding.DecodeString(s); err == nil {
			return decoded
		}
		return []byte(s)
	}

	// 1. Explicit config / env var takes priority.
	if s := viper.GetString("jwt_secret"); s != "" {
		return decode(s)
	}

	// 2. Generate a new secret.
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("generate jwt_secret: %v", err))
	}

	// Only persist to degubase.yaml when auth is enabled.
	if !disableAuth {
		encoded := base64.StdEncoding.EncodeToString(b)
		configPath := resolveConfigPath()
		if err := upsertYAMLKey(configPath, "jwt_secret", encoded); err != nil {
			slog.Warn("could not save jwt_secret to config", "path", configPath, "error", err)
		} else {
			slog.Info("generated jwt_secret and saved to config", "path", configPath)
		}
	}
	return b
}

// resolveConfigPath returns the path to the degubase.yaml config file.
func resolveConfigPath() string {
	if p := viper.ConfigFileUsed(); p != "" {
		return p
	}
	return filepath.Join(viper.GetString("config_dir"), "degubase.yaml")
}

// upsertYAMLKey sets a single top-level key in a YAML file, preserving all other keys.
func upsertYAMLKey(path, key, value string) error {
	// Read existing file (or start empty).
	var root map[string]any
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, &root); err != nil {
			return fmt.Errorf("parse yaml: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if root == nil {
		root = make(map[string]any)
	}

	root[key] = value

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return fmt.Errorf("encode yaml: %w", err)
	}
	enc.Close()

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return nil
}
