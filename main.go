package main

import (
	"errors"
	"fmt"
	_ "net/http/pprof"
	"time"
)

const (
	apiUrl          = "http://as.dun.163.com/v5/text/check"
	version         = "v5.2"
	secretId        = "9280c943101e0c957ae50ebc72770187" //产品密钥ID，产品标识
	secretKey       = "0bceb55435ef9b9224cfa91c5610a112" //产品私有密钥，服务端生成签名信息使用，请严格保管，避免泄露
	businessId      = "d59d7ce4c3b2684215c71e2104c2b5c2" //业务ID，易盾根据产品业务特点分配
	imageApiUrl     = "http://as.dun.163.com/v5/image/check"
	imageBusinessId = "981c7a4afbcbdfe624cd7a3ba6cae5b0"
	imageVersion    = "v5.1"
)

var (
	ErrorIllegalImg  = errors.New("illegal img")
	ErrorIllegalText = errors.New("illegal text")
)

func AA() {
	sem := make(chan struct{}, 5)
	for i := 0; i < 100; i++ {
		sem <- struct{}{}
		go func() {
			defer func() {
				<- sem
			}()
			time.Sleep(2 * time.Second)
		}()
		fmt.Println(i)
	}
}

func main() {
	tp := &CensorTextParams{
		Text: []string{"测试哈哈哈哈哈哈哈"},
	}
	ip := &CensorImageParams{
		ImageUrl: []string{"https://nos.netease.com/yidun/2-0-0-a6133509763d4d6eac881a58f1791976.jpg"},
	}
	censor := DefaultCensor["wangyi"]
	err := censor.CensorText(tp)
	if err != nil {
		fmt.Println("ccc tt ---> ", err)
	}
	err = censor.CensorImage(ip)
	if err != nil {
		fmt.Println("ccc ip ---> ", err)
	}
	//go func() {
	//	log.Println(http.ListenAndServe("localhost:9876", nil))
	//}()
	//AA()
	//
	//select{}
}
