package s3

import (
	"fmt"
	"os"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type Config struct {
	AwsAccessKeyId     string
	AwsSecretAccessKey string
	AwsRegion          string
	Bucket             string
	Endpoint           string
}

func fromEnv() Config {
	accessKeyId, _ := os.LookupEnv("AWS_ACCESS_KEY_ID")
	awsSecretAccessKey, _ := os.LookupEnv("AWS_SECRET_ACCESS_KEY")
	awsRegion, _ := os.LookupEnv("AWS_DEFAULT_REGION")
	endpoint, _ := os.LookupEnv("AWS_ENDPOINT")
	bucket, _ := os.LookupEnv("AWS_BUCKET")
	return Config{
		AwsAccessKeyId:     accessKeyId,
		AwsSecretAccessKey: awsSecretAccessKey,
		AwsRegion:          awsRegion,
		Endpoint:           endpoint,
		Bucket:             bucket,
	}
}

func newSession() *session.Session {
	config := fromEnv()
	return session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentialsFromCreds(credentials.Value{
			AccessKeyID:     config.AwsAccessKeyId,
			SecretAccessKey: config.AwsSecretAccessKey,
		}),
		Region:   aws.String(config.AwsRegion),
		Endpoint: aws.String(config.Endpoint),
	}))
}

func UploadBackupToS3(file *os.File) error {
	config := fromEnv()
	sess := newSession()
	uploader := s3manager.NewUploader(sess)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(path.Base(file.Name())),
		Body:   file,
	})
	return err
}

func ListBackups() ([]*s3.Object, error) {
	sess := newSession()
	config := fromEnv()
	svc := s3.New(sess)
	resp, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(config.Bucket)})
	if err != nil {
		return nil, fmt.Errorf("unable to list items in bucket %s: %s", config.Bucket, err.Error())
	}
	return resp.Contents, nil
}

func DeleteBackupFromS3(key string) error {
	sess := newSession()
	config := fromEnv()
	svc := s3.New(sess)
	_, err := svc.DeleteObject(&s3.DeleteObjectInput{Bucket: aws.String(config.Bucket), Key: aws.String(key)})
	return err
}
