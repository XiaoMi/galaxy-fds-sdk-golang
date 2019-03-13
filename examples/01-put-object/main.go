package main

import (
	"fmt"
	"log"
	"os"

	fds "github.com/XiaoMi/galaxy-fds-sdk-golang"
)

func main() {
	momentContent := "hello"
	momentName := "b.mp4"
	bucketName := "examples"
	fdsClient := fds.NEWFDSClient(os.Getenv("FDS_AK"), os.Getenv("FDS_SK"), "", os.Getenv("FDS_ENDPOINT"), true, false)
	_, err := fdsClient.Put_Object(bucketName, momentName, []byte(momentContent), "video/mp5", nil)
	if err != nil {
		log.Fatalf("err = %v", err)
		return
	}
	_, err = fdsClient.Set_Public(bucketName, momentName, true)
	if err != nil {
		log.Fatalf("err = %v", err)
		return
	}

	url := fdsClient.Generate_Download_Object_Uri(bucketName, momentName)
	fmt.Println(url)
}
