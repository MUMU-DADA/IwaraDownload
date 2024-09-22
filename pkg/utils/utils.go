package utils

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

// GenXVersion 生成 X-Version
func GenXVersion(apiUrl string) (string, error) {
	u, err := url.Parse(apiUrl)
	if err != nil {
		return "", err
	}

	// 提取文件名
	iwaraFilename := strings.TrimPrefix(u.Path, "/file/")
	if iwaraFilename == "" {
		panic("Could not parse filename from URL")
	}

	// 提取过期时间
	queryParams := u.Query()
	iwaraExpires := queryParams.Get("expires")
	if iwaraExpires == "" {
		panic("Could not parse expires from URL")
	}

	// 计算X-Version
	h := sha1.New()
	h.Write([]byte(fmt.Sprintf("%s_%s_5nFp9kmbNnHdAFhaqMvt", iwaraFilename, iwaraExpires)))
	return hex.EncodeToString(h.Sum(nil)), nil
}
