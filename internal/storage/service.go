// =============================================================================
// STORAGE SERVICE
// File upload and management with S3 and local storage support
// =============================================================================

package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// =============================================================================
// TYPES
// =============================================================================

// FileInfo represents uploaded file metadata
type FileInfo struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	OriginalName string   `json:"original_name"`
	Path        string    `json:"path"`
	URL         string    `json:"url"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	Checksum    string    `json:"checksum"`
	Bucket      string    `json:"bucket,omitempty"`
	UploadedBy  uuid.UUID `json:"uploaded_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// UploadOptions for file uploads
type UploadOptions struct {
	Bucket       string
	Path         string // e.g., "vendors/123/photos"
	MaxSize      int64  // Max file size in bytes
	AllowedTypes []string // e.g., ["image/jpeg", "image/png"]
	Private      bool   // If true, requires signed URL to access
	Metadata     map[string]string
}

// StorageProvider interface for different storage backends
type StorageProvider interface {
	Upload(ctx context.Context, file io.Reader, filename string, opts UploadOptions) (*FileInfo, error)
	Download(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	GetURL(ctx context.Context, path string, expiry time.Duration) (string, error)
	Exists(ctx context.Context, path string) (bool, error)
}

// =============================================================================
// SERVICE
// =============================================================================

// Config for storage service
type Config struct {
	Provider     string // "s3", "local"
	
	// S3 config
	S3Bucket     string
	S3Region     string
	S3Endpoint   string // For MinIO or other S3-compatible
	S3AccessKey  string
	S3SecretKey  string
	
	// Local config
	LocalPath    string
	LocalBaseURL string
	
	// Common
	MaxFileSize  int64
	CDNBaseURL   string
}

// Service handles file storage
type Service struct {
	config   *Config
	provider StorageProvider
}

// NewService creates a new storage service
func NewService(ctx context.Context, cfg *Config) (*Service, error) {
	var provider StorageProvider
	var err error
	
	switch cfg.Provider {
	case "s3":
		provider, err = newS3Provider(ctx, cfg)
	case "local":
		provider, err = newLocalProvider(cfg)
	default:
		return nil, fmt.Errorf("unknown storage provider: %s", cfg.Provider)
	}
	
	if err != nil {
		return nil, err
	}
	
	return &Service{
		config:   cfg,
		provider: provider,
	}, nil
}

// =============================================================================
// FILE OPERATIONS
// =============================================================================

// UploadFile uploads a file from multipart form
func (s *Service) UploadFile(ctx context.Context, file *multipart.FileHeader, userID uuid.UUID, opts UploadOptions) (*FileInfo, error) {
	// Validate file size
	maxSize := opts.MaxSize
	if maxSize <= 0 {
		maxSize = s.config.MaxFileSize
	}
	if file.Size > maxSize {
		return nil, fmt.Errorf("file too large: %d > %d bytes", file.Size, maxSize)
	}
	
	// Open file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()
	
	// Detect content type
	buffer := make([]byte, 512)
	n, _ := src.Read(buffer)
	contentType := http.DetectContentType(buffer[:n])
	src.Seek(0, io.SeekStart)
	
	// Validate content type
	if len(opts.AllowedTypes) > 0 {
		allowed := false
		for _, t := range opts.AllowedTypes {
			if strings.HasPrefix(contentType, t) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("file type not allowed: %s", contentType)
		}
	}
	
	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	uniqueName := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	
	// Build path
	path := opts.Path
	if path != "" {
		path = strings.TrimSuffix(path, "/") + "/" + uniqueName
	} else {
		path = uniqueName
	}
	
	// Calculate checksum
	hash := md5.New()
	tee := io.TeeReader(src, hash)
	
	// Read into buffer for upload
	var buf bytes.Buffer
	size, err := io.Copy(&buf, tee)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	checksum := hex.EncodeToString(hash.Sum(nil))
	
	// Upload
	info, err := s.provider.Upload(ctx, &buf, path, opts)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	
	// Populate additional info
	info.ID = uuid.New()
	info.OriginalName = file.Filename
	info.Size = size
	info.ContentType = contentType
	info.Checksum = checksum
	info.UploadedBy = userID
	info.CreatedAt = time.Now()
	
	return info, nil
}

// UploadFromReader uploads a file from an io.Reader
func (s *Service) UploadFromReader(ctx context.Context, reader io.Reader, filename string, size int64, userID uuid.UUID, opts UploadOptions) (*FileInfo, error) {
	// Generate unique filename
	ext := filepath.Ext(filename)
	uniqueName := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	
	// Build path
	path := opts.Path
	if path != "" {
		path = strings.TrimSuffix(path, "/") + "/" + uniqueName
	} else {
		path = uniqueName
	}
	
	info, err := s.provider.Upload(ctx, reader, path, opts)
	if err != nil {
		return nil, err
	}
	
	info.ID = uuid.New()
	info.OriginalName = filename
	info.Size = size
	info.UploadedBy = userID
	info.CreatedAt = time.Now()
	
	return info, nil
}

// Download downloads a file
func (s *Service) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return s.provider.Download(ctx, path)
}

// Delete removes a file
func (s *Service) Delete(ctx context.Context, path string) error {
	return s.provider.Delete(ctx, path)
}

// GetURL returns a URL to access the file
func (s *Service) GetURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	if s.config.CDNBaseURL != "" && expiry == 0 {
		return s.config.CDNBaseURL + "/" + path, nil
	}
	return s.provider.GetURL(ctx, path, expiry)
}

// Exists checks if a file exists
func (s *Service) Exists(ctx context.Context, path string) (bool, error) {
	return s.provider.Exists(ctx, path)
}

// =============================================================================
// S3 PROVIDER
// =============================================================================

type s3Provider struct {
	client *s3.Client
	bucket string
	cdnURL string
}

func newS3Provider(ctx context.Context, cfg *Config) (*s3Provider, error) {
	var awsCfg aws.Config
	var err error
	
	if cfg.S3Endpoint != "" {
		// Custom endpoint (MinIO, etc.)
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cfg.S3Endpoint,
				SigningRegion:     cfg.S3Region,
				HostnameImmutable: true,
			}, nil
		})
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.S3Region),
			config.WithEndpointResolverWithOptions(customResolver),
		)
	} else {
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.S3Region),
		)
	}
	
	if err != nil {
		return nil, err
	}
	
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.S3Endpoint != "" // Use path style for custom endpoints
	})
	
	return &s3Provider{
		client: client,
		bucket: cfg.S3Bucket,
		cdnURL: cfg.CDNBaseURL,
	}, nil
}

func (p *s3Provider) Upload(ctx context.Context, file io.Reader, path string, opts UploadOptions) (*FileInfo, error) {
	bucket := opts.Bucket
	if bucket == "" {
		bucket = p.bucket
	}
	
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
		Body:   file,
	}
	
	if opts.Private {
		input.ACL = "private"
	} else {
		input.ACL = "public-read"
	}
	
	if opts.Metadata != nil {
		input.Metadata = opts.Metadata
	}
	
	_, err := p.client.PutObject(ctx, input)
	if err != nil {
		return nil, err
	}
	
	url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucket, path)
	if p.cdnURL != "" {
		url = p.cdnURL + "/" + path
	}
	
	return &FileInfo{
		Name:   filepath.Base(path),
		Path:   path,
		URL:    url,
		Bucket: bucket,
	}, nil
}

func (p *s3Provider) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	output, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, err
	}
	return output.Body, nil
}

func (p *s3Provider) Delete(ctx context.Context, path string) error {
	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
	})
	return err
}

func (p *s3Provider) GetURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	if expiry == 0 {
		if p.cdnURL != "" {
			return p.cdnURL + "/" + path, nil
		}
		return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", p.bucket, path), nil
	}
	
	presignClient := s3.NewPresignClient(p.client)
	
	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	
	if err != nil {
		return "", err
	}
	
	return request.URL, nil
}

func (p *s3Provider) Exists(ctx context.Context, path string) (bool, error) {
	_, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}

// =============================================================================
// LOCAL PROVIDER
// =============================================================================

type localProvider struct {
	basePath string
	baseURL  string
}

func newLocalProvider(cfg *Config) (*localProvider, error) {
	// Ensure directory exists
	if err := os.MkdirAll(cfg.LocalPath, 0755); err != nil {
		return nil, err
	}
	
	return &localProvider{
		basePath: cfg.LocalPath,
		baseURL:  cfg.LocalBaseURL,
	}, nil
}

func (p *localProvider) Upload(ctx context.Context, file io.Reader, path string, opts UploadOptions) (*FileInfo, error) {
	fullPath := filepath.Join(p.basePath, path)
	
	// Create directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	
	// Create file
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()
	
	// Copy content
	_, err = io.Copy(dst, file)
	if err != nil {
		os.Remove(fullPath)
		return nil, err
	}
	
	url := p.baseURL + "/" + path
	
	return &FileInfo{
		Name: filepath.Base(path),
		Path: path,
		URL:  url,
	}, nil
}

func (p *localProvider) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(p.basePath, path)
	return os.Open(fullPath)
}

func (p *localProvider) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(p.basePath, path)
	return os.Remove(fullPath)
}

func (p *localProvider) GetURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	return p.baseURL + "/" + path, nil
}

func (p *localProvider) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(p.basePath, path)
	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// =============================================================================
// IMAGE PROCESSING HELPERS
// =============================================================================

// ImageSize represents image dimensions
type ImageSize struct {
	Name   string
	Width  int
	Height int
}

// Common image sizes
var (
	ImageSizeThumb  = ImageSize{Name: "thumb", Width: 150, Height: 150}
	ImageSizeSmall  = ImageSize{Name: "small", Width: 300, Height: 300}
	ImageSizeMedium = ImageSize{Name: "medium", Width: 600, Height: 600}
	ImageSizeLarge  = ImageSize{Name: "large", Width: 1200, Height: 1200}
)

// AllowedImageTypes for upload validation
var AllowedImageTypes = []string{
	"image/jpeg",
	"image/png",
	"image/gif",
	"image/webp",
}

// AllowedDocumentTypes for upload validation
var AllowedDocumentTypes = []string{
	"application/pdf",
	"application/msword",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"application/vnd.ms-excel",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
}

// MaxImageSize in bytes (10MB)
const MaxImageSize = 10 * 1024 * 1024

// MaxDocumentSize in bytes (50MB)
const MaxDocumentSize = 50 * 1024 * 1024
