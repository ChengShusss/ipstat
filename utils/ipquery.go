package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type WrapInfo struct {
	IsProxy bool
	Source  string
	Res     IpInfo `json:"res"`
}

type IpInfo struct {
	gorm.Model
	IpNumber    string  `json:"ipNumber"`
	IpVersion   int     `json:"ipVersion"`
	IpAddress   string  `json:"ipAddress"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	CountryName string  `json:"countryName"`
	CountryCode string  `json:"countryCode"`
	Isp         string  `json:"isp"`
	CityName    string  `json:"cityName"`
	RegionName  string  `json:"regionName"`
}

type Client struct {
	db *gorm.DB
}

const (
	QuerySource = "ip2location"
)

var (
	client *Client
)

func getClient() (*Client, error) {
	if client != nil {
		return client, nil
	}

	db, err := gorm.Open(sqlite.Open("ip_info.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		return nil, err
	}
	client = &Client{
		db: db,
	}

	err = client.db.AutoMigrate(&IpInfo{})

	return client, err
}

func GetIpInfo(ip string) (*IpInfo, error) {
	c, err := getClient()
	if err != nil {
		return nil, err
	}

	var ipInfo IpInfo
	err = c.db.First(&ipInfo, "ip_address = ?", ip).Error
	if err == nil {
		return &ipInfo, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Other err except not found
		return nil, err
	}

	info, err := getIpInfo(ip)
	if err != nil {
		return nil, err
	}

	c.db.Create(info)

	return info, nil
}

func getIpInfo(ip string) (*IpInfo, error) {

	log.Printf("Query IP %v\n", ip)
	time.Sleep(2 * time.Second)

	method := "POST"
	url := "https://www.iplocation.net/get-ipdata"
	payload := strings.NewReader(fmt.Sprintf("ip=%s&source=%s&ipv=4", ip, QuerySource))

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		return nil, err
	}
	req.Header.Add("authority", "www.iplocation.net")
	req.Header.Add("accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Add("accept-language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("content-type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Add("cookie", "PHPSESSID=186355b95a919d42d3561c40332bc2c3")
	req.Header.Add("origin", "https://www.iplocation.net")
	req.Header.Add("pragma", "no-cache")
	req.Header.Add("referer", "https://www.iplocation.net/ip-lookup")
	req.Header.Add("sec-ch-ua", "\"Microsoft Edge\";v=\"119\", \"Chromium\";v=\"119\", \"Not?A_Brand\";v=\"24\"")
	req.Header.Add("sec-ch-ua-mobile", "?0")
	req.Header.Add("sec-ch-ua-platform", "\"Windows\"")
	req.Header.Add("sec-fetch-dest", "empty")
	req.Header.Add("sec-fetch-mode", "cors")
	req.Header.Add("sec-fetch-site", "same-origin")
	req.Header.Add("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36 Edg/119.0.0.0")
	req.Header.Add("x-requested-with", "XMLHttpRequest")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var info WrapInfo
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, err
	}

	return &info.Res, nil
}
