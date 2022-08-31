package client

import (
	"ddns-watchdog/internal/common"
	"errors"
	"github.com/bitly/go-simplejson"
	"io"
	"net/http"
	"strings"
)

const DNSPodConfFileName = "dnspod.json"

type DNSPod struct {
	ID           string    `json:"id"`
	Token        string    `json:"token"`
	Domain       string    `json:"domain"`
	SubDomain    subdomain `json:"sub_domain"`
	RecordId     string    `json:"-"`
	RecordLineId string    `json:"-"`
}

func (dpc *DNSPod) InitConf() (msg string, err error) {
	*dpc = DNSPod{}
	dpc.ID = "在 https://console.dnspod.cn/account/token/token 获取"
	dpc.Token = dpc.ID
	dpc.Domain = "example.com"
	dpc.SubDomain.A = "A记录子域名"
	dpc.SubDomain.AAAA = "AAAA记录子域名"
	err = common.MarshalAndSave(dpc, ConfDirectoryName+"/"+DNSPodConfFileName)
	msg = "初始化 " + ConfDirectoryName + "/" + DNSPodConfFileName
	return
}

func (dpc *DNSPod) LoadConf() (err error) {
	err = common.LoadAndUnmarshal(ConfDirectoryName+"/"+DNSPodConfFileName, &dpc)
	if err != nil {
		return
	}
	if Client.Center.Enable {
		if dpc.Domain == "" || (dpc.SubDomain.A == "" && dpc.SubDomain.AAAA == "") {
			err = errors.New("请打开配置文件 " + ConfDirectoryName + "/" + DNSPodConfFileName + " 检查你的 domain, sub_domain 并重新启动")
		}
	} else {
		if dpc.ID == "" || dpc.Token == "" || dpc.Domain == "" || (dpc.SubDomain.A == "" && dpc.SubDomain.AAAA == "") {
			err = errors.New("请打开配置文件 " + ConfDirectoryName + "/" + DNSPodConfFileName + " 检查你的 id, token, domain, sub_domain 并重新启动")
		}
	}
	return
}

func (dpc DNSPod) Run(enabled common.Enable, ipv4, ipv6 string) (msg []string, errs []error) {
	if enabled.IPv4 && dpc.SubDomain.A != "" {
		// 获取解析记录
		recordIP, err := dpc.getParseRecord(dpc.SubDomain.A, "A")
		if err != nil {
			errs = append(errs, err)
		} else if recordIP != ipv4 {
			// 更新解析记录
			err = dpc.updateParseRecord(ipv4, "A", dpc.SubDomain.A)
			if err != nil {
				errs = append(errs, err)
			} else {
				msg = append(msg, "DNSPod: "+dpc.SubDomain.A+"."+dpc.Domain+" 已更新解析记录 "+ipv4)
			}
		}
	}
	if enabled.IPv6 && dpc.SubDomain.AAAA != "" {
		// 获取解析记录
		recordIP, err := dpc.getParseRecord(dpc.SubDomain.AAAA, "AAAA")
		if err != nil {
			errs = append(errs, err)
		} else if recordIP != ipv6 {
			// 更新解析记录
			err = dpc.updateParseRecord(ipv6, "AAAA", dpc.SubDomain.AAAA)
			if err != nil {
				errs = append(errs, err)
			} else {
				msg = append(msg, "DNSPod: "+dpc.SubDomain.AAAA+"."+dpc.Domain+" 已更新解析记录 "+ipv6)
			}
		}
	}
	return
}

func checkRespondStatus(jsonObj *simplejson.Json) (err error) {
	statusCode := jsonObj.Get("status").Get("code").MustString()
	if statusCode != "1" {
		err = errors.New("DNSPod: " + statusCode + ": " + jsonObj.Get("status").Get("message").MustString())
		return
	}
	return
}

func (dpc *DNSPod) getParseRecord(subDomain, recordType string) (recordIP string, err error) {
	postContent := dpc.publicRequestInit()
	postContent = postContent + "&" + dpc.recordRequestInit(subDomain)
	recvJson, err := postman("https://dnsapi.cn/Record.List", postContent)
	if err != nil {
		return
	}

	jsonObj, err := simplejson.NewJson(recvJson)
	if err != nil {
		return
	}
	err = checkRespondStatus(jsonObj)
	if err != nil {
		return
	}
	records, err := jsonObj.Get("records").Array()
	if len(records) == 0 {
		err = errors.New("DNSPod: " + subDomain + "." + dpc.Domain + " 解析记录不存在")
		return
	}
	for _, value := range records {
		element := value.(map[string]any)
		if element["name"].(string) == subDomain {
			dpc.RecordId = element["id"].(string)
			recordIP = element["value"].(string)
			dpc.RecordLineId = element["line_id"].(string)
			if element["type"].(string) == recordType {
				break
			}
		}
	}
	return
}

func (dpc DNSPod) updateParseRecord(ipAddr, recordType, subDomain string) (err error) {
	postContent := dpc.publicRequestInit()
	postContent = postContent + "&" + dpc.recordModifyRequestInit(ipAddr, recordType, subDomain)
	recvJson, err := postman("https://dnsapi.cn/Record.Modify", postContent)
	if err != nil {
		return
	}

	jsonObj, err := simplejson.NewJson(recvJson)
	if err != nil {
		return
	}
	err = checkRespondStatus(jsonObj)
	if err != nil {
		return
	}
	return
}

func (dpc DNSPod) publicRequestInit() (pp string) {
	pp = "login_token=" + dpc.ID + "," + dpc.Token +
		"&format=" + "json" +
		"&lang=" + "cn" +
		"&error_on_empty=" + "no"
	return
}

func (dpc DNSPod) recordRequestInit(subDomain string) (rr string) {
	rr = "domain=" + dpc.Domain +
		"&sub_domain=" + subDomain
	return
}

func (dpc DNSPod) recordModifyRequestInit(ipAddr, recordType, subDomain string) (rm string) {
	rm = "domain=" + dpc.Domain +
		"&record_id=" + dpc.RecordId +
		"&sub_domain=" + subDomain +
		"&record_type=" + recordType +
		"&record_line_id=" + dpc.RecordLineId +
		"&value=" + ipAddr
	return
}

func postman(url, src string) (dst []byte, err error) {
	httpClient := getGeneralHttpClient()
	req, err := http.NewRequest("POST", url, strings.NewReader(src))
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		t := Body.Close()
		if t != nil {
			err = t
		}
	}(req.Body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", RunningName+"/"+common.LocalVersion+" ()")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		t := Body.Close()
		if t != nil {
			err = t
		}
	}(resp.Body)
	dst, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return
}
