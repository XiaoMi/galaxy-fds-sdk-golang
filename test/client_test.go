package test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	fds "github.com/XiaoMi/galaxy-fds-sdk-golang"
	"github.com/XiaoMi/galaxy-fds-sdk-golang/Model"
)

// Const
var (
	AppKey     = os.Getenv("FDS_AK")
	SecretKey  = os.Getenv("FDS_SK")
	BucketName = "go-lang-test"
	Endpoint   = os.Getenv("FDS_ENDPOINT") // fds domain
)

func getObjectName4test() string {
	pc, _, _, _ := runtime.Caller(1)
	return "golang-test-" + runtime.FuncForPC(pc).Name() + "-" + time.Now().Format(time.RFC3339)
}

var client *fds.FDSClient

func Test_Put_Get_Object(t *testing.T) {
	objectName := getObjectName4test()

	content := []byte("blah" + time.Now().Format(time.ANSIC))
	_, err := client.Put_Object(BucketName, objectName, content, "", nil)
	if err != nil {
		t.Error("Fail to put object: "+objectName, err)
	}

	fdsobject, err := client.Get_Object(BucketName, objectName, 0, -1)
	if err != nil {
		t.Error("Fail to get object: "+objectName, err)
	}

	if !bytes.Equal(content, fdsobject.ObjectContent) {
		t.Error("content changed")
	}
}

func Test_MultiPartUpload(t *testing.T) {
	objectName := getObjectName4test()

	initMultiPartResult, err := client.Init_MultiPart_Upload(BucketName, objectName, "")
	if err != nil {
		t.Error("Fail to init multipart upload", err)
	}

	var content [3][]byte
	content[0] = make([]byte, 5242880)
	content[1] = make([]byte, 5242880)
	content[2] = make([]byte, 77777)

	var uploadPartList Model.UploadPartList
	for i := range content {
		for j := range content[i] {
			content[i][j] = byte((i * j) ^ (i + j) - 7)
		}
		uploadPartResult, err := client.Upload_Part(initMultiPartResult, i+1, content[i])
		if err != nil {
			t.Error(fmt.Sprintf("Fail to upload part: %d", i))
		}
		uploadPartList.AddUploadPartResult(uploadPartResult)
	}

	if len(uploadPartList.UploadPartResultList) != len(content) {
		t.Error("unexpected")
	}

	_, err = client.Complete_Multipart_Upload(initMultiPartResult, &uploadPartList)
	if err != nil {
		t.Error("Fail to complete multipart upload", err)
	}

	fdsobject, err := client.Get_Object(BucketName, objectName, 0, -1)
	if err != nil {
		t.Error("Fail to get object "+objectName, err)
	}
	allContent := content[0]
	allContent = append(allContent, content[1]...)
	allContent = append(allContent, content[2]...)
	if !bytes.Equal(allContent, fdsobject.ObjectContent) {
		t.Error("content changed")
	}

	// test abort interface
	initMultiPartResult, err = client.Init_MultiPart_Upload(BucketName, objectName, "")
	if err != nil {
		t.Error("Fail to init multipart upload")
	}
	err = client.Abort_MultipartUpload(initMultiPartResult)
	if err != nil {
		t.Error("Fail to abort multipart upload")
	}

	u, err := client.Upload_Part(initMultiPartResult, 0, content[0])
	if u != nil {
		t.Error("Upload part succ after abort")
	}

	/* TODO fds do not claim failure
	if err == nil {
		t.Error("Abort_Multipart_Upload fail to clean up")
	}*/
}

func Test_ListObjects(t *testing.T) {
	objectName := []string{
		"aaa/bbb/ccc/file1",
		"aaa/bbb/ccc/file2",
		"aaa/ddd/file3",
		"aaa/ddd/file4",
		"aaa/eee"}
	objectContent := []byte("blah")

	for _, name := range objectName {
		client.Put_Object(BucketName, name, objectContent, "", nil)
	}

	listObjectResult, err := client.List_Object(BucketName, "aab/", "/", 2)
	if err != nil {
		t.Error("Fail to list objects")
	}

	if len(listObjectResult.ObjectSummaries) != 0 || len(listObjectResult.CommonPrefixes) != 0 {
		t.Error("list result should be empty")
	}

	listObjectResult, err = client.List_Object(BucketName, "aaa/", "/", 4)
	// expect:
	// commonPrefixes: ["bbb", "ddd"]
	// objectSummaries: []
	if err != nil {
		t.Error("Fail to list objects")
	}

	if len(listObjectResult.CommonPrefixes) != 2 {
		t.Error("List result should has 2 common prefixes")
	}

	if len(listObjectResult.ObjectSummaries) != 0 {
		t.Error("There should be no object in list result")
	}

	if strings.Compare(listObjectResult.CommonPrefixes[0], "aaa/bbb/") != 0 ||
		strings.Compare(listObjectResult.CommonPrefixes[1], "aaa/ddd/") != 0 {
		t.Error("List result not correct")
	}

	listObjectResult, err = client.List_Next_Batch_Of_Objects(listObjectResult)
	// expect:
	// commonPrefixes: []
	// objectSummaries: ["aaa/eee"]
	if err != nil {
		t.Error("Fail to list next batch of objects")
	}
	if len(listObjectResult.ObjectSummaries) != 1 {
		t.Error(fmt.Sprintf("There should be one ObjectSummaries, got %d", len(listObjectResult.CommonPrefixes)))
	}
	if len(listObjectResult.CommonPrefixes) != 0 {
		t.Error(fmt.Sprintf("There should be no CommonPrefixes, got %d", len(listObjectResult.ObjectSummaries)))
	}

}

func Test_DeleteObject(t *testing.T) {
	objectName := getObjectName4test()
	objectContent := "blah"

	_, err := client.Put_Object(BucketName, objectName, []byte(objectContent), "", nil)
	if err != nil {
		t.Error("Fail to put object: " + objectName)
	}

	exists, err := client.Is_Object_Exists(BucketName, objectName)
	if err != nil {
		t.Error("Fail to list object", err)
	}

	if !exists {
		t.Error("Fail to find object" + objectName)
	}

	_, err = client.Delete_Object(BucketName, objectName)
	if err != nil {
		t.Error("Fail to delete object: "+objectName, err)
	}

	exists, err = client.Is_Object_Exists(BucketName, objectName)
	if err != nil {
		t.Error("Fail to list object"+objectName, err)
	}
	if exists {
		t.Error("Deleted object still exists")
	}
}

func Test_Metadata(t *testing.T) {
	objectName := getObjectName4test()
	objectContent := "blah"
	contentType := "xxx/yyy"
	xiaomiMetaData := "x-xiaomi-meta-kakaka"

	headers := map[string]string{
		xiaomiMetaData: "I used to roll the dice",
		"wawawa":       "see the fear in my enemies' eyes",
	}

	_, err := client.Put_Object(BucketName, objectName, []byte(objectContent),
		contentType,
		&headers)
	if err != nil {
		t.Error("Fail to put object: "+objectName, err)
	}

	metadataGot, err := client.Get_Object_Meta(BucketName, objectName)
	if err != nil {
		t.Error("Fail to get object meta for object: "+objectName, err)
	}

	contentTypeGot, err := metadataGot.GetContentType()
	if err != nil {
		t.Error("No content type", err)
	}

	if strings.Compare(contentTypeGot, contentType) != 0 {
		t.Error("wrong content type, expect: "+contentType+" got: "+contentTypeGot, err)
	}

	h, err := metadataGot.GetKey(xiaomiMetaData)
	if err != nil {
		t.Error(xiaomiMetaData+" no exists", err)
	}

	if strings.Compare(h, headers[xiaomiMetaData]) != 0 {
		t.Error(xiaomiMetaData + " content changed, expect: " + headers[xiaomiMetaData] + " got: " + h)
	}

	_, err = metadataGot.GetKey("wawawa")
	if err == nil {
		t.Error("header wawawa not expected to exist")
	}
}

func Test_Presigned_Url(t *testing.T) {
	objectName := getObjectName4test()
	objectContent := "blah"
	contentType := "text/plain"
	url, err := client.GeneratePresignedURI(BucketName, objectName, "PUT", []string{},
		(time.Now().Add(time.Minute*5)).UnixNano()/int64(time.Millisecond),
		map[string][]string{
			"content-type": []string{contentType},
		})

	if err != nil {
		t.Error("Fail to generate presigned url")
	}

	req, err := http.NewRequest("PUT", url, bytes.NewReader([]byte(objectContent)))
	if err != nil {
		t.Error("Fail to allocate new request", err)
	}

	req.Header.Add("content-type", contentType)
	req.Header.Add("date", time.Now().Format(time.RFC1123))
	req.Header.Add("content-md5", "")

	c := http.Client{}
	resp, err := c.Do(req)

	resp.Body.Close()
	if err != nil {
		t.Error("Fail to put through presigned url", err)
	}

	url, err = client.GeneratePresignedURI(BucketName, objectName, "GET", []string{},
		time.Now().Add(time.Minute*5).UnixNano()/int64(time.Millisecond), map[string][]string{})

	if err != nil {
		t.Error("Fail to get presigned url", err)
	}

	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		t.Error("Fail to allocate new request", err)
	}

	res, err := c.Do(req)
	if err != nil {
		t.Error("Fail to execute request", err)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Error("Fail to close response")
	}
	if strings.Compare(string(body), objectContent) != 0 {
		t.Errorf("object content changed, expect: %s, got: %s", objectContent, string(body))
	}

	url, err = client.Generate_Presigned_URI(BucketName, objectName,
		"HEAD",
		time.Now().Add(time.Minute*5).UnixNano()/int64(time.Millisecond),
		map[string][]string{})

	req, err = http.NewRequest("HEAD", url, nil)
	if err != nil {
		t.Error("Fail to allocate new request", err)
	}

	res, err = c.Do(req)
	if err != nil {
		t.Error("Fail to execute request", err)
	}

	fmt.Printf("%v\n", res.Header)

	contentLength, err := strconv.Atoi(res.Header.Get("x-xiaomi-meta-content-length"))
	if contentLength != len(objectContent) {
		t.Errorf("content length check fail, expect: %d, got: %d", len(objectContent), contentLength)
	}
}

func Test_List_Multipart_uploads(t *testing.T) {
	objectName := getObjectName4test()
	objectContent := "blah"
	contentType := "text/plain"
	partNumber := 42

	initResult, err := client.Init_MultiPart_Upload(BucketName, objectName, contentType)
	if err != nil {
		t.Error("Fail to init multipart", err)
	}
	_, err = client.Upload_Part(initResult, partNumber, []byte(objectContent))
	if err != nil {
		t.Error("Fail to uplaod part", err)
	}

	listResult, err := client.List_Multipart_Uploads(BucketName, objectName, "", 10)
	if err != nil {
		t.Error("Fail to upload multipart", err)
	}

	if len(listResult.Uploads) != 1 {
		t.Error("multi part number, expcet: 1, got: " + strconv.Itoa(len(listResult.Uploads)))
	}

	if strings.Compare(initResult.UploadId, listResult.Uploads[0].UploadId) != 0 {
		t.Error("multi part upload id mismatch, expcect: " + initResult.UploadId + " got: " + listResult.Uploads[0].UploadId)
	}

	listParts, err := client.List_Parts(BucketName, objectName, initResult.UploadId)
	if err != nil {
		t.Error("Fail to list parts", err)
	}

	if len(listParts.UploadPartResultList) != 1 {
		t.Error("Expect 1 upload part, got: " + strconv.Itoa(len(listParts.UploadPartResultList)))
	}
	if listParts.UploadPartResultList[0].PartNumber != partNumber {
		t.Error("Expcet part Number: " + strconv.Itoa(partNumber) + " got: " +
			strconv.Itoa(listParts.UploadPartResultList[0].PartNumber))
	}
	if int(listParts.UploadPartResultList[0].PartSize) != len(objectContent) {
		t.Error("Expect part size: " + strconv.Itoa(len(objectContent)) + " got: " + fmt.Sprintf("%d", listParts.UploadPartResultList[0].PartSize))
	}
}

func clearOneBucket(client *fds.FDSClient) {
	client.Delete_Objects_With_Prefix(BucketName, "")
}

func setUpTest() {
	exists, err := client.Is_Bucket_Exists(BucketName)
	if err != nil {
		if exists {
			clearOneBucket(client)
		} else {
			client.Create_Bucket(BucketName)
		}
	}
}

func tearDown() {
	clearOneBucket(client)
}

func TestMain(m *testing.M) {
	client = fds.NEWFDSClient(AppKey, SecretKey, "", Endpoint, false, false)
	setUpTest()
	r := m.Run()
	tearDown()
	os.Exit(r)
}
