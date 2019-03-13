package galaxy_fds_sdk_golang

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/XiaoMi/galaxy-fds-sdk-golang/Model"
)

const (
	URI_CDN                            = "cdn"
	URI_FDS_SUFFIX                     = ".fds.api.xiaomi.com/"
	URI_FDS_CDN_SUFFIX                 = ".fds.api.mi-img.com/"
	URI_HTTP_PREFIX                    = "http://"
	URI_HTTPS_PREFIX                   = "https://"
	USER_DEFINED_METADATA_PREFIX       = "x-xiaomi-meta-"
	DELIMITER                          = "/"
	DEFAULT_LIST_MAX_KEYS              = 1000
	GALAXY_ACCESS_KEY_ID               = "GalaxyAccessKeyId"
	EXPIRES                            = "Expires"
	SIGNATURE                          = "Signature"
	SLICE_SIZE                   int64 = 52428800 // 50*1024*1024 下载时分片上限为50MB，低于50MB不可分片
)

// permission
const (
	PERMISSION_READ         = "READ"
	PERMISSION_WRITE        = "WRITE"
	PERMISSION_FULL_CONTROL = "FULL_CONTROL"
	PERMISSION_USER         = "USER"
	PERMISSION_GROUP        = "GROUP"
)

const (
	REGION_CNBJ0    = "cnbj0"
	REGION_CNBJ1    = "cnbj1"
	REGION_CNBJ2    = "cnbj2"
	REGION_AWSBJ0   = "awsbj0"
	REGION_AWSUSOR0 = "awsusor0"
	REGION_AWSSGP0  = "awssgp0"
	REGION_AWSDE0   = "awsde0"
)

var ALL_USERS = map[string]string{"id": "ALL_USERS"}
var AUTHENTICATED_USERS = map[string]string{"id": "AUTHENTICATED_USERS"}

var PRE_DEFINED_METADATA = []string{"cache-control",
	"content-encoding",
	"content-length",
	"content-md5",
	"content-type",
}

type FDSClient struct {
	AppKey      string
	AppSecret   string
	RegionName  string
	EndPoint    string
	EnableHttps bool
	EnableCDN   bool
}

type FDSAuth struct {
	UrlBase      string
	Method       string
	Data         []byte
	Content_Md5  string
	Content_Type string
	Headers      *map[string]string
	Params       *map[string]string
}

func NEWFDSClient(appkey, appSecret, regionName string, endPoint string, enableHttps, enableCDN bool) *FDSClient {
	if len(regionName) == 0 && len(endPoint) == 0 {
		// default to cnbj0
		regionName = REGION_CNBJ0
	}

	return &FDSClient{
		AppKey:      appkey,
		AppSecret:   appSecret,
		RegionName:  regionName,
		EndPoint:    endPoint,
		EnableHttps: enableHttps,
		EnableCDN:   enableCDN,
	}
}

func (c *FDSClient) getBaseUriPrefix() string {
	if c.EnableCDN {
		return URI_CDN + "." + c.RegionName
	}
	return c.RegionName
}

func (c *FDSClient) getUploadUriPrefix() string {
	if c.EnableCDN {
		return URI_CDN + "." + c.RegionName
	}
	return c.RegionName
}

func (c *FDSClient) getUploadUriSuffix() string {
	return URI_FDS_SUFFIX
}

func (c *FDSClient) getBaseUriSuffix() string {
	if c.EnableCDN {
		return URI_FDS_CDN_SUFFIX
	}
	return URI_FDS_SUFFIX
}

func (c *FDSClient) GetBaseUri() string {
	u := bytes.Buffer{}
	if c.EnableHttps {
		u.WriteString(URI_HTTPS_PREFIX)
	} else {
		u.WriteString(URI_HTTP_PREFIX)
	}

	if len(c.EndPoint) > 0 {
		u.WriteString(c.EndPoint)
		u.WriteString("/")
	} else {
		u.WriteString(c.getBaseUriPrefix())
		u.WriteString(c.getBaseUriSuffix())
	}
	return u.String()
}

func (c *FDSClient) GetUploadURL() string {
	u := bytes.Buffer{}
	if c.EnableHttps {
		u.WriteString(URI_HTTPS_PREFIX)
	} else {
		u.WriteString(URI_HTTP_PREFIX)
	}

	if len(c.EndPoint) > 0 {
		u.WriteString(c.EndPoint)
		u.WriteString("/")
	} else {
		u.WriteString(c.getUploadUriPrefix())
		u.WriteString(c.getUploadUriSuffix())
	}
	return u.String()
}

func (c *FDSClient) Auth(auth FDSAuth) (*http.Response, error) {
	client := &http.Client{}

	urlParsed, err := url.Parse(auth.UrlBase)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	params := url.Values{}
	for k, v := range urlParsed.Query() {
		if len(v) > 0 {
			params.Add(k, v[0])
		} else {
			params.Add(k, "")
		}
	}
	if auth.Params != nil {
		for k, v := range *auth.Params {
			params.Add(k, v)
		}
	}
	urlParsed.RawQuery = params.Encode()
	urlStr := urlParsed.String()

	req, _ := http.NewRequest(auth.Method, urlStr, ioutil.NopCloser(bytes.NewReader(auth.Data)))
	if auth.Headers != nil {
		for k, v := range *auth.Headers {
			req.Header.Add(k, v)
		}
	}
	req.Header.Add("date", time.Now().Format(time.RFC1123))
	req.Header.Add("content-md5", auth.Content_Md5)
	req.Header.Add("content-type", auth.Content_Type)

	signature, err := Signature(c.AppSecret, req.Method, urlStr, req.Header)
	if err != nil {
		return nil, err
	}

	req.Header.Add("authorization", fmt.Sprintf("Galaxy-V2 %s:%s", c.AppKey, signature))
	res, err := client.Do(req)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	return res, nil
}

//name:
//     Get_Bucket
//param:
//     bucketname: 要获取信息的bucket名字
//return:
//     *Model.BucketInfo: 由bucketinfo结构体，包含bucket相关信息
//     error: 正常返回nil，异常返回error Code
//example:
//     Not available now
//Exception: 这个接口跟java中定义不同，请谨慎使用，java中不返回任何值
func (c *FDSClient) Get_Bucket(bucketname string) (*Model.BucketInfo, error) {
	url := c.GetBaseUri() + bucketname
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "GET",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		bucketInfo, err := Model.NewBucketInfo(body)
		if err != nil {
			return nil, Model.NewFDSError(err.Error(), -1)
		}
		return bucketInfo, nil
	} else {
		return nil, Model.NewFDSError(string(body), res.StatusCode)
	}
}

func (c *FDSClient) Is_Bucket_Exists(bucketname string) (bool, error) {
	url := c.GetBaseUri() + bucketname
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "HEAD",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return true, nil
	} else {
		return false, Model.NewFDSError(string(body), res.StatusCode)
	}
}

//name:
//     List_Bucket
//param:
//     默认不使用任何参数，使用用户配置的AK SK，列出由该AK SK所属用户组创建的bucket
//return:
//     []string: 由bucket name组成的slice
//     error: 正常返回nil，异常返回error Code
//example:
//     Not available now
func (c *FDSClient) List_Bucket() ([]string, error) {
	bucketlist := []string{}
	url := c.GetBaseUri()
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "GET",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return bucketlist, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return bucketlist, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		var sj map[string]interface{}
		// 修复因为返回值为空导致json解析失败问题
		if string(body) == "" {
			return bucketlist, nil
		}
		err := json.Unmarshal(body, &sj)
		if err != nil {
			return bucketlist, Model.NewFDSError(err.Error(), -1)
		}
		buckets, ok := sj["buckets"]
		if !ok {
			return bucketlist, Model.NewFDSError(err.Error(), -1)
		}
		bucketsList, ok := buckets.([]interface{})
		if !ok {
			return bucketlist, Model.NewFDSError(err.Error(), -1)
		}
		for _, bucket := range bucketsList {
			// fmt.Printf("%#v\n", bucket.(map[string]interface{})["name"])
			bucket = bucket.(map[string]interface{})["name"]
			bucketlist = append(bucketlist, bucket.(string))
		}
		return bucketlist, nil
	} else {
		return bucketlist, Model.NewFDSError(string(body), res.StatusCode)
	}
}

//name:
//     List_Authorized_Buckets
//param:
//     默认不使用任何参数，使用用户配置的AK SK，列出由该AK SK所属用户组所有被授权的bucket
//return:
//     []string: 由bucket name组成的slice
//     error: 正常返回nil，异常返回error Code
//example:
//     Not available now
func (c *FDSClient) List_Authorized_Buckets() ([]string, error) {
	bucketlist := []string{}
	url := c.GetBaseUri() + "?authorizedBuckets"
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "GET",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return bucketlist, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return bucketlist, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		// 修复因为返回值为空导致json解析失败问题
		if string(body) == "" {
			return bucketlist, nil
		}
		var sj map[string]interface{}
		err := json.Unmarshal(body, &sj)
		if err != nil {
			return bucketlist, Model.NewFDSError(err.Error(), -1)
		}
		buckets, ok := sj["buckets"]
		if !ok {
			return bucketlist, Model.NewFDSError(err.Error(), -1)
		}
		bucketsList, ok := buckets.([]interface{})
		if !ok {
			return bucketlist, Model.NewFDSError(err.Error(), -1)
		}
		for _, bucket := range bucketsList {
			// fmt.Printf("%#v\n", bucket.(map[string]interface{})["name"])
			bucket = bucket.(map[string]interface{})["name"]
			bucketlist = append(bucketlist, bucket.(string))
		}
		return bucketlist, nil
	} else {
		return bucketlist, Model.NewFDSError(string(body), res.StatusCode)
	}
}

func (c *FDSClient) Create_Bucket(bucketname string) (bool, error) {
	url := c.GetUploadURL() + bucketname
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "PUT",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return true, nil
	} else {
		return false, Model.NewFDSError(string(body), res.StatusCode)
	}
}

func (c *FDSClient) Delete_Bucket(bucketname string) (bool, error) {
	url := c.GetBaseUri() + bucketname
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "DELETE",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return true, nil
	} else {
		return false, Model.NewFDSError(string(body), res.StatusCode)
	}
}

func (c *FDSClient) Is_Object_Exists(bucketname, objectname string) (bool, error) {
	url := c.GetBaseUri() + bucketname + DELIMITER + objectname
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "HEAD",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return true, nil
	} else if res.StatusCode == 404 {
		return false, nil
	} else {
		return false, Model.NewFDSError(string(body), res.StatusCode)
	}
}

//name:
//     Get_Object
//     获取指定的object
//param:
//     bucketname: 要获取object所属的bucketname
//     objectname: 要获取object的name
//     position:   指定要获取object的其实位置
//     size:       指定要获取object内容的大小，-1位最大，值必须为正数
//return:
//     *FDSObject: 返回与object相关信息
//     error:      正常返回nil，异常返回error Code
//example:
//     Not available now
func (c *FDSClient) Get_Object(bucketname, objectname string, position int64, size int64) (*Model.FDSObject, error) {
	if position < 0 {
		return nil, Model.NewFDSError("Seek position should be no less than 0", -1)
	}
	url := c.GetBaseUri() + bucketname + DELIMITER + objectname
	headers := map[string]string{}
	if position >= 0 && size < 0 {
		headers["range"] = fmt.Sprintf("bytes=%d-", position)
	} else if position >= 0 && size > 0 {
		headers["range"] = fmt.Sprintf("bytes=%d-%d", position, position+size-1)
	} else if position >= 0 && size == 0 {
		return nil, Model.NewFDSError("Request size should be larger than 0", -1)
	} else {
		return nil, Model.NewFDSError("position or size set error", -1)
	}
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "GET",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      &headers,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusPartialContent {
		return &Model.FDSObject{
			BucketName:    bucketname,
			ObjectName:    objectname,
			Metadata:      *Model.NewFDSMetaData(res.Header),
			ObjectContent: body,
		}, nil
	} else {
		return nil, Model.NewFDSError(string(body), res.StatusCode)
	}
}

//name:
//     Get_Object_With_Uri
//     获取使用uri标识的object
//param:
//     uri:      包含要获取object名字的uri，格式为：fds://
//     position: 指定获取object的起始位置
//     size:     指定要获取object内容的大小
//return:
//     *FDSObject: 返回与object相关信息
//     error:      正常返回nil，异常返回error Code
//example:
//     Not available now
func (c *FDSClient) Get_Object_With_Uri(uri string, position, size int64) (*Model.FDSObject, error) {
	bucketName, objectName := Uri_To_Bucket_And_Object(uri)
	return c.Get_Object(bucketName, objectName, position, size)
}

func (c *FDSClient) Get_Object_Reader(bucketname, objectname string, position int64, size int64) (*io.ReadCloser, error) {
	if position < 0 {
		return nil, Model.NewFDSError("Seek position should be no less than 0", -1)
	}
	url := c.GetBaseUri() + bucketname + DELIMITER + objectname
	headers := map[string]string{}
	if position >= 0 && size < 0 {
		headers["range"] = fmt.Sprintf("bytes=%d-", position)
	} else if position >= 0 && size > 0 {
		headers["range"] = fmt.Sprintf("bytes=%d-%d", position, position+size-1)
	} else if position >= 0 && size == 0 {
		return nil, Model.NewFDSError("Request size should be larger than 0", -1)
	} else {
		return nil, Model.NewFDSError("position or size set error", -1)
	}
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "GET",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      &headers,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusPartialContent {
		return &res.Body, nil
	} else {
		body, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, Model.NewFDSError(err.Error(), res.StatusCode)
		}
		return nil, Model.NewFDSError(string(body), res.StatusCode)
	}
}

//name:
//     Download_Object
//     将指定的object下载到本地文件，并不会判断本地是否存在同名文件，如果存在同名文件则文件内容会被重写，文件inode可能并不会改变,因为文件
//param:
//     bucketname: 使用的bucket的名字
//     objectname: 要下载的object的名字
//     filename:   本地要写入的文件名字
//return:
//     *string: 返回服务器端该文件的md5值，用户校验本地文件是否完整
//     error: 正常返回nil，异常返回error Code
//example:
//     Not available now
func (c *FDSClient) Download_Object(bucketname, objectname, filename string) (*string, error) {
	if _, err := os.Stat(filename); os.IsExist(err) {
		return nil, Model.NewFDSError("File exists", -1)
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
	defer file.Close()

	bufferdWriter := bufio.NewWriter(file)

	meta, err := c.Get_Object_Meta(bucketname, objectname)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	contentLength, err := meta.GetMetadataContentLength()
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	slices := int64(math.Ceil(float64(contentLength) / float64(SLICE_SIZE)))
	md5sum, err := meta.GetContentMD5()
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}

	// log.Printf("contentLength %d\n", contentLength)
	// log.Printf("Slices %d\n", slices)

	url := c.GetBaseUri() + bucketname + DELIMITER + objectname
	headers := map[string]string{}
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "GET",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      &headers,
	}
	var i int64 = 0
	// 如果要下载的文件大于50MB，则按照每个50MB分段下载，最后一个分片可以小于50MB
	for ; i < slices; i++ {
		var partSize int64
		partStartPosition := SLICE_SIZE * i
		if (i + 1) == slices {
			partSize = contentLength - i*SLICE_SIZE
		} else {
			partSize = SLICE_SIZE
		}
		headers["range"] = fmt.Sprintf("bytes=%d-%d", partStartPosition, partStartPosition+partSize-1)
		res, err := c.Auth(auth)
		if err != nil {
			return nil, Model.NewFDSError(err.Error(), -1)
		}
		body, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, Model.NewFDSError(err.Error(), -1)
		}
		if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusPartialContent {
			bufferdWriter.Write(body)
		} else {
			return nil, Model.NewFDSError(string(body), res.StatusCode)
		}
		bufferdWriter.Flush()
	}
	return &md5sum, nil
}

//name:
//     Uri_To_Bucket_And_Object
//     将指定Uri转换成bucketname和objectname
//param:
//     url: 要下载的object的uri地址，格式为：fds://bucketname/objectname
//return:
//     string: bucket name
//     string: object name
//example:
//     Not available now
func Uri_To_Bucket_And_Object(uri string) (string, string) {
	if !strings.HasPrefix(uri, "fds://") {
		return "", ""
	}
	bucketObjectPair := strings.Split(uri[6:], "/")
	if len(bucketObjectPair) >= 2 {
		bucketName := bucketObjectPair[0]
		objectName := bucketObjectPair[1]
		return bucketName, objectName
	}
	return "", ""
}

//name:
//     Download_Object_With_Uri
//     将指定Uri的object下载到本地文件，并不会判断本地是否存在同名文件，如果存在同名文件则文件内容会被重写，文件inode可能并不会改变,因为文件
//param:
//     url: 要下载的object的uri地址，格式为：fds://bucketname/objectname
//     filename:   本地要写入的文件名字
//return:
//     string: bucketname
//     string: objectname
//example:
//     Not available now
func (c *FDSClient) Download_Object_With_Uri(url, filename string) (*string, error) {
	bucketNmme, objectName := Uri_To_Bucket_And_Object(url)
	return c.Download_Object(bucketNmme, objectName, filename)
}

// prefix需要改进
func (c *FDSClient) List_Object(bucketname, prefix, delimiter string, maxKeys int) (*Model.FDSObjectListing, error) {
	urlStr := c.GetBaseUri() + bucketname
	auth := FDSAuth{
		UrlBase:      urlStr,
		Method:       "GET",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
		Params: &map[string]string{
			"prefix":    prefix,
			"delimiter": delimiter,
			"maxKeys":   strconv.Itoa(maxKeys),
		},
	}
	res, err := c.Auth(auth)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return Model.NewFDSObjectListing(body)
	} else {
		return nil, Model.NewFDSError(string(body), res.StatusCode)
	}
}

//name:
//     List_Trash_Object
//description:
//     列出用户配置的AK SK，列出由该AK SK所在org下响应bucket中被删除的object
//param:
//     prefix: The prefix of bucket_name/object_name
//     delimiter: The delimiter used in listing
//     maxKeys: 每次获取最大的bucket数量
//return:
//     *Model.FDSObjectListing: 包含object信息的结构体
//     error: 正常返回nil，异常返回error Code
//example:
//     Not available now
func (c *FDSClient) List_Trash_Object(prefix, delimiter string, maxKeys int) (*Model.FDSObjectListing, error) {
	urlStr := c.GetBaseUri() + "trash" //+ "?authorizedObjects"
	auth := FDSAuth{
		UrlBase:      urlStr,
		Method:       "GET",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
		Params: &map[string]string{
			"prefix":    prefix,
			"delimiter": delimiter,
			"maxKeys":   strconv.Itoa(maxKeys),
		},
	}
	res, err := c.Auth(auth)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return Model.NewFDSObjectListing(body)
	} else {
		return nil, Model.NewFDSError(string(body), res.StatusCode)
	}
}

func (c *FDSClient) List_Multipart_Uploads(bucketName, prefix,
	delimiter string, maxKeys int) (*Model.FDSListMultipartUploadsResult, error) {
	url := c.GetBaseUri() + bucketName
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "GET",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
		Params: &map[string]string{
			"uploads":   "",
			"prefix":    prefix,
			"delimiter": delimiter,
			"maxKeys":   strconv.Itoa(maxKeys),
		},
	}
	res, err := c.Auth(auth)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		listMultipartUploadsResult, err := Model.NewFDSListMultipartUploadsResult(body)
		if err != nil {
			return nil, Model.NewFDSError(err.Error(), -1)
		}
		return listMultipartUploadsResult, nil
	} else {
		return nil, Model.NewFDSError(string(body), res.StatusCode)
	}
}

func (c *FDSClient) List_Parts(bucketName, objectName, uploadId string) (*Model.UploadPartList, error) {
	url := c.GetBaseUri() + bucketName + DELIMITER + objectName
	headers := map[string]string{}
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "GET",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      &headers,
		Params: &map[string]string{
			"uploadId": uploadId,
		},
	}
	res, err := c.Auth(auth)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == http.StatusOK {
		return Model.NewUploadPartList(body)
	} else {
		return nil, Model.NewFDSError(string(body), res.StatusCode)
	}
}

func (c *FDSClient) List_Next_Batch_Of_Objects(previous *Model.FDSObjectListing) (*Model.FDSObjectListing, error) {
	if !previous.Truncated {
		return nil, errors.New("No more objects")
	}
	bucketName := previous.BucketName
	prefix := previous.Prefix
	delimiter := previous.Delimiter
	marker := previous.NextMarker
	maxKeys := previous.MaxKeys
	url := c.GetBaseUri() + bucketName

	auth := FDSAuth{
		UrlBase:      url,
		Method:       "GET",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
		Params: &map[string]string{
			"prefix":    prefix,
			"delimiter": delimiter,
			"maxKeys":   strconv.Itoa(maxKeys),
			"marker":    marker,
		},
	}
	res, err := c.Auth(auth)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return Model.NewFDSObjectListing(body)
	} else {
		return nil, Model.NewFDSError(string(body), res.StatusCode)
	}
}

// v1类型：objectname由服务端随机生成唯一名字
func (c *FDSClient) Post_Object(bucketname string, data []byte, filetype string) (string, error) {
	url := c.GetBaseUri() + bucketname + DELIMITER
	if !strings.HasPrefix(filetype, ".") {
		filetype = "." + filetype
	}
	content_type := mime.TypeByExtension(filetype)
	if content_type == "" {
		content_type = "application/octet-stream"
	}
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "POST",
		Data:         data,
		Content_Md5:  "",
		Content_Type: content_type,
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return "", Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return "", Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		var sj map[string]interface{}
		err := json.Unmarshal(body, &sj)
		if err != nil {
			return "", Model.NewFDSError(err.Error(), -1)
		}
		objectname, ok := sj["objectName"]
		if !ok {
			return "", Model.NewFDSError(err.Error(), -1)
		}
		objectNameStr, ok := objectname.(string)
		if !ok {
			return "", Model.NewFDSError(err.Error(), -1)
		}
		return objectNameStr, nil
	} else {
		return "", Model.NewFDSError(string(body), res.StatusCode)
	}
}

// v2类型  自定义文件名 如果object已存在，将会覆盖
func (c *FDSClient) Put_Object(bucketname string, objectname string,
	data []byte, contentType string,
	headers *map[string]string) (*Model.PutObjectResult, error) {
	url := c.GetUploadURL() + bucketname + DELIMITER + objectname
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	md5sum := fmt.Sprintf("%x", md5.Sum(data))
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "PUT",
		Data:         data,
		Content_Md5:  md5sum,
		Content_Type: contentType,
		Headers:      headers,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return Model.NewPutObjectResult(body)
	} else {
		return nil, Model.NewFDSError(string(body), res.StatusCode)
	}
}

//name:
//     Put_Object_With_Uri
//description:
//     通过用户指定的URI上传object
//param:
//     url:         上传object所使用的uri
//     data:        object内容
//     contentType: object内容类型
//     headers:     头信息
//return:
//     *Model.PutObjectResult: 上传结果信息
//     error: 正常返回nil，异常返回error Code
//example:
//     Not available now
func (c *FDSClient) Put_Object_With_Uri(url string, data []byte, contentType string,
	headers *map[string]string) (*Model.PutObjectResult, error) {
	bucketName, objectName := Uri_To_Bucket_And_Object(url)
	return c.Put_Object(bucketName, objectName, data, contentType, headers)
}

func checkNotEmpty(s string) bool {
	return len(s) > 0
}

func (c *FDSClient) Delete_Object(bucketname, objectname string) (bool, error) {
	if !checkNotEmpty(bucketname) || !checkNotEmpty(objectname) {
		return false, errors.New("empty argument")
	}
	url := c.GetBaseUri() + bucketname + DELIMITER + objectname
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "DELETE",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return true, nil
	} else {
		return false, Model.NewFDSError(string(body), res.StatusCode)
	}
}

func (c *FDSClient) Rename_Object(bucketname, src_objectname, dst_objectname string) (bool, error) {

	url := c.GetUploadURL() + bucketname + DELIMITER + src_objectname +
		"?renameTo=" + dst_objectname
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "PUT",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return true, nil
	} else {
		return false, Model.NewFDSError(string(body), res.StatusCode)
	}
}

func (c *FDSClient) Prefetch_Object(bucketname, objectname string) (bool, error) {
	url := c.GetUploadURL() + bucketname + DELIMITER + objectname + "?prefetch"
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "PUT",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return true, nil
	} else {
		return false, Model.NewFDSError(string(body), res.StatusCode)
	}
}

func (c *FDSClient) Refresh_Object(bucketname, objectname string) (bool, error) {
	url := c.GetUploadURL() + bucketname + DELIMITER + objectname + "?refresh"
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "PUT",
		Data:         nil,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return true, nil
	} else {
		return false, Model.NewFDSError(string(body), res.StatusCode)
	}
}

func (c *FDSClient) Set_Object_Acl(bucketname, objectname string, acl map[string]interface{}) (bool, error) {
	acp := make(map[string]interface{})
	acp["owner"] = map[string]string{"id": c.AppKey}
	acp["accessControlList"] = []interface{}{acl}
	jsonString, _ := json.Marshal(acp)
	url := c.GetUploadURL() + bucketname + DELIMITER + objectname + "?acl"
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "PUT",
		Data:         jsonString,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return true, nil
	} else {
		return false, Model.NewFDSError(string(body), res.StatusCode)
	}
}

//name:
//     Set_Object_Acl_New
//description:
//     修改指定object的ACL规则
//param:
//     bucketname:  要修改ACL的object所在的bucket
//     objectname:  要修改ACL的object
//     acl:         要设置的ACL规则
//return:
//     bool:  如果执行正常则返回true，发生错误是返回false
//     error: 正常返回nil，异常返回error Code
//example:
//     Not available now
func (c *FDSClient) Set_Object_Acl_New(bucketname, objectname string, acl Model.ACL) (bool, error) {
	jsonString, _ := json.Marshal(acl)
	url := c.GetUploadURL() + bucketname + DELIMITER + objectname + "?acl"
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "PUT",
		Data:         jsonString,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return true, nil
	} else {
		return false, Model.NewFDSError(string(body), res.StatusCode)
	}
}

//name:
//     Get_Object_ACL
//description:
//     获取指定Object的ACL规则
//param:
//     bucketname:  要获取ACL的Object所在的bucket
//     objectname:  要回去ACL的Object
//return:
//     *Model.ACL:  获取到ACL结构体
//     error: 正常返回nil，异常返回error Code
//example:
//     Not available now
func (c *FDSClient) Get_Object_ACL(bucketname, objectname string) (*Model.ACL, error) {
	url := c.GetUploadURL() + bucketname + DELIMITER + objectname + "?acl"
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "GET",
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		acl, err := Model.NewACL(body)
		if err != nil {
			return nil, nil
		}
		return acl, nil
	} else {
		return nil, Model.NewFDSError(string(body), res.StatusCode)
	}
}

//name:
//     Delete_Object_ACL
//description:
//     删除指定Object的ACL规则
//param:
//     bucketname:  要获取ACL的Object所在的bucket
//     objectname:  要回去ACL的Object
//     acl:         要删除的ACL
//return:
//     bool:  如果执行正常则返回true，发生错误是返回false
//     error: 正常返回nil，异常返回error Code
//example:
//     Not available now
func (c *FDSClient) Delete_Object_ACL(bucketname, objectname string, acl Model.ACL) (bool, error) {
	jsonString, _ := json.Marshal(acl)
	url := c.GetUploadURL() + bucketname + DELIMITER + objectname + "?acl"
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "PUT",
		Data:         jsonString,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
		Params: &map[string]string{
			"action": "delete",
		},
	}
	res, err := c.Auth(auth)
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return true, nil
	} else {
		return false, Model.NewFDSError(string(body), res.StatusCode)
	}
}

//name:
//     Set_Bucket_ACL
//description:
//     设置指定bucket的ACL规则
//param:
//     bucketname:  要修改ACL规则的bucket
//     acl:         ACL规则
//return:
//     bool:  如果执行正常则返回true，发生错误是返回false
//     error: 正常返回nil，异常返回error Code
//example:
//     Not available now
func (c *FDSClient) Set_Bucket_ACL(bucketname string, acl Model.ACL) (bool, error) {
	jsonString, _ := json.Marshal(acl)
	url := c.GetUploadURL() + bucketname + "?acl"
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "PUT",
		Data:         jsonString,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return true, nil
	} else {
		return false, Model.NewFDSError(string(body), res.StatusCode)
	}
}

//name:
//     Delete_Bucket_ACL
//description:
//     删除指定bucket的ACL规则
//param:
//     bucketname:  要删除ACL的bucket
//     acl:         要删除的ACL
//return:
//     bool:  如果执行正常则返回true，发生错误是返回false
//     error: 正常返回nil，异常返回error Code
//example:
//     Not available now
func (c *FDSClient) Delete_Bucket_ACL(bucketname string, acl Model.ACL) (bool, error) {
	jsonString, _ := json.Marshal(acl)
	url := c.GetUploadURL() + bucketname + "?acl"
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "PUT",
		Data:         jsonString,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
		Params: &map[string]string{
			"action": "delete",
		},
	}
	res, err := c.Auth(auth)
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return true, nil
	} else {
		return false, Model.NewFDSError(string(body), res.StatusCode)
	}
}

//name:
//     Get_Bucket_ACL
//description:
//     获取指定bucket的ACL规则
//param:
//     bucketname:  要获取ACL的bucket
//return:
//     bool:  如果执行正常则返回true，发生错误是返回false
//     error: 正常返回nil，异常返回error Code
//example:
//     Not available now
func (c *FDSClient) Get_Bucket_ACL(bucketname string) (*Model.ACL, error) {
	url := c.GetUploadURL() + bucketname + "?acl"
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "GET",
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		acl, err := Model.NewACL(body)
		if err != nil {
			return nil, nil
		}
		return acl, nil
	} else {
		return nil, Model.NewFDSError(string(body), res.StatusCode)
	}
}

func (c *FDSClient) Set_Public(bucketname, objectname string, disable_prefetch bool) (bool, error) {
	grant := map[string]interface{}{
		"grantee":    ALL_USERS,
		"type":       PERMISSION_GROUP,
		"permission": string(PERMISSION_READ),
	}
	// acl := make(map[string]interface{})
	// key := ALL_USERS["id"] + ":" + PERMISSION_GROUP
	// acl[key] = grant
	// result := Set_Object_Acl(bucketname, objectname, acl)
	_, err := c.Set_Object_Acl(bucketname, objectname, grant)
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	if !disable_prefetch {
		_, err := c.Prefetch_Object(bucketname, objectname)
		if err != nil {
			return false, Model.NewFDSError(err.Error(), -1)
		}
	}
	return true, nil
}

func (c *FDSClient) Init_MultiPart_Upload(bucketname, objectname string, contentType string) (*Model.InitMultipartUploadResult, error) {
	url := c.GetUploadURL() + bucketname + DELIMITER + objectname
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	md5sum := fmt.Sprintf("%x", md5.Sum([]byte("")))
	auth := FDSAuth{
		UrlBase:      url,
		Method:       "PUT",
		Data:         []byte(""),
		Content_Md5:  md5sum,
		Content_Type: contentType,
		Headers:      &map[string]string{"x-xiaomi-estimated-object-size": "1000000000"},
		Params: &map[string]string{
			"uploads": "",
		},
	}
	res, err := c.Auth(auth)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode != 200 {
		return nil, Model.NewFDSError(string(body), res.StatusCode)
	}

	return Model.NewInitMultipartUploadResult(body)
}

func (c *FDSClient) Upload_Part(initUploadPartResult *Model.InitMultipartUploadResult, partnumber int, data []byte) (*Model.UploadPartResult, error) {
	bucketname := initUploadPartResult.BucketName
	objectname := initUploadPartResult.ObjectName
	uploadId := initUploadPartResult.UploadId
	url := c.GetUploadURL() + bucketname + DELIMITER + objectname
	auth := FDSAuth{
		UrlBase:     url,
		Method:      "PUT",
		Data:        data,
		Content_Md5: "",
		Headers:     nil,
		Params: &map[string]string{
			"uploadId":   uploadId,
			"partNumber": strconv.Itoa(partnumber),
		},
	}
	res, err := c.Auth(auth)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode != 200 {
		return nil, Model.NewFDSError(string(body), res.StatusCode)
	}
	return Model.NewUploadPartResult(body)
}

// Complete_Multipart_Upload completes multipart upload
// Deprecated: Use CompleteMultipartUpload
func (c *FDSClient) Complete_Multipart_Upload(initPartuploadResult *Model.InitMultipartUploadResult,
	uploadPartResultList *Model.UploadPartList) (*Model.PutObjectResult, error) {
	return c.CompleteMultipartUpload(initPartuploadResult, http.Header{}, uploadPartResultList)
}

// CompleteMultipartUpload completes multipart upload
func (c *FDSClient) CompleteMultipartUpload(initResult *Model.InitMultipartUploadResult,
	metadata http.Header, uploadPartResultList *Model.UploadPartList) (*Model.PutObjectResult, error) {
	bucketName := initResult.BucketName
	objectName := initResult.ObjectName
	uploadID := initResult.UploadId
	url := fmt.Sprintf("%s%s/%s", c.GetUploadURL(), bucketName, objectName)
	uploadPartResultListByteArray, err := json.Marshal(*uploadPartResultList)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}

	internalHeader := map[string]string{}
	for k := range metadata {
		if k != "Content-Type" {
			internalHeader[k] = metadata.Get(k)
		}
	}

	auth := FDSAuth{
		UrlBase:      url,
		Method:       "PUT",
		Data:         uploadPartResultListByteArray,
		Content_Md5:  "",
		Content_Type: metadata.Get("Content-Type"),
		Headers:      &internalHeader,
		Params: &map[string]string{
			"uploadId": uploadID,
		},
	}
	res, err := c.Auth(auth)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode != 200 {
		return nil, Model.NewFDSError(string(body), res.StatusCode)
	}
	return Model.NewPutObjectResult(body)
}

func (c *FDSClient) Abort_MultipartUpload(initPartuploadResult *Model.InitMultipartUploadResult) error {
	bucketName := initPartuploadResult.BucketName
	objectName := initPartuploadResult.ObjectName
	uploadId := initPartuploadResult.UploadId
	url := c.GetUploadURL() + bucketName + DELIMITER + objectName
	auth := FDSAuth{
		UrlBase:     url,
		Method:      "DELETE",
		Data:        nil,
		Content_Md5: "",
		Headers:     nil,
		Params: &map[string]string{
			"uploadId": uploadId,
		},
	}
	res, err := c.Auth(auth)
	if err != nil {
		return Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode != 200 {
		return Model.NewFDSError(string(body), res.StatusCode)
	}
	return nil
}

func (c *FDSClient) Get_Object_Meta(bucketname, objectname string) (*Model.FDSMetaData, error) {
	url := c.GetBaseUri() + bucketname +
		DELIMITER + objectname + "?metadata"
	auth := FDSAuth{
		UrlBase:     url,
		Method:      "GET",
		Data:        nil,
		Content_Md5: "",
		Headers:     nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode != 200 {
		return nil, Model.NewFDSError(string(body), res.StatusCode)
	}
	return Model.NewFDSMetaData(res.Header), nil
}

// SetObjectMetadata method will change object's metadata without puting object
func (c *FDSClient) SetObjectMetadata(bucketname string, objectname string, metadata Model.FDSMetaData) (bool, error) {
	url := c.GetBaseUri() + bucketname + DELIMITER + objectname + "?setMetaData"

	data, err := metadata.Serialize()
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	// md5sum := fmt.Sprintf("%x", md5.Sum(data))

	auth := FDSAuth{
		UrlBase:      url,
		Method:       "PUT",
		Data:         data,
		Content_Md5:  "",
		Content_Type: "",
		Headers:      nil,
	}
	res, err := c.Auth(auth)
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}

	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return false, Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode == 200 {
		return true, nil
	}

	return false, Model.NewFDSError(string(body), res.StatusCode)
}

// Generate_Presigned_URI generates presigned uri
// Deprecated: Use GeneratePresignedURI
func (c *FDSClient) Generate_Presigned_URI(bucketname, objectname, method string, expiration int64, headers map[string][]string) (string, error) {
	return c.GeneratePresignedURI(bucketname, objectname, method, nil, expiration, headers)
}

// GeneratePresignedURI generates presigned uri for all actions
func (c *FDSClient) GeneratePresignedURI(bucketname, objectname, method string, subResources []string, expiration int64, headers map[string][]string) (string, error) {
	var queryString string
	if len(subResources) != 0 {
		queryString = fmt.Sprintf("?%s", strings.Join(subResources, "&"))
	}
	if len(objectname) != 0 {
		objectname = fmt.Sprintf("/%s", objectname)
	}

	urlStr := fmt.Sprintf("%s%s%s%s", c.GetBaseUri(), bucketname, objectname, queryString)

	urlParsed, err := url.Parse(urlStr)
	if err != nil {
		return "", Model.NewFDSError(err.Error(), -1)
	}
	params := urlParsed.Query()
	if method == "HEAD" {
		params.Add("metadata", "")
	}
	params.Add(GALAXY_ACCESS_KEY_ID, c.AppKey)
	params.Add(EXPIRES, fmt.Sprintf("%d", expiration))
	urlParsed.RawQuery = params.Encode()
	signature, err := Signature(c.AppSecret, method, urlParsed.String(), headers)
	if err != nil {
		return "", Model.NewFDSError(err.Error(), -1)
	}

	//params.Add(SIGNATURE, signature)
	//urlParsed.RawQuery = params.Encode()
	return urlParsed.String() + "&" + SIGNATURE + "=" + signature, nil
}

func (c *FDSClient) Delete_Objects(bucketname string, prefix []string) error {
	url := c.GetUploadURL() + bucketname
	prefixJson, err := json.Marshal(prefix)
	if err != nil {
		return Model.NewFDSError(err.Error(), -1)
	}
	auth := FDSAuth{
		UrlBase:     url,
		Method:      "PUT",
		Data:        prefixJson,
		Content_Md5: "",
		Headers:     nil,
		Params: &map[string]string{
			"deleteObjects": "",
		},
	}
	res, err := c.Auth(auth)
	if err != nil {
		return Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode != 200 {
		return Model.NewFDSError(string(body), res.StatusCode)
	}
	return nil
}

//name:
//     Restore_Object
//description:
//     恢复指定删除的object
//param:
//     bucketname： 被删除object原来所在的bucket
//     objectname： 被删除的object名字
//return:
//     error: 正常返回nil，异常返回error Code
//example:
//     Not available now
// TODO 防止restore时替换掉原来的Object的冲突检测
func (c *FDSClient) Restore_Object(bucketname, objectname string) error {
	url := c.GetBaseUri() + bucketname + "/" + objectname

	auth := FDSAuth{
		UrlBase:     url,
		Method:      "PUT",
		Content_Md5: "",
		Headers:     nil,
		Params: &map[string]string{
			"restore": "",
		},
	}

	res, err := c.Auth(auth)
	if err != nil {
		return Model.NewFDSError(err.Error(), -1)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return Model.NewFDSError(err.Error(), -1)
	}
	if res.StatusCode != 200 {
		return Model.NewFDSError(string(body), -1)
	}
	return nil
}

func (c *FDSClient) Delete_Objects_With_Prefix(bucketname, prefix string) error {
	listObjectResult, err := c.List_Object(bucketname, prefix, "", DEFAULT_LIST_MAX_KEYS)
	if err != nil {
		return Model.NewFDSError(err.Error(), -1)
	}

	for true {
		prefixArray := []string{}
		for _, k := range listObjectResult.ObjectSummaries {
			prefixArray = append(prefixArray, k.ObjectName)
		}

		err = c.Delete_Objects(bucketname, prefixArray)
		if err != nil {
			return Model.NewFDSError(err.Error(), -1)
		}

		if !listObjectResult.Truncated {
			break
		}
		listObjectResult, err = c.List_Next_Batch_Of_Objects(listObjectResult)
		if err != nil {
			return Model.NewFDSError(err.Error(), -1)
		}
	}

	return nil
}

//name:
//     Generate_Download_Object_Uri
//description:
//     生成object的下载链接
//param:
//     bucketname： 要生成链接的object原来所在的bucket
//     objectname： 要生成链接的object名字
//return:
//     string: 返回指定object的下载地址
//example:
//     Not available now
func (c *FDSClient) Generate_Download_Object_Uri(bucketname, objectname string) string {
	return c.GetBaseUri() + bucketname + DELIMITER + objectname
}

// list_object_next
// generate_presigned_uri
