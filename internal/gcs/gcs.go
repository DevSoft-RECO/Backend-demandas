package gcs

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"cloud.google.com/go/storage"
	"github.com/DevSoft-RECO/backend-creditos-go/internal/config"
	"google.golang.org/api/option"
)

var client *storage.Client

func InitGCS() error {
	if client != nil {
		return nil
	}
	ctx := context.Background()

	var err error
	if config.Envs.GcsKeyFile != "" {
		client, err = storage.NewClient(ctx, option.WithCredentialsFile(config.Envs.GcsKeyFile))
	} else {
		client, err = storage.NewClient(ctx)
	}
	if err != nil {
		return fmt.Errorf("error inicializando cliente GCS: %v", err)
	}
	return nil
}

// UploadFile sube un archivo a Google Cloud Storage
func UploadFile(fileHeader *multipart.FileHeader, destPath string) (string, error) {
	if err := InitGCS(); err != nil {
		return "", err
	}

	ctx := context.Background()
	bucket := client.Bucket(config.Envs.GcsBucketName)
	obj := bucket.Object(destPath)

	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	wc := obj.NewWriter(ctx)
	wc.ContentType = fileHeader.Header.Get("Content-Type")

	if _, err := io.Copy(wc, file); err != nil {
		return "", err
	}
	if err := wc.Close(); err != nil {
		return "", err
	}

	return destPath, nil
}

// GenerateSignedURL genera un enlace temporal firmado para acceder a un documento
func GenerateSignedURL(objectName string) (string, error) {
	if err := InitGCS(); err != nil {
		return "", err
	}

	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(1 * time.Hour), // Expira en 1 hora
	}

	// Como el cliente está autenticado vía Service Account (key file),
	// Bucket().SignedURL() usa esas mismas credenciales.
	bucket := client.Bucket(config.Envs.GcsBucketName)
	url, err := bucket.SignedURL(objectName, opts)
	if err != nil {
		return "", err
	}

	return url, nil
}
