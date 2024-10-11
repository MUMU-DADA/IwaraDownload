package request

import (
	"IwaraDownload/model"
	"IwaraDownload/pkg/config"
	"IwaraDownload/pkg/utils"
	"encoding/json"
	"fmt"
	"log"
	"sort"
)

const (
	apiHost         = "https://api.iwara.tv"                          // api地址
	apiLoginUrl     = apiHost + "/user/login"                         // 登录地址
	apiTokenUrl     = apiHost + "/user/token"                         // 获取token地址
	apiPageUrl      = apiHost + "/videos?rating=all&limit=32&page=%d" // 视频列表地址
	apiVideoMainUrl = apiHost + "/video/%s"                           // 视频主页地址
)

// 流程为: 登录 -> 获取token -> 获取视频列表 -> 视频主页 -> 获取视频地址 -> 下载视频

// Login 登录
func Login(user *model.User) error {
	log.Println("正在登录", user.Username)
	bodyStr, err := user.GetLoginBody()
	if err != nil {
		return err
	}

	body, err := getWeb(apiLoginUrl, POST, user, bodyStr, nil)
	if err != nil {
		return err
	}

	if len(body) == 0 {
		return model.ErrLoginFailed
	}

	// a := string(body)
	// log.Println(a)

	err = user.SetLoginToken(body)
	if err != nil {
		return err
	}

	// 更新好了登录token,保存一下配置文件
	err = config.SaveConfig(user)
	if err != nil {
		log.Println("配置文件保存失败", err)
	}
	log.Println("登录成功", user.Username)
	return nil
}

// GetToken 获取token
func GetToken(user *model.User) error {
	log.Println("正在获取token", user.Username)
	if !user.CheckLoginToken() {
		log.Println("登录token已过期,重新登录")
		err := Login(user)
		if err != nil {
			return err
		}
	}

	user.SetAuthorization(user.LoginToken)
	body, err := getWeb(apiTokenUrl, POST, user, "", nil)
	if err != nil {
		return err
	}

	// a := string(body)
	// log.Println(a)

	err = user.SetAccessToken(body)
	if err != nil {
		return err
	}
	user.SetAuthorization(user.AccessToken)

	// 更新好了访问token,保存一下配置文件
	err = config.SaveConfig(user)
	if err != nil {
		log.Println("配置文件保存失败", err)
	}
	log.Println("获取token成功", user.Username)
	return nil
}

// RefreshAccessToken 刷新token
func RefreshAccessToken(user *model.User) error {
	defer user.SetAuthorization(user.AccessToken)
	if user.CheckAccessToken() {
		return nil
	}
	log.Println("访问token已过期,重新获取")
	return GetToken(user)
}

// GetVideoData 获取视频地址
func GetVideoData(user *model.User, page int) (*model.PageDataRoot, error) {
	err := RefreshAccessToken(user)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(apiPageUrl, page)
	if user.Subscribe {
		// 获取订阅的视频
		url = url + "&subscribed=true"
	} else if user.Hot {
		// 获取热门视频
		url = url + "&sort=hot"
	} else {
		// 默认全部下载模式,依据时间排序
		url = url + "&sort=date"
	}
	body, err := getWeb(url, GET, user, "", nil)
	if err != nil {
		return nil, err
	}

	// a := string(body)
	// log.Println(a)

	var rsp model.PageDataRoot
	err = json.Unmarshal(body, &rsp)
	if err != nil {
		return nil, err
	}
	return &rsp, err
}

// GetVideoDownloadUrl 获取视频下载地址
func GetVideoDownloadUrl(user *model.User, videoData model.Result) ([]*model.Video, error) {
	err := RefreshAccessToken(user)
	if err != nil {
		return nil, err
	}

	videoMainUrl := fmt.Sprintf(apiVideoMainUrl, videoData.ID)
	body, err := getWeb(videoMainUrl, GET, user, "", nil)
	if err != nil {
		return nil, err
	}

	// a := string(body)
	// log.Println(a)

	var rsp model.Result
	err = json.Unmarshal(body, &rsp)
	if err != nil {
		return nil, err
	}

	fileUrl := rsp.FileUrl
	if fileUrl == "" {
		return nil, model.ErrNoVideoUrl
	}

	// originName := "Iwara - " + videoData.Title + " [" + videoData.ID + "].mp4"
	// lastUrl := fileUrl + "&download=" + url.QueryEscape(originName)

	// // 先尝试发一个options
	// getWeb(fileUrl, OPTIONS, user, "")

	xVersion, err := utils.GenXVersion(fileUrl)
	if err != nil {
		return nil, err
	}

	// 再进行一次get
	body, err = getWeb(fileUrl, GET, user, "", map[string]string{"X-Version": xVersion})
	if err != nil {
		return nil, err
	}

	var videoSrc []*model.Video
	err = json.Unmarshal(body, &videoSrc)
	if err != nil {
		return nil, err
	}

	if len(videoSrc) < 1 {
		// 跳过当前视频
		return nil, fmt.Errorf("获取视频地址为空: %s\n", videoData.ID)
	}

	// 排序默认下载最清晰的视频
	sort.Slice(videoSrc, func(i, j int) bool {
		return model.VideoDefinitionMap[videoSrc[i].Name] > model.VideoDefinitionMap[videoSrc[j].Name]
	})

	return videoSrc, nil
}

// Download 下载视频
func Download(user *model.User, videoUrl string, filePath string) error {
	downloadUrl := "https:" + videoUrl
	err := saveWebRspToFile(filePath, downloadUrl, GET, user, "")
	return err
}
