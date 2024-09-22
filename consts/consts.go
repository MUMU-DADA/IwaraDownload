package consts

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/MUMU-DADA/structflag"
)

const (
	MODEL_NAME      = "IwaraDownload"                             // 模块名
	DEFAULT_WORKDIR = "." + string(os.PathSeparator) + MODEL_NAME // 默认下载目录
	LOG_FILE_NAME   = MODEL_NAME + ".log"                         // 日志文件名
	LOG_PATH        = "." + string(os.PathSeparator) + LOG_FILE_NAME

	SCAN_STEP = time.Minute * 10 // 多久执行一次扫描任务
)

var (
	FlagConf config // 程序解析参数
)

// 程序运行参数
type config struct {
	SaveLog  bool   `flag:"savelog" sort:"l" default:"true" usage:"是否保存日志文件"`      // 是否保存日志文件
	WorkDIr  string `flag:"workdir" sort:"d" default:"" usage:"下载目录"`              // 下载目录
	Username string `flag:"username" sort:"u" default:"" usage:"指定用户名,比配置文件优先级要高"` // 指定用户名
	Password string `flag:"password" sort:"p" default:"" usage:"指定密码,比配置文件优先级要高"`  // 指定密码
}

func init() {
	// 初始化配置
	structflag.Load(&FlagConf)

	// 写入日志文件
	if FlagConf.SaveLog {
		// 创建日志文件
		file, err := os.OpenFile(LOG_PATH, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("打开日志文件失败: %v", err)
		}
		// 创建多路输出
		multiWriter := io.MultiWriter(os.Stdout, file)
		log.SetOutput(multiWriter)

		// 设置日志前缀和标志
		log.SetPrefix("[" + MODEL_NAME + "] ")
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
	log.Println("程序启动")

	// 初始化下载目录
	if FlagConf.WorkDIr == "" {
		FlagConf.WorkDIr = DEFAULT_WORKDIR
	}

	// 检查目录是否存在,不存着则创建
	if err := checkDirOrCreate(FlagConf.WorkDIr); err != nil {
		log.Fatalf("检查目录失败: %v", err)
	}
}
