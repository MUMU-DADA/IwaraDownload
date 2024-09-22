package model

import (
	"encoding/base64"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
	"time"
)

// User 用户信息
type User struct {
	Username      string         `json:"username"`    // 用户名
	Password      string         `json:"password"`    // 密码
	LoginToken    string         `json:"loginToken"`  // 登录token
	AccessToken   string         `json:"accessToken"` // 访问token
	Cookies       []*http.Cookie `json:"-"`           // cookie
	authorization string         // 当前使用的jwt
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
