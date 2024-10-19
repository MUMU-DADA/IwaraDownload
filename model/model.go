package model

import (
	"encoding/base64"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type DownloadMode int

const (
	// AllMode 全部模式
	AllMode DownloadMode = iota
	// SubscribeMode 订阅模式
	SubscribeMode
	// HotMode 热门模式
	HotMode
	// ArtistMode 下载指定用户模式
	ArtistMode
)

func (d DownloadMode) String() string {
	return strconv.Itoa(int(d))
}

func (d DownloadMode) Name() string {
	return [...]string{"全部模式", "订阅模式", "热门模式", "指定用户模式"}[d]
}

// User 用户信息
type User struct {

	// ↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓ 登录信息 ↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓
	Username     string         `json:"username"`     // 用户名
	Password     string         `json:"password"`     // 密码
	LoginToken   string         `json:"loginToken"`   // 登录token
	AccessToken  string         `json:"accessToken"`  // 访问token
	HotPageLimit int            `json:"hotPageLimit"` // 热门页下载限制
	Mode         DownloadMode   `json:"mode"`         // 下载模式
	MultiMode    []DownloadMode `json:"multiMode"`    // 多模式下载
	// ↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑ 登录信息 ↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑

	// ↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓ 下载条件 ↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓
	Tags    []string `json:"tags"`    // 下载指定标签
	Artists []string `json:"artists"` // 下载指定用户的内容

	BanArtists []string `json:"banArtists"` // 禁止下载指定用户的内容
	BanTags    []string `json:"banTags"`    // 跳过标签
	LikeLimit  int      `json:"likeLimit"`  // 下载达到目标点赞数量的视频
	// ↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑ 下载条件 ↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑

	// ↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓ 临时数据 ↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓
	Cookies       []*http.Cookie    `json:"-"` // cookie
	authorization string            // 当前使用的jwt
	ArtistUIDMap  map[string]string `json:"-"` // 用户uid
	NowArtist     string            `json:"-"` // 当前下载用户 (用户下载模式使用)
	// ↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑ 临时数据 ↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑
}

// PrintLimit 打印下载条件
func (u *User) PrintLimit() {
	log.Println("当前下载模式:", u.Mode.Name())

	log.Println("当前下载条件:")
	var hasRules bool
	if len(u.Tags) > 0 {
		hasRules = true
		log.Printf("下载标签: %#v", u.Tags)
	}
	if len(u.Artists) > 0 {
		hasRules = true
		log.Printf("下载用户: %#v", u.Artists)
	}
	if len(u.BanTags) > 0 {
		hasRules = true
		log.Printf("ban标签: %#v", u.BanTags)
	}
	if len(u.BanArtists) > 0 {
		hasRules = true
		log.Printf("ban用户: %#v", u.BanArtists)
	}
	if u.LikeLimit > 0 {
		hasRules = true
		log.Printf("下载点赞数达到: %v", u.LikeLimit)
	}

	if !hasRules {
		log.Println("没有设置下载条件,下载所有视频")
	}
}

// GetAuthorization 获取当前使用的jwt
func (u *User) GetAuthorization() string {
	if u.authorization == "" {
		return ""
	}
	return "Bearer " + u.authorization
}

// SetAuthorization 设置当前使用的jwt
func (u *User) SetAuthorization(jwt string) {
	u.authorization = jwt
}

// Check 检查用户信息
func (u *User) Check() error {
	if u.Username == "" || u.Password == "" {
		return ErrEmptyUsernameOrPassword
	}
	return nil
}

// GetLoginBody 获取登录body
func (u *User) GetLoginBody() (string, error) {
	if err := u.Check(); err != nil {
		return "", err
	}
	type LoginBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	newBody := LoginBody{
		Email:    u.Username,
		Password: u.Password,
	}
	data, err := json.Marshal(newBody)
	return string(data), err
}

// SetLoginToken 设置登录token
func (u *User) SetLoginToken(rspBody []byte) error {
	type LoginRsp struct {
		Token string `json:"token"`
	}
	var rsp LoginRsp
	err := json.Unmarshal(rspBody, &rsp)
	if err != nil {
		return err
	}
	u.LoginToken = rsp.Token
	return nil
}

// SetAccessToken 设置访问token
func (u *User) SetAccessToken(rspBody []byte) error {
	type AccessToken struct {
		AccessToken string `json:"accessToken"`
	}

	var accessToken AccessToken
	err := json.Unmarshal(rspBody, &accessToken)
	if err != nil {
		return err
	}
	u.AccessToken = accessToken.AccessToken
	_, err = NewAccessTokenJwt(u.AccessToken)
	if err != nil {
		return err
	}
	return nil
}

// CheckLoginToken 检查登录token是否过期 返回true表示未过期
func (u *User) CheckLoginToken() bool {
	if u.LoginToken == "" {
		return false
	}
	jwtData, err := NewAccessTokenJwt(u.LoginToken)
	if err != nil {
		return false
	}
	return time.Now().Before(jwtData.ExpiresAt.Time)
}

// CheckAccessToken 检查访问token是否过期 返回true表示未过期
func (u *User) CheckAccessToken() bool {
	if u.AccessToken == "" {
		return false
	}
	jwtData, err := NewAccessTokenJwt(u.AccessToken)
	if err != nil {
		return false
	}
	return time.Now().Before(jwtData.ExpiresAt.Time)
}

// CustomClaims 自定义jwt
type CustomClaims struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Premium bool   `json:"premium"`
	jwt.RegisteredClaims
}

// NewAccessTokenJwt 解析jwt不校验
func NewAccessTokenJwt(accessToken string) (*CustomClaims, error) {
	// 分割 JWT
	parts := strings.Split(accessToken, ".")
	if len(parts) < 2 {
		return nil, ErrTokenMalformed
	}

	// 解析载荷部分
	payloadBase64 := parts[1]
	padding := len(payloadBase64) % 4
	if padding > 0 {
		payloadBase64 += strings.Repeat("=", 4-padding)
	}

	payloadBytes, err := base64.StdEncoding.DecodeString(payloadBase64)
	if err != nil {
		return nil, err
	}

	var claims CustomClaims
	err = json.Unmarshal(payloadBytes, &claims)
	if err != nil {
		return nil, err
	}
	return &claims, nil
}
