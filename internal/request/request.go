package request

import (
	"IwaraDownload/consts"
	"IwaraDownload/model"
	"IwaraDownload/pkg/config"
	"IwaraDownload/pkg/files"
	"IwaraDownload/pkg/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"
	"time"
)

const (
	apiHost             = "https://api.iwara.tv"  // api地址
	apiLoginUrl         = apiHost + "/user/login" // 登录地址
	apiTokenUrl         = apiHost + "/user/token" // 获取token地址
	apiPageUrl          = apiHost + "/videos"     // 视频列表地址
	apiVideoMainUrl     = apiHost + "/video/%s"   // 视频主页地址
	apiArtistProfileUrl = apiHost + "/profile/%s" // 用户主页地址
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

	u, err := url.Parse(apiPageUrl)
	if err != nil {
		return nil, err
	}
	values := u.Query()
	values.Add("rating", "all")

	pageNum := consts.PAGE_NUM_DEFAULT
	switch user.Mode {
	case model.AllMode:
		// 默认全部下载模式,依据时间排序
		values.Add("sort", "date")
	case model.SubscribeMode:
		// 获取订阅的视频
		values.Add("sort", "date")
		values.Add("subscribed", "true")
	case model.HotMode:
		// 获取热门视频
		pageNum = consts.PAGE_NUM_HOT
		values.Add("sort", "hot")
	case model.ArtistMode:
		// 获取指定用户的视频
		values.Add("sort", "date")
		values.Add("user", user.ArtistUIDMap[user.NowArtist])
	default:
		log.Fatalln("不支持的模式", user.Mode)
	}
	values.Add("limit", fmt.Sprintf("%d", pageNum))
	values.Add("page", fmt.Sprintf("%d", page))
	u.RawQuery = values.Encode()

	body, err := getWeb(u.String(), GET, user, "", nil)
	if err != nil {
		return nil, err
	}

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

	var rsp model.Result
	err = json.Unmarshal(body, &rsp)
	if err != nil {
		return nil, err
	}

	fileUrl := rsp.FileUrl
	if fileUrl == "" {
		return nil, model.ErrNoVideoUrl
	}

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

// QuickCheckVideoExist 快速检查视频是否存在
func QuickCheckVideoExist(video model.Result, savePath, filePath string, urlName string) bool {
	// 快速检查检查Source
	if urlName == "" {
		urlName = "Source"
	}
	videoName := files.SanitizeFileName(fmt.Sprintf("[%s] %s [%s].mp4", video.User.Username, video.Title, urlName))
	videoPath := savePath + string(os.PathSeparator) + videoName
	log.Printf("检查视频是否存在: %s 分辨率: %s\n", videoPath, urlName)

	// 检查文件是否已经下载了
	if !consts.RUN_IN_WINDOWS {
		var existA, existB bool
		// 检查download目录是否存在
		if files.CheckFileExists(videoPath) {
			existA = true
		}
		// 检查保存目录是否存在
		if files.CheckFileExists(filePath + string(os.PathSeparator) + videoName) {
			existB = true
		}

		if (existA || existB) && (existA && existB) {
			// 有目录缺失则把文件复制到另一个目录
			if existA {
				// 将download目录文件复制到保存目录
				files.TryFileLink(videoPath, filePath+string(os.PathSeparator)+videoName)
			} else {
				// 将保存目录文件复制到download目录
				files.TryFileLink(filePath+string(os.PathSeparator)+videoName, videoPath)
			}
		}
		if existA || existB {
			log.Printf("视频已存在: %s 跳过...\n", filePath)
			return true
		}
	} else {
		var existA bool
		// 检查download目录是否存在
		if files.CheckFileExists(videoPath) {
			existA = true
		}
		if existA {
			log.Printf("视频已存在: %s 跳过...\n", videoPath)
			return true
		}
	}
	return false
}

// DownloadAndSaveVideo 下载视频并保存到指定路径
func DownloadAndSaveVideo(user *model.User, video model.Result, savePath, filePath string, videoUrl []*model.Video) error {
	videoName := files.SanitizeFileName(fmt.Sprintf("[%s] %s [%s].mp4", video.User.Username, video.Title, videoUrl[0].Name))
	videoPath := savePath + string(os.PathSeparator) + videoName
	startDownloadTime := time.Now()
	log.Printf("开始下载视频: %s 分辨率: %s\n", videoPath, videoUrl[0].Name)

	// 检查文件是否已经下载了
	if !consts.RUN_IN_WINDOWS {
		var existA, existB bool
		// 检查download目录是否存在
		if files.CheckFileExists(videoPath) {
			existA = true
		}
		// 检查保存目录是否存在
		if files.CheckFileExists(filePath + string(os.PathSeparator) + videoName) {
			existB = true
		}

		if (existA || existB) && (existA && existB) {
			// 有目录缺失则把文件复制到另一个目录
			if existA {
				// 将download目录文件复制到保存目录
				files.TryFileLink(videoPath, filePath+string(os.PathSeparator)+videoName)
			} else {
				// 将保存目录文件复制到download目录
				files.TryFileLink(filePath+string(os.PathSeparator)+videoName, videoPath)
			}
		}
		if existA || existB {
			log.Printf("视频已存在: %s 跳过...\n", filePath)
			return nil
		}
	} else {
		var existA bool
		// 检查download目录是否存在
		if files.CheckFileExists(videoPath) {
			existA = true
		}
		if existA {
			log.Printf("视频已存在: %s 跳过...\n", videoPath)
			return nil
		}
	}

	err := Download(user, videoUrl[0].Src.Download, videoPath)
	if err != nil {
		log.Printf("下载视频失败: %s %s\n", videoName, err.Error())
		// 跳过当前视频
		return err
	}
	log.Println("视频下载完成, 耗时:", time.Since(startDownloadTime))

	// 尝试软链接
	if !consts.RUN_IN_WINDOWS {
		files.TryFileLink(videoPath, filePath+string(os.PathSeparator)+videoName)
	}
	return nil
}

// Download 下载视频
func Download(user *model.User, videoUrl string, filePath string) error {
	downloadUrl := "https:" + videoUrl
	err := saveWebRspToFile(filePath, downloadUrl, GET, user, "")
	return err
}

// GetArtistInfo 获取用户信息
func GetArtistInfo(user *model.User, artistName string) (*model.Artist, error) {
	err := RefreshAccessToken(user)
	if err != nil {
		return nil, err
	}

	urlRaw := fmt.Sprintf(apiArtistProfileUrl, artistName)
	body, err := getWeb(urlRaw, GET, user, "", nil)
	if err != nil {
		return nil, err
	}

	var rsp model.ArtistProfile
	err = json.Unmarshal(body, &rsp)
	return &rsp.User, err
}
