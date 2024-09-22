package model

type E struct {
	Code int
	Msg  string
}

// Error 实现error接口
func (e E) Error() string {
	return e.Msg
}

var (
	ErrSuccess                 = E{0, "成功"}
	ErrNoLogin                 = E{1, "未登录"}
	ErrLoginFailed             = E{2, "登录失败"}
	ErrNoVideo                 = E{3, "没有视频"}
	ErrNoVideoUrl              = E{4, "没有视频地址"}
	ErrEmptyUsernameOrPassword = E{5, "用户名或密码为空"}
	ErrTokenMalformed          = E{6, "token格式错误"}
)
