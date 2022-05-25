package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	_ "fmt"
	"github.com/tjfoc/gmsm/sm3"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type WangYiCensor struct {
}

func NewWangYiCensor() Censor {
	c := &WangYiCensor{}
	return c
}

func init() {
	DefaultCensor["wangyi"] = NewWangYiCensor()
}

// 参考：https://support.dun.163.com/documents/588434200783982592?docId=589310433773625344
func (w *WangYiCensor) CensorText(p *CensorTextParams) error {
	if p == nil || len(p.Text) == 0 {
		return nil
	}
	var texts []map[string]string
	var errResult error
	errChan := make(chan error)
	go func() {
		for err := range errChan {
			if errResult == nil && err != nil {
				errResult = err
			}
		}
	}()
	concurrent := make(chan struct{}, 5)
	defer func() {
		close(errChan)
		close(concurrent)
	}()
	for index, t := range p.Text {
		if errResult != nil {
			return errResult
		}
		params := make(map[string]string)
		params["dataId"] = genDataId()
		params["content"] = t
		texts = append(texts, params)
		if index < len(p.Text) - 1 && len(texts) < 99 {
			concurrent <- struct{}{}
			go func() {
				defer func() {
					<- concurrent
				}()
				err := w.censorText(texts, p)
				if err != nil {
					errChan <- err
				}
			}()
		}
	}
	return nil
}

func (w *WangYiCensor) censorText(texts []map[string]string, p *CensorTextParams) error {
	params := url.Values{}
	bytes, _ := json.Marshal(texts)
	params["texts"] = []string{string(bytes)}
	params["secretId"] = []string{secretId}
	params["businessId"] = []string{businessId}
	params["version"] = []string{version}
	params["timestamp"] = []string{strconv.FormatInt(time.Now().UnixNano()/1000000, 10)}
	params["nonce"] = []string{strconv.FormatInt(rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(10000000000), 10)}
	params["signature"] = []string{genSignature(params, secretKey)}
	resp, err := http.Post(apiUrl, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil && err != io.EOF {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("res body: %s, statusCode: %d", string(contents), resp.StatusCode))
	}
	res := &WangYiTextCensorResult{}
	err = json.Unmarshal(contents, res)
	if err != nil {
		return err
	}
	fmt.Println("wang contents------> ", string(contents))
	if len(res.Result) == 0 || res.Result["antispam"] == nil || res.Result["antispam"].Suggestion != 0 {
		return ErrorIllegalText
	}
	return nil
}


// 参考：https://support.dun.163.com/documents/588434277524447232?docId=588512292354793472
func (w *WangYiCensor) CensorImage(p *CensorImageParams) error {
	if p == nil || (len(p.ImageUrl) == 0 && len(p.ImageBase64) == 0) {
		return nil
	}
	params := w.buildCensorImageParams(p)
	resp, err := http.Post(imageApiUrl, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil && err != io.EOF {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("res body: %s, statusCode: %d", string(contents), resp.StatusCode))
	}
	res := &WangYiImageCensorResult{}
	err = json.Unmarshal(contents, res)
	if err != nil {
		return err
	}
	if res.Code != http.StatusOK {
		return errors.New(fmt.Sprintf("res body: %s, statusCode: %d", string(contents), res.Code))
	}
	fmt.Println("wang contents------> ", string(contents))
	if len(res.Result) == 0 || len(res.Result) == 0 || res.Result[0]["antispam"] == nil || res.Result[0]["antispam"].Status == 3 || res.Result[0]["antispam"].Suggestion != 0 {
		return ErrorIllegalImg
	}
	return nil
}

func (w *WangYiCensor) buildCensorImageParams(p *CensorImageParams) url.Values {
	params := url.Values{}
	var images []map[string]string
	for _, item := range p.ImageUrl {
		imageUrl := map[string]string{
			"name": item,
			"type": "1",
			"data": item,
		}
		images = append(images, imageUrl)
	}
	for _, item := range p.ImageBase64 {
		imageUrl := map[string]string{
			"name": item,
			"type": "2",
			"data": item,
		}
		images = append(images, imageUrl)
	}
	jsonString, _ := json.Marshal(images)
	params["images"] = []string{string(jsonString)}
	params["secretId"] = []string{secretId}
	params["businessId"] = []string{imageBusinessId}
	params["version"] = []string{imageVersion}
	params["timestamp"] = []string{strconv.FormatInt(time.Now().UnixNano()/1000000, 10)}
	params["nonce"] = []string{strconv.FormatInt(rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(10000000000), 10)}
	params["signature"] = []string{genSignature(params, secretKey)}
	return params
}

func genDataId() string {
	randBytes := make([]byte, 8)
	rand.Read(randBytes)
	return fmt.Sprintf("%x", randBytes)
}

//生成签名信息
func genSignature(params url.Values, secretKey string) string {
	var paramStr string
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		paramStr += key + params[key][0]
	}
	paramStr += secretKey
	if params["signatureMethod"] != nil && params["signatureMethod"][0] == "SM3" {
		sm3Reader := sm3.New()
		sm3Reader.Write([]byte(paramStr))
		return hex.EncodeToString(sm3Reader.Sum(nil))
	} else {
		md5Reader := md5.New()
		md5Reader.Write([]byte(paramStr))
		return hex.EncodeToString(md5Reader.Sum(nil))
	}
}

type WangYiTextCensorResult struct {
	Code   int32                    `json:"code"`
	Msg    string                   `json:"msg"`
	Result map[string]*TextAntispam `json:"result"`
}

type TextAntispam struct {
	TaskId       string        `json:"taskId"`
	DataId       string        `json:"dataId"`
	Suggestion   int32         `json:"suggestion"`
	ResultType   int32         `json:"resultType"`
	CensorType   int32         `json:"censorType"`
	IsRelatedHit bool          `json:"isRelatedHit"`
	Labels       []interface{} `json:"labels"`
}

type WangYiImageCensorResult struct {
	Code   int32                       `json:"code"`
	Msg    string                      `json:"msg"`
	Result []map[string]*ImageAntispam `json:"result"`
}

type ImageAntispam struct {
	TaskId          string `json:"taskId"`
	DataId          string `json:"dataId"`
	Name            string `json:"name"`
	Status          int32  `json:"status"`
	Suggestion      int32  `json:"suggestion"`
	ResultType      int32  `json:"resultType"`
	SuggestionLevel int32  `json:"suggestionLevel"`
}
