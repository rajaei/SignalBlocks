package main

import (
	"context"
	//"log"
	//"os"

	"github.com/rajaei/SignalBlocks/internal/config"
	"github.com/rajaei/SignalBlocks/internal/eventbus/nats"
	"github.com/rajaei/SignalBlocks/internal/storage"

	"github.com/rs/zerolog"
	zerologger "github.com/rs/zerolog/log"
)

var logger zerolog.Logger

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger = zerologger.With().Timestamp().Logger()
}

func main() {
	logger.Info().Msg("SignalBlocks starting...")

	// لود کانفیگ (اولویت با فایل)
	cfgMgr := config.NewConfigManager("../../internal/config/config.json")
	cfg := cfgMgr.GetConfig()

	// استفاده از cfg برای رفع unused var
	logger.Info().
		Str("env", cfg.Environment).
		Str("nats_url", cfg.NATSURL).
		Str("minio_endpoint", cfg.MinioEndpoint).
		Msg("Config loaded")

	// تست NATS
	natsClient, err := nats.NewClient(nats.Config{
		URL: cfg.NATSURL,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("NATS connection failed")
	}
	defer natsClient.Close()

	logger.Info().Msg("NATS connected")

	// تست MinIO
	minioCfg := storage.MinioConfig{
		Endpoint:  cfg.MinioEndpoint,
		AccessKey: cfg.MinioAccessKey,
		SecretKey: cfg.MinioSecretKey,
		UseSSL:    cfg.MinioUseSSL,
		Bucket:    cfg.MinioBucket,
	}

	minioClient, err := storage.NewMinio(minioCfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("MinIO client error")
	}

	// استفاده ساده از minioClient (برای رفع unused)
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, cfg.MinioBucket)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to check bucket")
	} else if exists {
		logger.Info().Str("bucket", cfg.MinioBucket).Msg("Bucket exists")
	} else {
		logger.Info().Str("bucket", cfg.MinioBucket).Msg("Bucket will be created on first write")
	}

	// نگه داشتن برنامه
	select {}
}