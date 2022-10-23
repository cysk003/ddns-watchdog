package client

import (
	"ddns-watchdog/internal/common"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const (
	ConfFileName = "client.json"
)

type client struct {
	APIUrl            apiUrl        `json:"api_url"`
	Center            center        `json:"center"`
	Enable            common.Enable `json:"enable"`
	NetworkCard       networkCard   `json:"network_card"`
	Services          service       `json:"services"`
	CheckCycleMinutes int           `json:"check_cycle_minutes"`
	LatestIPv4        string        `json:"-"`
	LatestIPv6        string        `json:"-"`
}

type apiUrl struct {
	IPv4    string `json:"ipv4"`
	IPv6    string `json:"ipv6"`
	Version string `json:"version"`
}

type center struct {
	Enable bool   `json:"enable"`
	APIUrl string `json:"api_url"`
	Token  string `json:"token"`
}

type networkCard struct {
	Enable bool   `json:"enable"`
	IPv4   string `json:"ipv4"`
	IPv6   string `json:"ipv6"`
}

type service struct {
	DNSPod     bool `json:"dnspod"`
	AliDNS     bool `json:"alidns"`
	Cloudflare bool `json:"cloudflare"`
}

func (conf *client) InitConf() (msg string, err error) {
	*conf = client{}
	conf.APIUrl.IPv4 = common.DefaultAPIUrl
	conf.APIUrl.IPv6 = common.DefaultIPv6APIUrl
	conf.APIUrl.Version = common.DefaultAPIUrl
	conf.CheckCycleMinutes = 0
	err = common.MarshalAndSave(conf, ConfDirectoryName+"/"+ConfFileName)
	msg = "初始化 " + ConfDirectoryName + "/" + ConfFileName
	return
}

func (conf *client) LoadConf() (err error) {
	err = common.LoadAndUnmarshal(ConfDirectoryName+"/"+ConfFileName, &conf)
	if err != nil {
		return
	}
	// 检查启用 IP 类型
	if !conf.Enable.IPv4 && !conf.Enable.IPv6 {
		err = errors.New("请打开客户端配置文件 " + ConfDirectoryName + "/" + ConfFileName + " 启用需要使用的 IP 类型并重新启动")
		return
	}
	// 检查启用服务
	if !conf.Center.Enable && !conf.Services.DNSPod && !conf.Services.AliDNS && !conf.Services.Cloudflare {
		err = errors.New("请打开客户端配置文件 " + ConfDirectoryName + "/" + ConfFileName + " 启用需要使用的服务并重新启动")
		return
	}
	return
}

func (conf client) GetLatestVersion() (str string) {
	resp, err := http.Get(conf.APIUrl.Version)
	if err != nil {
		return "N/A (请检查网络连接)"
	}
	defer func(Body io.ReadCloser) {
		t := Body.Close()
		if t != nil {
			str = t.Error()
		}
	}(resp.Body)
	recvJson, err := io.ReadAll(resp.Body)
	if err != nil {
		return "N/A (数据包错误)"
	}
	recv := common.GetIPResp{}
	err = json.Unmarshal(recvJson, &recv)
	if err != nil {
		return "N/A (数据包错误)"
	}
	if recv.Version == "" {
		return "N/A (没有获取到版本信息)"
	}
	return recv.Version
}

func (conf *client) CheckLatestVersion() {
	if conf.APIUrl.Version == "" {
		conf.APIUrl.Version = common.DefaultAPIUrl
	}
	common.VersionTips(conf.GetLatestVersion())
}
