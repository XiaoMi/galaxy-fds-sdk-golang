package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	fds "github.com/XiaoMi/galaxy-fds-sdk-golang"
	"github.com/XiaoMi/galaxy-fds-sdk-golang/Model"
)

func main() {
	momentName := "c.mp4"
	bucketName := "examples"
	fdsClient := fds.NEWFDSClient(os.Getenv("FDS_AK"), os.Getenv("FDS_SK"), "", os.Getenv("FDS_ENDPOINT"), true, false)

	initResult, err := fdsClient.Init_MultiPart_Upload(bucketName, momentName, "video/mp4")
	if err != nil {
		log.Fatalf("failed to init, %v\n", err)
	}

	uploadResult, err := fdsClient.Upload_Part(initResult, 1, []byte("hello world"))
	if err != nil {
		log.Fatalf("failed to upload, %v\n", err)
	}

	resultList := Model.UploadPartList{
		UploadPartResultList: []Model.UploadPartResult{*uploadResult},
	}

	metadata := http.Header{}
	metadata.Add("Content-Type", "video/mp4")
	completeResult, err := fdsClient.CompleteMultipartUpload(initResult, metadata, &resultList)
	if err != nil {
		log.Fatalf("failed to complete, %v\n", err)
	}
	fmt.Printf("%v\n", completeResult)
}
