package s3

import (
	"context"
	"errors"
	"io"
	"testing"

	"code.byted.org/flow/eino/components/document"
	"code.byted.org/flow/eino/schema"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"
)

func TestNewS3Loader(t *testing.T) {
	mockey.PatchConvey("TestNewS3Loader", t, func() {
		var (
			l   document.Loader
			err error
			ctx = context.Background()
		)

		_, err = NewS3Loader(ctx, nil)
		assert.Error(t, err)

		_, err = NewS3Loader(ctx, &LoaderConfig{
			AWSSecretKey: aws.String("xxx"),
		})
		assert.Error(t, err)

		_, err = NewS3Loader(ctx, &LoaderConfig{
			AWSAccessKey: aws.String("xxx"),
		})
		assert.Error(t, err)

		l, err = NewS3Loader(ctx, &LoaderConfig{
			Region:       aws.String("region"),
			AWSAccessKey: aws.String("ak"),
			AWSSecretKey: aws.String("sk"),
		})

		assert.NoError(t, err)
		assert.NotNil(t, l)
	})
}

func TestLoader_Load(t *testing.T) {
	mockey.PatchConvey("TestLoader_Load", t, func() {
		var (
			result []*schema.Document
			err    error
			ctx    = context.Background()
		)

		s3Loader, err := NewS3Loader(ctx, &LoaderConfig{
			Region:           aws.String("region"),
			AWSAccessKey:     aws.String("ak"),
			AWSSecretKey:     aws.String("sk"),
			UseObjectKeyAsID: true,
		})
		assert.NoError(t, err)

		_, err = s3Loader.Load(ctx, document.Source{
			URI: "bucket/key",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not s3 uri")

		_, err = s3Loader.Load(ctx, document.Source{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "uri is empty")

		_, err = s3Loader.Load(ctx, document.Source{
			URI: "s3://bucket",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "incomplete")

		_, err = s3Loader.Load(ctx, document.Source{
			URI: "s3://bucket/prefix/",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not support batch load with prefix")

		mockey.PatchConvey("get object returns no such key", func() {
			mockey.Mock((*s3.Client).GetObject).Return(nil, &types.NoSuchKey{}).Build()

			_, err = s3Loader.Load(ctx, document.Source{
				URI: "s3://bucket/key",
			})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})

		mockey.PatchConvey("get object returns other error", func() {
			mockey.Mock((*s3.Client).GetObject).Return(nil, errors.New("")).Build()

			_, err = s3Loader.Load(ctx, document.Source{
				URI: "s3://bucket/key",
			})
			assert.Error(t, err)
		})

		r, w := io.Pipe()
		go func() {
			_, _ = w.Write([]byte("hello"))
			_, _ = w.Write([]byte(" world!"))
			_ = w.Close()
		}()

		mockey.Mock((*s3.Client).GetObject).Return(&s3.GetObjectOutput{
			Body: r,
		}, nil).Build()

		result, err = s3Loader.Load(ctx, document.Source{
			URI: "s3://bucket/key.txt",
		})
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "hello world!", result[0].Content)
		assert.Equal(t, "key.txt", result[0].ID)
	})
}
