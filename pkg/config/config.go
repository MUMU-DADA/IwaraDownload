package config

import (
	"IwaraDownload/consts"
	"IwaraDownload/model"
	"IwaraDownload/pkg/files"
	"encoding/json"
	"log"
	"os"
)

const (
	configFileName = "user.json"
	configPath     = "." + string(os.PathSeparator) + configFileName
)

var (
	Config *model.User
)

func SaveConfig(c *model.User) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return files.WriteFile(configPath, data)
}

func init() {
	c := &model.User{}

	// 读取配置文件
	configFileData, err := files.ReadFile(configPath)
	if err != nil {
		log.Println("读取配置文件失败:", err, "尝试使用程序读取配置")
	}
	if err := json.Unmarshal(configFileData, c); err != nil {
		log.Println("解析配置文件json格式解析失败:", err, "尝试使用程序读取配置")
	}

	// 优先读取命令行参数
	if consts.FlagConf.Username != "" {
		c.Username = consts.FlagConf.Username
	}
	if consts.FlagConf.Password != "" {
		c.Password = consts.FlagConf.Password
	}
	if consts.FlagConf.Subscribed {
		c.Subscribe = consts.FlagConf.Subscribed
		// 如果命令行开启了订阅模式, 需要关闭热门下载模式
		c.Hot = false
	}
	if consts.FlagConf.Hot {
		c.Hot = consts.FlagConf.Hot
		// 如果命令行开启了订阅模式, 需要关闭热门下载模式
		c.Subscribe = false

		// 如果开启了热门下载模式, 需要设置下载监听页码
		if consts.FlagConf.HotPageLimit > 0 {
			// 优先使用命令行参数
			c.HotPageLimit = consts.FlagConf.HotPageLimit
		} else if c.HotPageLimit > 0 {
			// 其次使用配置的参数
			c.HotPageLimit = 0
		} else {
			// 否则使用默认值
			log.Println("未设置热门下载页数, 使用默认值:", consts.HOT_PAGE_DEFAULT_LIMIT)
			c.HotPageLimit = consts.HOT_PAGE_DEFAULT_LIMIT
		}
	}

	if c.Username == "" || c.Password == "" {
		log.Fatalln("用户名或密码为空, 请使用 -u 和 -p 指定用户名密码,或者配置好", configFileName, "配置文件")
	}
	if c.Hot && c.Subscribe {
		log.Fatalln("订阅模式和热门模式不能同时开启")
	}

	Config = c
}
