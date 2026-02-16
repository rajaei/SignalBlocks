package storage

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioConfig - تنظیمات اتصال (از env می‌خونی)
type MinioConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Bucket    string
}

// NewMinio - ساخت کلاینت MinIO
func NewMinio(cfg MinioConfig) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// چک bucket وجود داشته باشه (اگر نه، بساز)
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}
	if !exists {
		err = client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket %q: %w", cfg.Bucket, err)
		}
		log.Printf("Bucket %q created", cfg.Bucket)
	}

	return client, nil
}

// UploadFile - آپلود یک فایل به MinIO
func UploadFile(client *minio.Client, bucket, objectName, filePath string) error {
	ctx := context.Background()

	// باز کردن فایل
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// اطلاعات فایل
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// آپلود
	info, err := client.PutObject(ctx, bucket, objectName, file, fileInfo.Size(), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	log.Printf("Uploaded %s to %s (ETag: %s)", objectName, bucket, info.ETag)
	return nil
}