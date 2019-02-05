package testutils

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/percona/percona-backup-mongodb/internal/storage"
)

var (
	storages *storage.Storages
)

func TestingStorages() *storage.Storages {
	if storages != nil {
		return storages
	}
	tmpDir := os.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "dump_test"), os.ModePerm)

	st := &storage.Storages{
		Storages: map[string]storage.Storage{
			"s3-us-west": {
				Type: "s3",
				S3: storage.S3{
					Region: "us-west-2",
					Bucket: RandomBucket(),
					Credentials: storage.Credentials{
						AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
						SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
					},
				},
				Filesystem: storage.Filesystem{},
			},
			"local-filesystem": {
				Type: "filesystem",
				Filesystem: storage.Filesystem{
					Path: filepath.Join(tmpDir, "dump_test"),
				},
			},
		},
	}
	storages = st
	return st
}

func RandomBucket() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("pbm-test-bucket-%05d", rand.Int63n(99999))
}
