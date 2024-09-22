package main

import (
	"IwaraDownload/consts"
	"IwaraDownload/internal/request"
	"IwaraDownload/model"
	"IwaraDownload/pkg/config"
	"IwaraDownload/pkg/files"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/araddon/dateparse"
)

const (
	maxpage = 50 // 最大页数
)

// Month 开始月下载任务
func Month(user *model.User, year int, month int, lastDownloadTime time.Time) error {
	filePath := consts.FlagConf.WorkDIr + string(os.PathSeparator) + strconv.Itoa(year) + string(os.PathSeparator) + strconv.Itoa(month)
	err := files.CheckDirOrCreate(filePath)
	if err != nil {
		return err
	}

	startTime := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i <= maxpage; i++ {
		log.Printf("正在获取第%d页视频列表\n", i)
		// 尝试获取视频列表
		pageData, err := request.GetVideoData(user, i)
		if err != nil {
			// 视频页数据获取的失败是不能容忍的，直接返回错误
			log.Printf("获取视频列表失败: %s\n", err.Error())
			return err
		}
		log.Println("视频列表获取成功")

		for _, video := range pageData.Results {
			createTime, err := dateparse.ParseLocal(video.CreatedAt)
			if err != nil {
				log.Printf("解析时间失败: %s\n", err.Error())
				// 跳过当前视频
				continue
			}
			if createTime.Before(lastDownloadTime) {
				// 当前视频创建时间早于上次下载时间，判断为下载任务完成
				return nil
			}
			if createTime.Before(startTime) {
				// 当前视频创建时间早于开始时间，判断为下载任务完成
				return nil
			}

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
			for _, v := range videoUrl {
				log.Printf("获取到视频分辨率: %s", v.Name)
			}
			// 排序默认下载最清晰的视频
			sort.Slice(videoUrl, func(i, j int) bool {
				return model.VideoDefinitionMap[videoUrl[i].Name] > model.VideoDefinitionMap[videoUrl[j].Name]
			})

			// 判断文件是否存在
			videoName := files.SanitizeFileName(fmt.Sprintf("[%s] %s [%s].mp4", video.User.Name, video.Title, videoUrl[0].Name))
			videoPath := filePath + string(os.PathSeparator) + videoName
			if files.CheckFileExists(videoPath) {
				log.Printf("文件已存在: %s\n", videoPath)
				continue
			}

			log.Printf("开始下载视频: %s 分辨率: %s\n", videoPath, videoUrl[0].Name)
			err = request.Download(user, videoUrl[0].Src.Download, videoPath)
			if err != nil {
				log.Printf("下载视频失败: %s %s\n", videoName, err.Error())
				// 跳过当前视频
				continue
			}
		}
	}

	log.Println("视频下载任务完成")
	return nil
}

// 流程为: 获取cookie -> 登录 -> 获取token -> 获取视频列表 -> 视频主页 -> 获取视频地址 -> 下载视频
func main() {
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
