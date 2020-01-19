# Warning
这个FDS client进入维护阶段，不会再添加新的功能，仅修复重大bug。

对于新用户，请使用: [github.com/XiaoMi/go-fds](https://github.com/XiaoMi/go-fds).

对于老用户，如果有深度使用fds go client的需求，强烈建议迁移到新版。

旧版本存在一下问题：

1. 代码结构不清晰，不利于维护
2. 不支持流式接口，也很难支持
3. 不支持Context
4. 未遵循Go的一些默认代码规范

# galaxy-fds-sdk-golang

Golang SDK for Xiaomi File Data Storage.

# install

<mark>内部用户请不要使用go get 方式获取，直接克隆代码即可</mark>

```
go get github.com/XiaoMi/galaxy-fds-sdk-golang
```

#example

```
package main

import (
	fds "github.com/XiaoMi/galaxy-fds-sdk-golang"
	"log"
	"os"
)

const (
	BUCKET_NAME = "test"
	APP_KEY     = "APP_KEY"
	SECRET_KEY  = "SECRET_KEY"
	REGION_NAME = "cnbj0" // region
	END_POINT   = "" // fds domain
)

func main() {
	objectName := "test_object"

	client := fds.NEWFDSClient(APP_KEY, SECRET_KEY, REGION_NAME, END_POINT, false, false)
	client.Create_Bucket(BUCKET_NAME)
	content := []byte("object content data")
	result, err := client.Put_Object(BUCKET_NAME, objectName, content, "", nil)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	log.Println(result.ObjectName)

	fdsObject, err := client.Get_Object(BUCKET_NAME, objectName, 0, -1)
	if err != nil {
		log.Println(err)
		os.Exit(2)
	}
	log.Println(string(fdsObject.ObjectContent))
}

```

# Changes

20170822:
> 1. 修正README.md example中的错误
> 2. fix因为struct内变量名字引起的multi upload失败问题
> 3. 使用go fmt 工具格式化所有代码，使代码风格一致


20170823
> 1. 修复get_object代码中存在的逻辑漏洞

20170829
> 1. 添加方法注释：List_Bucket
> 2. 方法 List_Authorized_Buckets 实现并添加该方法注释
> 3. 方法 List_Trash_Object 实现并添加该方法注释
> 4. fix List_Bucket方法因返回值为空字符串导致json解析失败的问题

20170831
> 1. 添加下载Object到本地的SDK调用接口，Download_Object
> 2. 添加Download_Object_With_Uri接口
> 3. 添加Get_Object_With_Uri 接口

20170906
> 1. 添加Put_Object_With_Uri调用接口

20170907
> 1. 添加Get_Object_Acl调用接口
> 2. 修复NewFDSObjectListing中因为"lastModified":null字段产生的bug
> 3. 修复Grantee struct json注释bug
> 4. 添加新接口Set_Object_Acl_New，原Set_Object_Acl接口保留，使用Set_Object_Acl_New更规范ACL的定义
> 5. 添加Get_Object_ACL接口
> 6. 添加Delete_Object_ACL 接口
> 7. 添加Set_Bucket_ACL 接口
> 8. 添加Delete_Bucket_ACL 接口
> 9. 添加Get_Bucket_ACL 接口
> 10. 添加Generate_Download_Object_Uri 接口

20171013
> 1. 修复readme中的example code中的bug
> 2. 修复 Test/FDSClient_test.go 中的bug
> 3. 添加Get_Bucket 接口，但是该接口与java中语义不同，请谨慎使用，java中该方法不返回任何值

20180612
> 1. 修复download中的分片错误，该错误会导致小于50M的文件无法下载
