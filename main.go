package main

import (
	"IwaraDownload/consts"
	"IwaraDownload/internal/request"
	"IwaraDownload/model"
	"IwaraDownload/pkg/config"
	"IwaraDownload/pkg/date"
	"IwaraDownload/pkg/files"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/araddon/dateparse"
)

var (
	maxPage int = 50 // 最大页数
)

// Month 开始月下载任务
func Month(user *model.User, year int, month int, lastDownloadTime time.Time) error {
	log.Println("开始下载", year, "年", month, "月视频")

	filePath := consts.FlagConf.WorkDIr + string(os.PathSeparator) + strconv.Itoa(year) + string(os.PathSeparator) + strconv.Itoa(month)
	err := files.CheckDirOrCreate(filePath)
	if err != nil {
		return err
	}

	// 获取目标月份的第一天
	startTime := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	// 获取目标月份的最后一天
	lastDayOfMonth := date.GetLastDayOfMonth(startTime)

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

				// 如果当前分页的扫描视频时间远远晚于目标扫描日期范围,则直接大范围跳页 (每超过2天/3页)
				if days > 4 {
					jumpPage := (days / 2) * 3
					i += jumpPage
					log.Println("当前分页视频时间远远晚于目标扫描日期范围, 跳", jumpPage, "页")
					break
				}

				continue
			}

			if createTime.Before(lastDownloadTime) {
				// 当前视频创建时间早于上次下载时间，判断为下载任务完成
				log.Println("视频下载任务完成")
				return nil
			}
			if createTime.Before(startTime) {
				// 当前视频创建时间早于开始时间，判断为下载任务完成
				log.Println("视频下载任务完成")
				return nil
			}
			log.Println("视频符合时间范围,继续...")

			// 检查文件是否已经被下载,如果被下载则跳过
			// 尝试从目标路径中获取可能的文件名
			log.Printf("正在检查文件是否已下载, 作者: %s, 名称: %s", video.User.Name, video.Title)
			checkName := files.SanitizeFileName(fmt.Sprintf("[%s] %s", video.User.Name, video.Title))
			checkDir := filePath + string(os.PathSeparator)
			f := files.CheckVideoFileExist(checkName, checkDir)
			if f != "" {
				log.Printf("视频已存在: %s 跳过...\n", f)
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

			if len(videoUrl) < 1 {
				log.Printf("获取视频地址为空: %s\n", video.ID)
				// 跳过当前视频
				continue
			}

			// 排序默认下载最清晰的视频
			sort.Slice(videoUrl, func(i, j int) bool {
				return model.VideoDefinitionMap[videoUrl[i].Name] > model.VideoDefinitionMap[videoUrl[j].Name]
			})
			log.Printf("视频地址: %s\n", videoUrl[0].Src.Download)

			// 判断文件是否存在
			videoName := files.SanitizeFileName(fmt.Sprintf("[%s] %s [%s].mp4", video.User.Name, video.Title, videoUrl[0].Name))
			videoPath := filePath + string(os.PathSeparator) + videoName
			if files.CheckFileExists(videoPath) {
				log.Printf("文件已存在: %s\n", videoPath)
				continue
			}

			startDownloadTime := time.Now()
			log.Printf("开始下载视频: %s 分辨率: %s\n", videoPath, videoUrl[0].Name)
			err = request.Download(user, videoUrl[0].Src.Download, videoPath)
			if err != nil {
				log.Printf("下载视频失败: %s %s\n", videoName, err.Error())
				// 跳过当前视频
				continue
			}
			log.Println("视频下载完成, 耗时:", time.Since(startDownloadTime))
		}
		request.DelaySwitch = videoDownload
	}

	log.Println("视频下载任务完成")
	return nil
}

func loop() {
	lastScanTime := time.Time{}
	retryTimes := 0
	for {
		start := time.Now()
		log.Println("开始扫描任务")
		// 获取当前年月
		year := time.Now().Year()
		month := time.Now().Month()
		if err := Month(config.Config, year, int(month), lastScanTime); err != nil {
			if retryTimes > 3 {
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
		lastScanTime = time.Now()
		retryTimes = 0

		if useTime < consts.SCAN_STEP {
			time.Sleep(consts.SCAN_STEP - useTime)
		}
	}
}

func once() {
	log.Println("开始单次扫描任务")
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

// 流程为: 获取cookie -> 登录 -> 获取token -> 获取视频列表 -> 视频主页 -> 获取视频地址 -> 下载视频
func main() {
	if consts.FlagConf.Year != 0 || consts.FlagConf.Month != 0 {
		log.Println("指定了年份或月份,开始下载指定月份视频")
		once()
	} else {
		log.Println("未指定年份或月份,进行从当月开始的挂机下载任务")
		loop()
	}
}
