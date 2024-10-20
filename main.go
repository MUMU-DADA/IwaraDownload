package main

import (
	"IwaraDownload/consts"
	"IwaraDownload/internal/request"
	"IwaraDownload/model"
	"IwaraDownload/pkg/config"
	"IwaraDownload/pkg/date"
	"IwaraDownload/pkg/files"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/araddon/dateparse"
)

var (
	maxPage int = 50 // 初始最大页数
)

// saveVideoDatabase 保存视频数据到本地数据库
func saveVideoDatabase(filePath string, videoData model.Result, fileData []*model.Video) {
	dataFileName := filePath + string(os.PathSeparator) + consts.VIDEO_DATABASE

	var db model.Data
	// 检查文件是否存在
	if files.CheckFileExists(dataFileName) {
		// 读取文件
		data, err := files.ReadFile(dataFileName)
		if err != nil {
			log.Println("读取数据库文件失败:", err)
			return
		}
		// 解析文件
		err = json.Unmarshal(data, &db)
		if err != nil {
			log.Println("解析数据库文件失败:", err)
			return
		}
		// 添加新数据
		if db.VideoMap == nil {
			db.VideoMap = make(map[string]model.VideoData)
		}
	} else {
		db = model.Data{
			VideoMap: make(map[string]model.VideoData),
		}
	}

	addData := model.VideoData{
		Video: &videoData,
		Files: fileData,
	}
	oldData, ok := db.VideoMap[videoData.ID]
	if fileData == nil && ok {
		// 如果没有文件数据,则不更新文件数据
		addData.Files = oldData.Files
	}
	db.VideoMap[videoData.ID] = addData

	// 保存数据库
	data, err := json.Marshal(db)
	if err != nil {
		log.Println("序列化数据库文件失败:", err)
		return
	}
	err = files.WriteFile(dataFileName, data)
	if err != nil {
		log.Println("写入数据库文件失败:", err)
		return
	}
}

// skipVideo 依据配置检查是否跳过视频
func skipVideo(user *model.User, video model.Result) bool {
	var hasRules bool

	// 检查点赞 (点赞是最顶级的优先度,如果设置了但是视频没有达到,那么不检查tag或者作者直接跳过)
	if user.LikeLimit > 0 && video.NumLikes < user.LikeLimit {
		// 如果点赞数超过配置,则跳过当前视频
		log.Printf("视频点赞数: %d,小于配置: %d,跳过当前视频\n", video.NumLikes, user.LikeLimit)
		return true
	}

	// 优先匹配指定的条件不进行不跳过
	// 1. 指定标签
	if len(user.Tags) > 0 {
		hasRules = true
		tempVideoMap := make(map[string]bool)
		for _, tag := range user.Tags {
			tempVideoMap[tag] = true
		}
		for _, tag := range video.Tags {
			if tempVideoMap[tag.ID] {
				// 如果标签在配置中,则跳过当前视频
				log.Printf("视频标签: %s 在配置中,下载当前视频\n", tag.ID)
				return false
			}
		}
	}
	// 2. 指定作者
	if len(user.Artists) > 0 {
		hasRules = true
		for _, artist := range user.Artists {
			if video.User.Username == artist {
				// 如果作者在配置中,则跳过当前视频
				log.Printf("视频作者: %s 在配置中,下载当前视频\n", artist)
				return false
			}
		}
	}

	// 再检查是否符合禁止条件进行跳过
	// 1. 禁止标签
	if len(user.BanTags) > 0 {
		tempVideoMap := make(map[string]bool)
		for _, tag := range user.BanTags {
			tempVideoMap[tag] = true
		}
		for _, tag := range video.Tags {
			if tempVideoMap[tag.ID] {
				// 如果标签在禁止列表中,则跳过当前视频
				log.Printf("视频标签: %s 在禁止列表中,跳过当前视频\n", tag.ID)
				return true
			}
		}
	}
	// 2. 禁止作者
	if len(user.BanArtists) > 0 {
		tempVideoMap := make(map[string]bool)
		for _, artist := range user.BanArtists {
			tempVideoMap[artist] = true
		}
		if tempVideoMap[video.User.Username] {
			// 如果作者在禁止列表中,则跳过当前视频
			log.Printf("视频作者: %s 在禁止列表中,跳过当前视频\n", video.User.Username)
			return true
		}
	}

	// 如果一个指定条件都没有配置,则最后默认放行,如果配置了,则最后默认禁止
	return hasRules
}

// rangePage 遍历页码
func rangePage(user *model.User, rangeFunc func(pageNum int, videoData model.Result) (Break bool, page int, err error)) error {
	for i := 0; i <= maxPage; i++ {
		log.Printf("正在获取第%d页视频列表\n", i)
		// 尝试获取视频列表
		pageData, err := request.GetVideoData(user, i)
		if err != nil {
			// 视频页数据获取的失败是不能容忍的，直接返回错误
			log.Printf("获取视频列表失败: %s\n", err.Error())
			return err
		}
		log.Println("视频列表获取成功")

		maxPage = pageData.Count / pageData.Limit
		log.Printf("总计有%d页", maxPage)

		for _, video := range pageData.Results {
			Break, page, err := rangeFunc(i, video)
			if err != nil {
				return err
			}
			// 再设置当前页码
			i = page
			if Break {
				return nil
			}
		}
	}
	return nil
}

// Month 开始月下载任务
func Month(user *model.User, year int, month int, lastDownloadTime time.Time) error {
	log.Println("开始下载", year, "年", month, "月视频")

	filePath := consts.FlagConf.WorkDIr + string(os.PathSeparator) + strconv.Itoa(year) + string(os.PathSeparator) + strconv.Itoa(month)
	err := files.CheckDirOrCreate(filePath)
	if err != nil {
		return err
	}
	savePath := filePath
	if !consts.RUN_IN_WINDOWS {
		savePath = consts.UNIX_SAVE_PATH
	}
	err = files.CheckDirOrCreate(savePath)
	if err != nil {
		return err
	}

	// 获取目标月份的第一天
	startTime := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	// 获取目标月份的最后一天
	lastDayOfMonth := date.GetLastDayOfMonth(startTime)

	var downloadCount int
	var fullCount int

	var videoDownload bool
	for i := 0; i <= maxPage; i++ {
		log.Printf("正在获取第%d页视频列表\n", i)
		// 尝试获取视频列表
		pageData, err := request.GetVideoData(user, i)
		if err != nil {
			// 视频页数据获取的失败是不能容忍的，直接返回错误
			log.Printf("获取视频列表失败: %s\n", err.Error())
			return err
		}
		log.Println("视频列表获取成功")

		maxPage = pageData.Count / pageData.Limit // 依据分页数据重新设置最大页码
		log.Printf("总计有%d页", maxPage)

		for _, video := range pageData.Results {
			log.Println("处理视频:", video.Title)
			createTime, err := dateparse.ParseLocal(video.CreatedAt)
			if err != nil {
				log.Printf("解析时间失败: %s\n", err.Error())
				// 跳过当前视频
				continue
			}
			// 因为获取到的时间是标准时,但是转换库会将其转换为本地时区,所以需要重新修改回UTC后再次转换为本地时区
			createTime = time.Date(createTime.Year(), createTime.Month(), createTime.Day(), createTime.Hour(), createTime.Minute(), createTime.Second(), createTime.Nanosecond(), time.UTC)
			createTime = createTime.Local()
			log.Println("视频创建时间:", createTime)

			// 跳过不需要下载日期范围的视频
			createYear := createTime.Year()
			createMonth := int(createTime.Month())
			if (createYear != year || createMonth != month) && lastDayOfMonth.Before(createTime) {
				// 当前视频不在目标月份，跳过
				// 此处分支跳过意味着当前页面的视频位于目标下载时间段的后面月份
				log.Println("视频不在目标月份,跳过")

				// 计算 t1 到目标月份最后一天的最后一秒的差值
				duration := createTime.Sub(lastDayOfMonth)
				// 将差值转换为天数
				days := int(duration.Hours() / 24)

				if user.Mode == model.SubscribeMode {
					// 订阅下载模式的分页视频数量明显较少,为了放置跳转过头,使用较小的跳页 (每超过3天/1页)
					if days > 3 {
						jumpPage := (days / 3) * 1
						i += jumpPage
						log.Println("当前分页视频时间远远早于目标扫描日期范围, 跳", jumpPage, "页")
						break
					}
				}
				if days > 4 {
					// 如果当前分页的扫描视频时间远远晚于目标扫描日期范围,则直接大范围跳页 (每超过2天/3页)
					jumpPage := (days / 2) * 3
					i += jumpPage
					log.Println("当前分页视频时间远远晚于目标扫描日期范围, 跳", jumpPage, "页")
					break
				}

				continue
			}

			if createTime.Before(lastDownloadTime) {
				// 当前视频创建时间早于上次开始下载时间，判断为下载任务完成
				log.Println("视频创建时间", createTime, "早于上次开始下载时间", lastDownloadTime, ",判断为下载任务完成")
				log.Println("视频下载任务完成")
				return nil
			}
			if createTime.Before(startTime) {
				// 当前视频创建时间早于开始时间，判断为下载任务完成
				log.Println("视频下载任务完成")
				return nil
			}
			log.Println("视频符合时间范围,继续...")

			// 检查是否需要跳过当前视频
			if skipVideo(user, video) {
				log.Println("视频不符合下载条件,跳过...")
				continue
			}
			fullCount++

			// 检查文件是否已经被下载,如果被下载则跳过
			// 尝试从目标路径中获取可能的文件名
			log.Printf("正在检查文件是否已下载, 作者: %s, 名称: %s", video.User.Username, video.Title)
			checkName := files.SanitizeFileName(fmt.Sprintf("[%s] %s", video.User.Username, video.Title))
			checkDir := savePath + string(os.PathSeparator)
			f := files.CheckVideoFileExist(checkName, checkDir)
			if f != "" {
				log.Printf("视频已存在: %s 跳过...\n", f)
				// 保存视频数据到数据库
				saveVideoDatabase(savePath, video, nil)
				// 尝试软链接
				if !consts.RUN_IN_WINDOWS {
					videoPath := savePath + string(os.PathSeparator) + f
					files.TryFileLink(videoPath, filePath+string(os.PathSeparator)+f)
				}
				continue
			}
			// 检查时会额外检查昵称(可更改的用户名),兼容旧版本下载数据(因为旧版本下载数据使用的是文件名是昵称,为了防止重复下载,特意添加的屎山容错)
			log.Printf("(检查昵称)正在检查文件是否已下载, 作者: %s, 名称: %s", video.User.Name, video.Title)
			checkName = files.SanitizeFileName(fmt.Sprintf("[%s] %s", video.User.Name, video.Title))
			checkDir = savePath + string(os.PathSeparator)
			f = files.CheckVideoFileExist(checkName, checkDir)
			if f != "" {
				log.Printf("(检查昵称)视频已存在: %s 跳过...\n", f)
				// 保存视频数据到数据库
				saveVideoDatabase(savePath, video, nil)
				// 尝试软链接
				if !consts.RUN_IN_WINDOWS {
					videoPath := savePath + string(os.PathSeparator) + f
					files.TryFileLink(videoPath, filePath+string(os.PathSeparator)+f)
				}
				continue
			}
			log.Println("文件不存在,准备获取视频下载地址")

			videoDownload = true

			// 开始下载视频
			videoUrl, err := request.GetVideoDownloadUrl(user, video)
			if err != nil {
				log.Printf("获取视频地址失败: %s\n", err.Error())
				// 跳过当前视频
				continue
			}
			log.Printf("视频地址: %s\n", videoUrl[0].Src.Download)

			// 保存视频数据到数据库
			saveVideoDatabase(savePath, video, videoUrl)

			videoName := files.SanitizeFileName(fmt.Sprintf("[%s] %s [%s].mp4", video.User.Username, video.Title, videoUrl[0].Name))
			videoPath := savePath + string(os.PathSeparator) + videoName
			startDownloadTime := time.Now()
			log.Printf("开始下载视频: %s 分辨率: %s\n", videoPath, videoUrl[0].Name)
			err = request.Download(user, videoUrl[0].Src.Download, videoPath)
			if err != nil {
				log.Printf("下载视频失败: %s %s\n", videoName, err.Error())
				// 跳过当前视频
				continue
			}
			log.Println("视频下载完成, 耗时:", time.Since(startDownloadTime))
			downloadCount++

			// 尝试软链接
			if !consts.RUN_IN_WINDOWS {
				files.TryFileLink(videoPath, filePath+string(os.PathSeparator)+videoName)
			}
		}
		request.DelaySwitch = videoDownload
	}

	log.Println("视频下载任务完成")
	log.Println("本轮扫描一共需要下载", fullCount, "个视频,本次下载", downloadCount)
	return nil
}

// Hot 下载热门视频
func Hot(user *model.User, pageLimit int) error {
	filePath := consts.FlagConf.WorkDIr + string(os.PathSeparator) + consts.HOT_DIR
	err := files.CheckDirOrCreate(filePath)
	if err != nil {
		return err
	}
	savePath := filePath
	if !consts.RUN_IN_WINDOWS {
		savePath = consts.UNIX_SAVE_PATH
	}
	err = files.CheckDirOrCreate(savePath)
	if err != nil {
		return err
	}

	var downloadCount int
	var fullCount int
	rangeErr := rangePage(user, func(pageNum int, video model.Result) (Break bool, page int, err error) {
		if pageNum > pageLimit {
			log.Println("热门视频下载任务完成")
			return true, pageNum, nil
		}
		log.Println("处理视频:", video.Title)
		// 检查是否需要跳过当前视频
		if skipVideo(user, video) {
			log.Println("视频不符合下载条件,跳过...")
			return false, pageNum, nil
		}
		fullCount++
		// 检查文件是否已经被下载,如果被下载则跳过
		// 尝试从目标路径中获取可能的文件名
		log.Printf("正在检查文件是否已下载, 作者: %s, 名称: %s", video.User.Username, video.Title)
		checkName := files.SanitizeFileName(fmt.Sprintf("[%s] %s", video.User.Username, video.Title))
		checkDir := savePath + string(os.PathSeparator)
		f := files.CheckVideoFileExist(checkName, checkDir)
		if f != "" {
			log.Printf("视频已存在: %s 跳过...\n", f)
			// 保存视频数据到数据库
			saveVideoDatabase(savePath, video, nil)
			// 尝试软链接
			if !consts.RUN_IN_WINDOWS {
				videoPath := savePath + string(os.PathSeparator) + f
				files.TryFileLink(videoPath, filePath+string(os.PathSeparator)+f)
			}
			return false, pageNum, nil
		}
		log.Println("文件不存在,准备获取视频下载地址")

		// 开始下载视频
		videoUrl, err := request.GetVideoDownloadUrl(user, video)
		if err != nil {
			log.Printf("获取视频地址失败: %s\n", err.Error())
			// 跳过当前视频
			return false, pageNum, nil
		}
		log.Printf("视频地址: %s\n", videoUrl[0].Src.Download)

		// 保存视频数据到数据库
		saveVideoDatabase(savePath, video, videoUrl)

		videoName := files.SanitizeFileName(fmt.Sprintf("[%s] %s [%s].mp4", video.User.Username, video.Title, videoUrl[0].Name))
		videoPath := savePath + string(os.PathSeparator) + videoName
		startDownloadTime := time.Now()
		log.Printf("开始下载视频: %s 分辨率: %s\n", videoPath, videoUrl[0].Name)
		err = request.Download(user, videoUrl[0].Src.Download, videoPath)
		if err != nil {
			log.Printf("下载视频失败: %s %s\n", videoName, err.Error())
			// 跳过当前视频
			return false, pageNum, nil
		}
		log.Println("视频下载完成, 耗时:", time.Since(startDownloadTime))
		downloadCount++

		// 尝试软链接
		if !consts.RUN_IN_WINDOWS {
			files.TryFileLink(videoPath, filePath+string(os.PathSeparator)+videoName)
		}

		return false, pageNum, nil
	})

	log.Println("本轮扫描一共需要下载", fullCount, "个视频,本次下载", downloadCount)
	return rangeErr
}

func Artist(user *model.User) error {
	filePath := consts.FlagConf.WorkDIr + string(os.PathSeparator) + consts.ARTIST_DIR + string(os.PathSeparator) + user.NowArtist
	err := files.CheckDirOrCreate(filePath)
	if err != nil {
		return err
	}
	savePath := filePath
	if !consts.RUN_IN_WINDOWS {
		savePath = consts.UNIX_SAVE_PATH
	}
	err = files.CheckDirOrCreate(savePath)
	if err != nil {
		return err
	}

	var downloadCount int
	var fullCount int
	rangeErr := rangePage(user, func(pageNum int, video model.Result) (Break bool, page int, err error) {
		log.Println("处理视频:", video.Title)
		// 检查是否需要跳过当前视频
		if skipVideo(user, video) {
			log.Println("视频不符合下载条件,跳过...")
			return false, pageNum, nil
		}
		fullCount++
		// 检查文件是否已经被下载,如果被下载则跳过
		// 尝试从目标路径中获取可能的文件名
		log.Printf("正在检查文件是否已下载, 作者: %s, 名称: %s", video.User.Username, video.Title)
		checkName := files.SanitizeFileName(fmt.Sprintf("[%s] %s", video.User.Username, video.Title))
		checkDir := savePath + string(os.PathSeparator)
		f := files.CheckVideoFileExist(checkName, checkDir)
		if f != "" {
			log.Printf("视频已存在: %s 跳过...\n", f)
			// 保存视频数据到数据库
			saveVideoDatabase(savePath, video, nil)
			// 尝试软链接
			if !consts.RUN_IN_WINDOWS {
				videoPath := savePath + string(os.PathSeparator) + f
				files.TryFileLink(videoPath, filePath+string(os.PathSeparator)+f)
			}
			return false, pageNum, nil
		}
		log.Println("文件不存在,准备获取视频下载地址")

		// 开始下载视频
		videoUrl, err := request.GetVideoDownloadUrl(user, video)
		if err != nil {
			log.Printf("获取视频地址失败: %s\n", err.Error())
			// 跳过当前视频
			return false, pageNum, nil
		}
		log.Printf("视频地址: %s\n", videoUrl[0].Src.Download)

		// 保存视频数据到数据库
		saveVideoDatabase(savePath, video, videoUrl)

		videoName := files.SanitizeFileName(fmt.Sprintf("[%s] %s [%s].mp4", video.User.Username, video.Title, videoUrl[0].Name))
		videoPath := savePath + string(os.PathSeparator) + videoName
		startDownloadTime := time.Now()
		log.Printf("开始下载视频: %s 分辨率: %s\n", videoPath, videoUrl[0].Name)
		err = request.Download(user, videoUrl[0].Src.Download, videoPath)
		if err != nil {
			log.Printf("下载视频失败: %s %s\n", videoName, err.Error())
			// 跳过当前视频
			return false, pageNum, nil
		}
		log.Println("视频下载完成, 耗时:", time.Since(startDownloadTime))
		downloadCount++

		// 尝试软链接
		if !consts.RUN_IN_WINDOWS {
			files.TryFileLink(videoPath, filePath+string(os.PathSeparator)+videoName)
		}

		return false, pageNum, nil
	})

	log.Println("本轮扫描一共需要下载", fullCount, "个视频,本次下载", downloadCount)
	return rangeErr
}

var loopLastScanTimeMap = map[model.DownloadMode]time.Time{} // 上次扫描时间

func loop() {
	log.Println("开始循环扫描任务")
	config.Config.PrintLimit()

	loopLastScanTime := loopLastScanTimeMap[config.Config.Mode]
	if loopLastScanTime.IsZero() {
		loopLastScanTime = time.Now()
	}

	retryTimes := 0
	for {
		start := time.Now()
		log.Println("开始扫描任务")
		// 获取当前年月
		year := time.Now().Year()
		month := time.Now().Month()
		if err := Month(config.Config, year, int(month), loopLastScanTime); err != nil {
			if retryTimes > consts.MAX_RETRY_TIMES {
				log.Println("重试次数过多,程序退出")
				os.Exit(1)
			}
			log.Println("扫描任务失败", err, "开始重试")
			retryTimes++
			continue
		}
		log.Println("扫描任务完成")
		useTime := time.Since(start)
		log.Println("本次扫描任务耗时:", useTime)
		loopLastScanTime = start
		log.Println("===============================================================")
		retryTimes = 0

		if useTime < consts.SCAN_STEP {
			time.Sleep(consts.SCAN_STEP - useTime)
		}

		// 检测是否开启多模式下载的话每个模式只执行一次循环任务
		if len(config.Config.MultiMode) > 0 {
			loopLastScanTimeMap[config.Config.Mode] = loopLastScanTime
			break
		}
	}
}

func hot() {
	log.Println("开始下载热门视频")
	config.Config.PrintLimit()

	retryTimes := 0
	for {
		start := time.Now()
		log.Println("开始下载热门视频")
		if err := Hot(config.Config, 1); err != nil {
			if retryTimes > consts.MAX_RETRY_TIMES {
				log.Println("重试次数过多,程序退出")
				os.Exit(1)
			}
			log.Println("下载热门视频任务失败", err, "开始重试")
			retryTimes++
			continue
		}
		log.Println("下载热门视频任务完成")
		useTime := time.Since(start)
		log.Println("本次扫描任务耗时:", useTime)
		log.Println("===============================================================")
		retryTimes = 0

		if useTime < consts.SCAN_STEP {
			time.Sleep(consts.SCAN_STEP - useTime)
		}

		// 检测是否开启多模式下载的话每个模式只执行一次循环任务
		if len(config.Config.MultiMode) > 0 {
			break
		}
	}
}

func once() {
	log.Println("开始单次扫描任务")
	config.Config.PrintLimit()

	start := time.Now()
	mounth := consts.FlagConf.Month
	year := consts.FlagConf.Year
	if mounth == 0 {
		mounth = int(time.Now().Month())
	}
	if year == 0 {
		year = time.Now().Year()
	}
	if err := Month(config.Config, year, mounth, time.Time{}); err != nil {
		log.Println("下载任务失败", err)
	}
	log.Println("单次扫描任务完成")
	useTime := time.Since(start)
	log.Println("本次扫描任务耗时:", useTime)
}

func artistLoop() {
	log.Println("开始循环扫描任务")
	config.Config.PrintLimit()

	// 获取用户的ID信息
	var artistUidMap = make(map[string]string)
	for _, v := range config.Config.DownloadArtists {
		info, err := request.GetArtistInfo(config.Config, v)
		if err != nil {
			log.Println("获取用户信息失败:", err)
			continue
		}
		artistUidMap[v] = info.ID
	}
	config.Config.DownloadArtists = []string{}
	for k, v := range artistUidMap {
		log.Println("获取用户信息成功: 用户:", k, "ID:", v)
		config.Config.DownloadArtists = append(config.Config.DownloadArtists, k)
	}
	config.Config.ArtistUIDMap = artistUidMap
	log.Println("获取用户ID信息完成", "下载用户", config.Config.DownloadArtists)

	// 开始循环下载作者的任务
	retryTimes := 0
	for {
		start := time.Now()
		log.Println("开始下载指定作者视频")
		for _, v := range config.Config.DownloadArtists {
			config.Config.NowArtist = v
			log.Println("当前下载作者:", v)
			config.Config.NowArtist = v
			if err := Artist(config.Config); err != nil {
				if retryTimes > consts.MAX_RETRY_TIMES {
					log.Println("重试次数过多,程序退出")
					os.Exit(1)
				}
				log.Println("下载指定作者视频任务失败", err, "开始重试")
				retryTimes++
				continue
			}
			log.Println("下载指定作者", config.Config.NowArtist, "视频任务完成")
		}
		useTime := time.Since(start)
		log.Println("本次扫描任务耗时:", useTime)
		log.Println("===============================================================")
		retryTimes = 0

		if useTime < consts.SCAN_STEP {
			time.Sleep(consts.SCAN_STEP - useTime)
		}

		// 检测是否开启多模式下载的话每个模式只执行一次循环任务
		if len(config.Config.MultiMode) > 0 {
			break
		}
	}
}

func multiMode() {
	for {
		for _, v := range config.Config.MultiMode {
			config.Config.Mode = v
			switch v {
			case model.SubscribeMode:
				loop()
			case model.HotMode:
				hot()
			case model.ArtistMode:
				artistLoop()
			default:
				log.Println("不支持的模式", v)
				continue
			}
		}
	}
}

// 流程为: 获取cookie -> 登录 -> 获取token -> 获取视频列表 -> 视频主页 -> 获取视频地址 -> 下载视频
func main() {
	if consts.FlagConf.Year != 0 || consts.FlagConf.Month != 0 {
		log.Println("指定了年份或月份,开始下载指定月份视频")
		once()
	}

	if len(config.Config.MultiMode) > 0 {
		log.Println("多模式:", config.Config.MultiMode)
		multiMode()
		return
	}

	switch config.Config.Mode {
	case model.AllMode:
		log.Println("未指定年份或月份,进行从当月开始的挂机下载任务")
		loop()
	case model.SubscribeMode:
		log.Println("未指定年份或月份,进行从当月开始的挂机下载任务")
		loop()
	case model.HotMode:
		log.Println("指定了热门视频模式,开始下载热门视频")
		hot()
	case model.ArtistMode:
		log.Println("指定了艺术家模式,开始下载指定艺术家的视频")
		artistLoop()
	default:
		log.Fatalln("不支持的模式", config.Config.Mode)
	}
}
