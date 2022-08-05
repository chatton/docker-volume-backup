package s3backup

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"

	"docker-volume-backup/cmd/util/dateutil"
	"docker-volume-backup/cmd/util/dockerutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/s3-example-basic-bucket-operations.html

type Mode struct {
	hostPathForBackups string
	config             Config
}

func NewMode(hostPath string) *Mode {
	return &Mode{
		hostPathForBackups: hostPath,
		config:             fromEnv(),
	}
}

func (s *Mode) CrateBackup(ctx context.Context, cli *client.Client, mountPoint types.MountPoint) error {
	nameOfBackedupArchive := fmt.Sprintf("%s-%s.tar.gz", mountPoint.Name, dateutil.GetDayMonthYear())
	filePath := fmt.Sprintf("/backups/.s3tmp/%s", nameOfBackedupArchive)
	cmd := []string{"tar", "-czvf", filePath, "/data"}
	if err := dockerutil.RunCommandInMountedContainer(ctx, s.hostPathForBackups, cli, mountPoint, cmd); err != nil {
		return fmt.Errorf("failed running command in container: %s", err)
	}

	backupFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	log.Printf("backing up to s3")
	if err := UploadBackupToS3(mountPoint.Name, backupFile); err != nil {
		return fmt.Errorf("failed backing up to s3: %s", err)
	}
	if err := DeleteOtherBackupsForVolume(path.Base(backupFile.Name()), mountPoint.Name); err != nil {
		return fmt.Errorf("failed deleting older backups: %s", err)
	}

	// remove the archive after it was successfully uploaded to s3.
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to remove temprory archive: %s", err)
	}

	log.Println("successfully ensured no other backups for the same volume exist in s3")
	return nil
}

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

func UploadBackupToS3(volumeName string, file *os.File) error {
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

func DeleteOtherBackupsForVolume(backupFileKey, volumeName string) error {
	objects, err := ListBackups(volumeName)
	if err != nil {
		return err
	}
	for _, obj := range objects {
		// this is the only one we want to keep.
		if *obj.Key == backupFileKey {
			continue
		}
		if err := DeleteBackupFromS3(*obj.Key); err != nil {
			log.Printf("failed deleting backup for key %s: %s\n", *obj.Key, err)
		}
	}
	return nil
}

func ListBackups(prefix string) ([]*s3.Object, error) {
	sess := newSession()
	config := fromEnv()
	svc := s3.New(sess)
	resp, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(config.Bucket), Prefix: aws.String(prefix)})
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
