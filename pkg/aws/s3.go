package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"io/ioutil"
)

type S3Client struct {
	Client *s3.S3
}

func NewS3Client(session *session.Session, region string, creds *credentials.Credentials) S3Client {
	return S3Client{
		Client: getS3ClientFn(session, region, creds),
	}
}

func getS3ClientFn(session *session.Session, region string, creds *credentials.Credentials) *s3.S3 {
	if creds == nil {
		return s3.New(session, &aws.Config{Region: aws.String(region)})
	}
	return s3.New(session, &aws.Config{Region: aws.String(region), Credentials: creds})
}

func (s S3Client) GetManifest(bucket, key string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := s.Client.GetObject(input)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
