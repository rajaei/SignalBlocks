package main

import (
	"log"
	"os"

	"github.com/rajaei/SignalBlocks/internal/eventbus/nats"
	"github.com/rajaei/SignalBlocks/internal/storage"
	//"github.com/rs/zerolog/log"
)

func main() {
	cfg := storage.MinioConfig{
		Endpoint:  "127.0.0.1:9000",
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		UseSSL:    false,
		Bucket:    "signalblocks",
	}

	client, err := storage.NewMinio(cfg)
	if err != nil {
		log.Fatalf("MinIO client error: %v", err)
	}

	// ساخت فایل تست
	testFile := "test-upload.txt"
	content := []byte("Hello from SignalBlocks - MinIO Test")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		log.Fatalf("Failed to create test file: %v", err)
	}

	// آپلود
	err = storage.UploadFile(client, cfg.Bucket, "test/test-upload.txt", testFile)
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	// پاک کردن فایل تست (اختیاری)
	os.Remove(testFile)

	log.Println("Test upload successful!")

	client1, err := nats.NewClient(nats.Config{
		URL: "nats://127.0.0.1:4222",
	})
	if err != nil {
		log.Fatalf("NATS client error: %v", err)
	}
	defer client1.Close()

	log.Println("NATS test successful!")
}
