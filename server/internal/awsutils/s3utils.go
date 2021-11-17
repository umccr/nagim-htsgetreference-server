package awsutils

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"strings"
)

type S3ClientApi interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)

}

type S3Dto struct {
	ObjPath string
	Client  S3ClientApi
}

func (dto *S3Dto) getBucketAndKey() (string, string) {
	objPath := dto.ObjPath
	trimmedPath := strings.TrimPrefix(objPath, S3Proto)
	bucketName := strings.Split(trimmedPath, "/")[0]
	objKeyName := strings.TrimPrefix(trimmedPath, bucketName+"/")
	return bucketName, objKeyName
}

func (dto *S3Dto) NewS3Client() S3ClientApi {
	if dto.Client != nil {
		return dto.Client
	}

	defaultCfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil
	}
	return s3.NewFromConfig(defaultCfg)
}

func HeadS3Object(dto S3Dto) (int64, error) {
	client := dto.NewS3Client()
	bucketName, objKeyName := dto.getBucketAndKey()

	headResp, herr := client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objKeyName),
	})
	if herr != nil {
		return 0, herr
	}
	return headResp.ContentLength, nil
}

func GetS3Object(dto S3Dto) (io.ReadCloser, error) {
	client := dto.NewS3Client()
	bucketName, objKeyName := dto.getBucketAndKey()

	getResp, gErr := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objKeyName),
	})

	if gErr != nil {
		return nil, gErr
	}

	return getResp.Body, nil
}

func GetS3ObjectRange(dto S3Dto, start int64, end int64) (io.ReadCloser, error) {
	client := dto.NewS3Client()
	bucketName, objKeyName := dto.getBucketAndKey()

	getResp, gErr := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objKeyName),
		Range:  aws.String(fmt.Sprintf("bytes=%d-%d", start, end)),
	})

	if gErr != nil {
		return nil, gErr
	}

	return getResp.Body, nil
}

func PresignGetObjectRange(dto S3Dto, start int64, end int64) (string, error) {
	client := dto.NewS3Client()

	// the use of the mock client/S3ApiClient means that the Presign object cannot be
	// initialised from the client type without coercion
	// TODO: a more idiomatic Go way to still have the mock test client but allow this
	fullClient, ok := client.(*s3.Client)
	if !ok {
		return "", errors.New("Tried to use the presign operation when in mock test mode")
	}

	presignClient := s3.NewPresignClient(fullClient)

	bucketName, objKeyName := dto.getBucketAndKey()

	req, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objKeyName),
		Range:  aws.String(fmt.Sprintf("bytes=%d-%d", start, end)),
	})

	if err != nil {
		return "", err
	}

	return req.URL, nil
}
