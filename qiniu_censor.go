package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gojektech/heimdall/httpclient"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

//七牛 内容审核
const (
	qiniuImageCensorApiUrl = "http://ai.qiniuapi.com/v3/image/censor"
	//qiniuAccessKey         = "pHC4IE2wGOLBsN5p7w50GZVtBf1mtMocPw9_dEbK"
	//qiniuSecretKey         = "bYDaT71OQPJiF-ZzyiZpLCzmzXH_rxjQNxvwIodx"
	ScenesPulp       = "pulp"       //图片鉴黄
	ScenesTerror     = "terror"     //图片鉴暴恐
	ScenesPolitician = "politician" //图片敏感人物识别
	ScenesAds        = "ads"        //图片广告识别
)

var (
	defaultTimeOut      = 60 * time.Second
	ErrorNotSupportType = errors.New("not support type")
	httpClient          = httpclient.NewClient(httpclient.WithHTTPTimeout(defaultTimeOut))
)

type CensorResponse struct {
	Code    int32                `json:"code"`
	Message string               `json:"message"`
	Result  CensorResponseResult `json:"result"`
}

type CensorResponseResult struct {
	Suggestion string                 `json:"suggestion"`
	Scenes     map[string]interface{} `json:"scenes"`
}

type CensorReqBody struct {
	Data   map[string]string      `json:"data"`
	Params map[string]interface{} `json:"params"`
}

type QiNiuCensor struct {
}

func NewQiNiuCensor() Censor {
	c := &QiNiuCensor{}
	return c
}

func init() {
	DefaultCensor["qiniu"] = NewQiNiuCensor()
}

func (q *QiNiuCensor) CensorText(p *CensorTextParams) error {
	if p == nil || len(p.Text) == 0 {
		return nil
	}
	return nil
}

func (q *QiNiuCensor) CensorImage(p *CensorImageParams) error {
	if p == nil || (len(p.ImageUrl) == 0 && len(p.ImageBase64) == 0) {
		return nil
	}
	for _, item := range p.ImageUrl {
		err := imageCensor(item)
		if err != nil {
			return err
		}
	}
	for _, item := range p.ImageBase64 {
		err := imageCensor(item)
		if err != nil {
			return err
		}
	}
	return nil
}

//参见 https://developer.qiniu.com/censor/api/5588/image-censor
//图片资源。支持两种资源表达方式：
//1. 网络图片URL地址，支持http及https；
//2. 图片 base64 编码字符串，需在编码字符串前加上前缀 data:application/octet-stream;base64, 例：data:application/octet-stream;base64,xxx
func imageCensor(dataUri string, scenes ...string) error {
	if dataUri == "" {
		return nil
	}
	contentType := "application/json"
	body := &CensorReqBody{
		Data: map[string]string{
			"uri": dataUri,
		},
		Params: map[string]interface{}{
			"scenes": scenes,
		},
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	token, err := generateQiniuToken("POST", qiniuImageCensorApiUrl, contentType, string(buf))
	if err != nil {
		return err
	}
	headers := map[string][]string{
		"Content-Type":  {contentType},
		"Authorization": {token},
	}
	response, err := httpClient.Post(qiniuImageCensorApiUrl, bytes.NewReader(buf), headers)
	if err != nil {
		return err
	}
	b := make([]byte, 1024)
	i, err := response.Body.Read(b)
	if err != nil && err != io.EOF {
		return err
	}
	if response.StatusCode != http.StatusOK {
		text := string(b[:i])
		if strings.Contains(text, "not support") || strings.Contains(text, "Invalid image") || strings.Contains(text, "fetch uri failed") {
			return ErrorNotSupportType
		}
		return errors.New(fmt.Sprintf("resBody: %s, statusCode: %d", text, response.StatusCode))
	}
	res := &CensorResponse{}
	err = json.Unmarshal(b[:i], res)
	if err != nil {
		return err
	}
	if res.Result.Suggestion != "pass" {
		return ErrorIllegalImg
	}
	return nil
}

//参见 https://developer.qiniu.com/kodo/kb/3702/QiniuToken
//data = <Method> + " " + <Path> + "?<RawQuery>" + "\nHost: " + <Host> + "\nContent-Type: " + <contentType> + "\n\n" + <bodyStr>
func generateQiniuToken(method, apiUrl, contentType, bodyStr string) (string, error) {
	//cfg := config.GetConfig().ImgTextFilterConf
	uri, err := url.Parse(apiUrl)
	if err != nil {
		return "", err
	}
	path := uri.Path
	if uri.RawQuery != "" {
		path = path + "?" + uri.RawQuery
	}
	data := method + " " + path
	data += "\nHost: " + uri.Host
	if contentType != "" {
		data += "\nContent-Type: " + contentType
	}
	data += "\n\n"
	if bodyStr != "" && contentType != "" && contentType != "application/octet-stream" {
		data += bodyStr
	}
	h := hmac.New(sha1.New, []byte(secretKey))
	h.Write([]byte(data))
	d := h.Sum(nil)

	encodedSign := base64.RawURLEncoding.EncodeToString(d)
	token := "Qiniu " + secretId + ":" + encodedSign
	return token, nil
}
