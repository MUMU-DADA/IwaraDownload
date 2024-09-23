package request

import (
	"IwaraDownload/model"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type METHOD string

const (
	GET     METHOD = "GET"
	POST    METHOD = "POST"
	PUT     METHOD = "PUT"
	DELETE  METHOD = "DELETE"
	OPTIONS METHOD = "OPTIONS"
)

var (
	lastRequestTimes time.Time         // 上次请求时间
	headers          map[string]string = map[string]string{
		"User-Agent":         ua,
		"Host":               "api.iwara.tv",
		"Sec-Ch-Ua":          `"Not;A=Brand";v="24", "Chromium";v="128"`,
		"Accept":             "application/json",
		"Sec-Ch-Ua-Platform": `"Windows"`,
		"Accept-Language":    "zh-CN,zh;q=0.9",
		"Sec-Ch-Ua-Mobile":   "?0",
		"Content-Type":       "application/json",
		"Origin":             "https://www.iwara.tv",
		"Sec-Fetch-Site":     "same-site",
		"Sec-Fetch-Mode":     "cors",
		"Sec-Fetch-Dest":     "empty",
		"Referer":            "https://www.iwara.tv/",
		"Priority":           "u=1, i",
	}

	DelaySwitch bool = true // 是否开启请求延时
)

const (
	setCookies = ""
	ua         = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.6533.100 Safari/537.36" // 浏览器UA
	reqDelay   = time.Second * 40                                                                                                       // 请求延时
)

// 保存页面文本到文件
func saveWebRspToFile(filePath string, url string, method METHOD, user *model.User, body string) error {
	rsp, err := reqWeb(url, method, user, body, nil)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	// 创建本地文件
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 将数据写入文件
	_, err = io.Copy(file, rsp.Body)
	if err != nil {
		return err
	}

	return nil
}

// 获取页面主页文本
func getWeb(url string, method METHOD, user *model.User, body string, addHeader map[string]string) ([]byte, error) {
	rsp, err := reqWeb(url, method, user, body, addHeader)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	rsp.Cookies()
	return io.ReadAll(rsp.Body)
}

// requestDelay 请求延时
func requestDelay() {
	if !DelaySwitch {
		time.Sleep(time.Second * 3) // 虽然有延时开关,但是至少是3秒
		return
	}

	// 首次的请求不延时
	if lastRequestTimes.IsZero() {
		lastRequestTimes = time.Now()
		return
	}

	if time.Now().After(lastRequestTimes.Add(reqDelay)) {
		time.Sleep(reqDelay)
		lastRequestTimes = time.Now()
		return
	}
	delayTime := lastRequestTimes.Add(reqDelay).Sub(time.Now())
	time.Sleep(delayTime)
	lastRequestTimes = time.Now()
}

const (
	renewCookies = true
)

// 发送请求
func reqWeb(url string, method METHOD, user *model.User, bodyStr string, addHeader map[string]string) (*http.Response, error) {
	requestDelay()
	c := http.Client{}
	req, err := http.NewRequest(string(method), url, strings.NewReader(bodyStr))
	if err != nil {
		return nil, err
	}
	// 设置全局Header
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	jwt := user.GetAuthorization()
	if jwt != "" {
		req.Header.Set("Authorization", jwt)
	}
	for k, v := range addHeader {
		req.Header.Set(k, v)
	}
	// 设置全局Cookie
	for _, cookie := range user.Cookies {
		req.AddCookie(cookie)
	}
	rsp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if rsp.StatusCode != 200 {
		return nil, fmt.Errorf("status code is %v", rsp.StatusCode)
	}
	if renewCookies {
		user.Cookies = rsp.Cookies()
	}
	return rsp, nil
}

func parseCookies(cookieStr string) []*http.Cookie {
	var cookies []*http.Cookie
	for _, pair := range strings.Split(cookieStr, "; ") {
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}
		cookies = append(cookies, &http.Cookie{Name: parts[0], Value: parts[1]})
	}
	return cookies
}

func checkCookies(cookies []*http.Cookie) error {
	if cookies == nil {
		return fmt.Errorf("cookies is nil")
	}
	if len(cookies) == 0 {
		return fmt.Errorf("cookies is empty")
	}
	return nil
}
