/*
copyright 2020 the Goployer authors

licensed under the apache license, version 2.0 (the "license");
you may not use this file except in compliance with the license.
you may obtain a copy of the license at

    http://www.apache.org/licenses/license-2.0

unless required by applicable law or agreed to in writing, software
distributed under the license is distributed on an "as is" basis,
without warranties or conditions of any kind, either express or implied.
see the license for the specific language governing permissions and
limitations under the license.
*/

package aws

import (
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Client struct {
	Client *s3.S3
}

func NewS3Client(session client.ConfigProvider, region string, creds *credentials.Credentials) S3Client {
	return S3Client{
		Client: getS3ClientFn(session, region, creds),
	}
}

func getS3ClientFn(session client.ConfigProvider, region string, creds *credentials.Credentials) *s3.S3 {
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
