// Package s3 can load document from AWS S3.
package s3

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components/document"
	"code.byted.org/flow/eino/components/document/parser"
	"code.byted.org/flow/eino/schema"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// LoaderConfig is the configuration for s3 loader.
type LoaderConfig struct {
	Region       *string // the region of the AWS bucket
	AWSAccessKey *string
	AWSSecretKey *string

	UseObjectKeyAsID bool // whether to use object key as document ID

	Parser parser.Parser // the parser to parse the s3 object stream into documents, default to parser.TextParser, which directly converts []byte to string
}

type loader struct {
	client *s3.Client

	parser parser.Parser

	useObjectKeyAsID bool
}

// NewS3Loader creates a new s3 loader.
func NewS3Loader(ctx context.Context, conf *LoaderConfig) (document.Loader, error) {
	if conf == nil {
		return nil, errors.New("new s3 loader, config is nil")
	}

	var s3Opts []func(*config.LoadOptions) error
	if conf.Region != nil {
		s3Opts = append(s3Opts, config.WithRegion(*conf.Region))
	}

	var (
		hasAccessKey = conf.AWSAccessKey != nil
		hasSecretKey = conf.AWSSecretKey != nil
	)

	if (hasAccessKey && !hasSecretKey) || (!hasAccessKey && hasSecretKey) {
		return nil, errors.New("new s3 loader, aws access key and secret key must be set together")
	}

	if hasAccessKey {
		s3Opts = append(s3Opts, config.WithCredentialsProvider(&credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     *conf.AWSAccessKey,
				SecretAccessKey: *conf.AWSSecretKey,
			},
		}))
	}

	sdkConfig, err := config.LoadDefaultConfig(ctx, s3Opts...)
	if err != nil {
		return nil, fmt.Errorf("new s3 loader, load config err: %w", err)
	}

	client := s3.NewFromConfig(sdkConfig)

	p := conf.Parser
	if p == nil {
		p = &parser.TextParser{}
	}

	return &loader{
		client:           client,
		parser:           p,
		useObjectKeyAsID: conf.UseObjectKeyAsID,
	}, nil
}

// Load loads the s3 object from the given URI.
func (l *loader) Load(ctx context.Context, src document.Source, opts ...document.LoaderOption) (docs []*schema.Document, err error) {
	ctx = callbacks.OnStart(ctx, &document.LoaderCallbackInput{
		Source: src,
	})

	defer func() {
		if err != nil {
			_ = callbacks.OnError(ctx, err)
		}
	}()

	// parse document.Source's URI to be s3 bucket + object key, or bucket + prefix.
	bucket, key, isPrefix, e := uriToBucketAndKey(src.URI)
	if e != nil {
		err = e
		return nil, err
	}

	// prefix is not supported now
	if isPrefix {
		err = errors.New("s3 loader does not support batch load with prefix for now")
		return nil, err
	}

	// get object from s3
	resp, err := l.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var noKey *types.NoSuchKey
		if errors.As(err, &noKey) {
			err = fmt.Errorf("s3 loader bucket= %s, key= %s not found, err: %w", bucket, key, err)
			return nil, err
		}

		err = fmt.Errorf("s3 loader get object err: %w", err)
		return nil, err
	}
	defer resp.Body.Close()

	docs, err = l.parser.Parse(ctx, resp.Body, parser.WithURI(src.URI))
	if err != nil {
		err = fmt.Errorf("s3 loader parse err: %w", err)
		return nil, err
	}

	if l.useObjectKeyAsID {
		for _, doc := range docs {
			doc.ID = key
		}
	}

	_ = callbacks.OnEnd(ctx, &document.LoaderCallbackOutput{
		Source: src,
		Docs:   docs,
	})

	return docs, nil
}

func uriToBucketAndKey(uri string) (bucket string, key string, isPrefix bool, err error) {
	const (
		uriPrefix = `s3://`
		separator = `/`
	)

	if len(uri) == 0 {
		return "", "", false, errors.New("s3 loader source uri is empty")
	}

	if !strings.HasPrefix(uri, uriPrefix) {
		return "", "", false, fmt.Errorf("uri is not s3 uri, uri: %s", uri)
	}

	bucketAndKey := strings.TrimPrefix(uri, uriPrefix)
	bucketEnd := strings.Index(bucketAndKey, separator)
	if bucketEnd == -1 {
		return "", "", false, fmt.Errorf("s3 uri incomplete: %s", uri)
	}

	bucket = bucketAndKey[:bucketEnd]
	key = bucketAndKey[bucketEnd+1:]

	if strings.HasSuffix(key, separator) {
		return bucket, key, true, nil
	}

	return bucket, key, false, nil
}

func (l *loader) GetType() string {
	return "S3Loader"
}

func (l *loader) IsCallbacksEnabled() bool {
	return true
}
