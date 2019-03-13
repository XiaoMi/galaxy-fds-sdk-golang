package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	fds "github.com/XiaoMi/galaxy-fds-sdk-golang"
)

// InitMultipartUploadResult is init multipart upload result
type InitMultipartUploadResult struct {
	BucketName string `json:"bucketName"`
	ObjectName string `json:"objectName"`
	UploadID   string `json:"uploadId"`
	Type       string `json:"type"`
}

// UploadPartResult is upload part result
type UploadPartResult struct {
	PartNumber int    `json:"partNumber"`
	Etag       string `json:"etag"`
	PartSize   int64  `json:"partSize"`
}

func main() {

	now := time.Now()
	ts := now.Add(1000000*time.Millisecond).UnixNano() / int64(time.Millisecond)

	fdsClient := fds.NEWFDSClient(os.Getenv("FDS_AK"), os.Getenv("FDS_SK"), "", os.Getenv("FDS_ENDPOINT"), true, false)

	bucketName := "examples"
	objectName := "a.txt"
	url, err := fdsClient.GeneratePresignedURI(bucketName, objectName, http.MethodPut, []string{"uploads"}, int64(ts), http.Header{})
	if err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println(url)

	httpClient := http.DefaultClient

	// init
	initRequest, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		log.Fatal(err)
		return
	}
	initResponse, err := httpClient.Do(initRequest)
	if err != nil {
		log.Fatal(err)
		return
	}
	var initMultipartUploadResult InitMultipartUploadResult
	if b, err := ioutil.ReadAll(initResponse.Body); err == nil {
		fmt.Println(string(b))
		err = json.Unmarshal(b, &initMultipartUploadResult)
		if err != nil {
			log.Fatal(err)
			return
		}
	}

	uploadID := initMultipartUploadResult.UploadID

	fmt.Println(uploadID)

	// upload part
	url, err = fdsClient.GeneratePresignedURI(bucketName,
		objectName,
		http.MethodPut,
		[]string{fmt.Sprintf("uploadId=%s", uploadID), fmt.Sprintf("partNumber=1")},
		ts,
		http.Header{})
	fmt.Println(url)
	if err != nil {
		log.Fatal(err)
		return
	}
	uploadPartRequest, err := http.NewRequest(http.MethodPut, url, strings.NewReader("Hello, World"))
	if err != nil {
		log.Fatal(err)
		return
	}
	uploadPartResponse, err := httpClient.Do(uploadPartRequest)
	if err != nil {
		log.Fatal(err)
		return
	}

	// complete
	b, err := ioutil.ReadAll(uploadPartResponse.Body)
	if err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println(string(b))
	var uploadPartResult UploadPartResult
	json.Unmarshal(b, &uploadPartResult)
	b, err = json.Marshal(struct {
		UploadPartResultList []UploadPartResult `json:"uploadPartResultList"`
	}{[]UploadPartResult{uploadPartResult}})
	if err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println(string(b))

	url, err = fdsClient.GeneratePresignedURI(
		bucketName,
		objectName,
		http.MethodPut,
		[]string{fmt.Sprintf("uploadId=%s", uploadID)},
		ts,
		http.Header{})
	if err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println(url)
	completeRequest, err := http.NewRequest(http.MethodPut, url, ioutil.NopCloser(bytes.NewReader(b)))
	if err != nil {
		log.Fatal(err)
		return
	}
	completeResponse, err := httpClient.Do(completeRequest)
	if err != nil {
		log.Fatal(err)
		return
	}
	if b, err := ioutil.ReadAll(completeResponse.Body); err == nil {
		fmt.Println(string(b))
		return
	}
}
