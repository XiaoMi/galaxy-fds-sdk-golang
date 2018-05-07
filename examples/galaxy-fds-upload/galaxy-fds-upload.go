// Command galaxy-fds-upload provides a simple CLI for upload files to galaxy FDS.
// go install -v github.com/XiaoMi/galaxy-fds-sdk-golang/examples/galaxy-fds-upload
package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"os"

	galaxy_fds_sdk_golang "github.com/XiaoMi/galaxy-fds-sdk-golang"
)

var (
	endpoint = flag.String("endponit", "", "")
	key      = flag.String("key", "", "")
	secret   = flag.String("secret", "", "")
	region   = flag.String("region", "", "")
	bucket   = flag.String("bucket", "", "")
	file     = flag.String("file", "", "")
)

type uploader struct {
	endpoint string
	region   string
	key      string
	secret   string
}

func flagOrEnv(flagValue string, env string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(env)
}

func newUploader() (*uploader, error) {
	u := uploader{
		endpoint: flagOrEnv(*endpoint, "GALAXY_FDS_ENDPOINT"),
		region:   flagOrEnv(*region, "GALAXY_FDS_REGION"),
		key:      flagOrEnv(*key, "GALAXY_FDS_KEY"),
		secret:   flagOrEnv(*secret, "GALAXY_FDS_SECRET"),
	}
	if u.endpoint == "" {
		return nil, errors.New("-endpoint or GALAXY_FDS_ENDPOINT is required")
	}
	if u.key == "" {
		return nil, errors.New("-key or GALAXY_FDS_KEY is required")
	}
	if u.secret == "" {
		return nil, errors.New("-secret or GALAXY_FDS_SECRET is required")
	}
	return &u, nil
}

func main() {
	flag.Parse()
	u, err := newUploader()
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	if err := u.upload(*bucket, *file); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func (u uploader) upload(bucket, file string) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	client := galaxy_fds_sdk_golang.NEWFDSClient(u.key, u.secret, u.region, u.endpoint, true, false)
	_, err = client.Put_Object(bucket, file, content, "", nil)
	return err
}
