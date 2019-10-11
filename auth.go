package galaxy_fds_sdk_golang

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/url"
	"sort"
	"strings"

	"github.com/XiaoMi/galaxy-fds-sdk-golang/Model"
)

func getDateFromUrl(urlStr string) string {
	urlParsed, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	queryParams := urlParsed.Query()
	d, ok := queryParams["Expires"]

	if !ok || len(d) == 0 {
		return ""
	}
	return d[0]
}

func getStrFromHeader(headers map[string][]string, key string) string {
	a, ok := headers[key]
	if !ok {
		for k, v := range headers {
			if strings.EqualFold(k, key) {
				if len(v) == 0 {
					return ""
				}
				return v[0]
			}
		}
	}
	if len(a) == 0 {
		return ""
	}
	return a[0]
}

func canonicalizeXiaomiHeaders(headers map[string][]string) ([]byte, error) {
	if len(headers) == 0 {
		return nil, nil
	}

	klist := []string{}
	fileredMap := map[string]string{}
	for k, v := range headers {
		key := strings.ToLower(k)
		if !strings.HasPrefix(key, "x-xiaomi-") {
			continue
		}

		fileredMap[key] = strings.Join(v, ",")
		klist = append(klist, key)
	}
	sort.Strings(klist)

	var r bytes.Buffer
	for _, k := range klist {
		r.WriteString(k)
		r.WriteString(":")
		r.WriteString(fileredMap[k])
		r.WriteString("\n")
	}

	return r.Bytes(), nil
}

var SUB_RESOURCE_MAP = map[string]string{
	"acl":                "",
	"quota":              "",
	"uploads":            "",
	"partNumber":         "",
	"uploadId":           "",
	"storageAccessToken": "",
	"metadata":           "",
}

func canonicalizeResource(uri string) ([]byte, error) {
	uriParsed, err := url.Parse(uri)
	if err != nil {
		return nil, Model.NewFDSError(err.Error(), -1)
	}
	var path bytes.Buffer
	path.Write([]byte(uriParsed.Path))

	param := uriParsed.Query()
	filteredKey := []string{}
	filteredMap := map[string]string{}
	for k, v := range param {
		_, ok := SUB_RESOURCE_MAP[k]
		if !ok {
			continue
		}
		filteredKey = append(filteredKey, k)
		if len(v) > 0 {
			filteredMap[k] = v[0]
		} else {
			filteredMap[k] = ""
		}
	}

	if len(filteredKey) == 0 {
		return path.Bytes(), nil
	}

	sort.Strings(filteredKey)

	for i, k := range filteredKey {
		if i == 0 {
			path.WriteString("?")
		} else {
			path.WriteString("&")
		}
		path.WriteString(k)
		if len(filteredMap[k]) > 0 {
			path.WriteString("=")
			path.WriteString(filteredMap[k])
		}
	}

	return path.Bytes(), nil

}

func Signature(app_secret, method, u string, headers map[string][]string) (string, error) {
	var string_to_sign bytes.Buffer
	content_md5 := getStrFromHeader(headers, "content-md5")
	content_type := getStrFromHeader(headers, "content-type")
	date := getDateFromUrl(u)
	if len(date) == 0 {
		date = getStrFromHeader(headers, "date")
	}
	string_to_sign.WriteString(method)
	string_to_sign.WriteString("\n")
	string_to_sign.WriteString(content_md5)
	string_to_sign.WriteString("\n")
	string_to_sign.WriteString(content_type)
	string_to_sign.WriteString("\n")
	string_to_sign.WriteString(date)
	string_to_sign.WriteString("\n")

	/*
		url_str, _ := url.ParseRequestURI(u)
		if strings.Contains(url_str.RequestURI(), "?") {
			uri_list := strings.Split(url_str.RequestURI(), "?")
			if uri_list[1] != "acl" {
				uri = uri_list[0]
			} else {
				uri = url_str.RequestURI()
			}
		} else {
			uri = url_str.RequestURI()
		}*/
	ch, err := canonicalizeXiaomiHeaders(headers)
	if err != nil {
		return "", Model.NewFDSError(err.Error(), -1)
	}
	string_to_sign.Write(ch)
	cr, err := canonicalizeResource(u)
	if err != nil {
		return "", Model.NewFDSError(err.Error(), -1)
	}
	string_to_sign.Write(cr)
	h := hmac.New(sha1.New, []byte(app_secret))
	_, err = h.Write(string_to_sign.Bytes())
	if err != nil {
		return "", Model.NewFDSError(err.Error(), -1)
	}
	b := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return b, nil
}
