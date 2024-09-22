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

	if c.Username == "" || c.Password == "" {
		log.Fatalln("用户名或密码为空, 请使用 -u 和 -p 指定用户名密码,或者配置好", configFileName, "配置文件")
	}

	Config = c
}
