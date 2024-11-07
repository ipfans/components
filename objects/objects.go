package objects

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

type Config struct {
	BaseEndpoint    string `json:"BaseEndpoint"`
	Region          string `json:"Region"`
	BucketName      string `json:"BucketName"`
	AccessKeyId     string `json:"AccessKeyId"`
	AccessKeySecret string `json:"AccessKeySecret"`
}

type Object struct {
	Client     *s3.Client
	Uploader   *manager.Uploader
	Downloader *manager.Downloader
	cfg        Config
}

// New creates a new Object instance.
func New(cfg Config) (*Object, error) {
	conf, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyId, cfg.AccessKeySecret, "")),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(conf, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.BaseEndpoint)
	})
	uploader := manager.NewUploader(client)
	downloader := manager.NewDownloader(client)

	return &Object{
		Client:     client,
		Uploader:   uploader,
		Downloader: downloader,
		cfg:        cfg,
	}, nil
}

// Upload uploads an object to the bucket with the given key.
func (o *Object) Upload(ctx context.Context, key string, body io.Reader) error {
	_, err := o.Uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(o.cfg.BucketName),
		Key:    aws.String(key),
		Body:   body,
	})
	if err != nil {
		return err
	}
	err = s3.NewObjectExistsWaiter(o.Client).Wait(
		ctx,
		&s3.HeadObjectInput{
			Bucket: aws.String(o.cfg.BucketName),
			Key:    aws.String(key),
		},
		time.Minute,
	)
	return err
}

// UploadFile uploads a file to the bucket with the given key.
func (o *Object) UploadFile(ctx context.Context, key string, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return o.Upload(ctx, key, f)
}

// Download downloads an object in the bucket with the given key to the given writer.
func (o *Object) Download(ctx context.Context, key string, w io.WriterAt) error {
	_, err := o.Downloader.Download(ctx, w, &s3.GetObjectInput{
		Bucket: aws.String(o.cfg.BucketName),
		Key:    aws.String(key),
	})
	return err
}

// DownloadFile downloads an object in the bucket with the given key to the given path.
func (o *Object) DownloadFile(ctx context.Context, key string, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return o.Download(ctx, key, f)
}

// Delete deletes an object in the bucket with the given key.
func (o *Object) Delete(ctx context.Context, key string) error {
	_, err := o.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(o.cfg.BucketName),
		Key:    aws.String(key),
	})
	_ = s3.NewObjectNotExistsWaiter(o.Client).Wait(
		ctx,
		&s3.HeadObjectInput{Bucket: aws.String(o.cfg.BucketName), Key: aws.String(key)},
		time.Minute,
	)
	return err
}

// DeleteObjects deletes objects in the bucket with the given keys.
func (o *Object) DeleteObjects(ctx context.Context, keys []string) error {
	var objectIds []types.ObjectIdentifier
	for _, key := range keys {
		objectIds = append(objectIds, types.ObjectIdentifier{Key: aws.String(key)})
	}
	output, err := o.Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(o.cfg.BucketName),
		Delete: &types.Delete{Objects: objectIds, Quiet: aws.Bool(true)},
	})
	if err != nil {
		return err
	}
	for _, delObjs := range output.Deleted {
		_ = s3.NewObjectNotExistsWaiter(o.Client).Wait(
			ctx,
			&s3.HeadObjectInput{Bucket: aws.String(o.cfg.BucketName), Key: delObjs.Key},
			time.Minute,
		)
	}
	return err
}

// ListObjects lists objects in the bucket with the given prefix and limit. Prefix & limit will be ignored if it's default value.
func (o *Object) ListObjects(ctx context.Context, prefix string, limit int) ([]string, error) {
	var keys []string

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(o.cfg.BucketName),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}
	if limit > 0 {
		input.MaxKeys = aws.Int32(int32(limit))
	}

	var err error
	paginator := s3.NewListObjectsV2Paginator(o.Client, input)
	for paginator.HasMorePages() {
		var page *s3.ListObjectsV2Output
		page, err = paginator.NextPage(ctx)
		if err != nil {
			break
		}
		for _, obj := range page.Contents {
			keys = append(keys, *obj.Key)
		}
		if limit > 0 && len(keys) >= limit {
			keys = keys[:limit]
			break
		}
	}
	return keys, err
}

// Exists checks if an object exists in the bucket with the given key.
func (o *Object) Exists(ctx context.Context, key string) (bool, error) {
	_, err := o.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(o.cfg.BucketName),
		Key:    aws.String(key),
	})
	exists := true
	if err != nil {
		var apiError smithy.APIError
		if errors.As(err, &apiError) {
			switch apiError.(type) {
			case *types.NotFound:
				exists = false
				err = nil
				return exists, err
			default:
				return false, err
			}
		}
	}
	return exists, err
}
