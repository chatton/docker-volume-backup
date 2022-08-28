package periodic

import (
	"os"

	"gopkg.in/yaml.v2"
)

const (
	defaultConfigPath = "/etc/docker-volume-backup/config.yml"
)

type Config struct {
	PeriodicBackups []CronConfiguration `yaml:"periodic_backups"`
}

type CronConfiguration struct {
	Name        string   `yaml:"name"`
	Schedule    string   `yaml:"schedule"`
	ScheduleKey string   `yaml:"schedule_key"`
	Backups     []Backup `yaml:"backups"`
}

type Backup struct {
	Name              string             `yaml:"name"`
	Type              string             `yaml:"type"`
	FilesystemOptions *FilesystemOptions `yaml:"filesystem_options,omitempty"`
	S3Options         *S3Options         `yaml:"s3_options,omitempty"`
}

type FilesystemOptions struct {
	Hostpath string `yaml:"host_path"`
}
type S3Options struct {
	Hostpath           string `yaml:"host_path"`
	AwsAccessKeyID     string `yaml:"aws_access_key_id"`
	AwsSecretAccessKey string `yaml:"aws_secret_access_key"`
	AwsDefaultRegion   string `yaml:"aws_default_region"`
	AwsBucket          string `yaml:"aws_bucket"`
	AwsEndpoint        string `yaml:"aws_endpoint"`
}

func LoadConfig() (Config, error) {
	configBytes, err := os.ReadFile(defaultConfigPath)
	if err != nil {
		return Config{}, err
	}

	config := Config{}
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		return Config{}, err
	}
	return config, nil
}
