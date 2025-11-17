package process

import (
	"IwaraDownload/consts"
	"IwaraDownload/model"
	"IwaraDownload/pkg/files"
	"encoding/json"
	nodes "github.com/MUMU-DADA/nodes"
	"log"
	"os"
)

const (
	configFileName = "user.json"
	configPath     = "." + string(os.PathSeparator) + configFileName
)

type ReadConfig[T any] struct {
	nodes.BaseNode[*model.User]
}

func (r *ReadConfig[T]) GetName() string {
	return "ReadConfig"
}

func (r *ReadConfig[T]) Execute(ctx *nodes.NodeContext[T]) (*nodes.NodeData[T], error) {
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
	if consts.FlagConf.Mode != 0 {
		c.Mode = model.DownloadMode(consts.FlagConf.Mode)
	}

	if c.Mode == model.HotMode {
		// 如果开启了热门下载模式, 需要设置下载监听页码
		if consts.FlagConf.HotPageLimit > 0 {
			// 优先使用命令行参数
			c.HotPageLimit = consts.FlagConf.HotPageLimit
		} else if c.HotPageLimit > 0 {
			// 其次使用配置的参数

		} else {
			// 否则使用默认值
			log.Println("未设置热门下载页数, 使用默认值:", consts.HOT_PAGE_DEFAULT_LIMIT)
			c.HotPageLimit = consts.HOT_PAGE_DEFAULT_LIMIT
		}
	}

	if c.Mode == model.ArtistMode {
		// 如果开启了艺术家模式, 需要设置监听艺术家

		// 优先使用命令行参数
		if consts.FlagConf.Artist != "" {
			c.DownloadArtists = []string{consts.FlagConf.Artist}
		}

		// 其次使用配置的参数
		// 配置文件已经预先载入,无需重复写入

		if len(c.DownloadArtists) == 0 {
			log.Fatalln("未设置艺术家, 请在配置中设置艺术家")
		}
	}

	if c.Username == "" || c.Password == "" {
		log.Fatalln("用户名或密码为空, 请使用 -u 和 -p 指定用户名密码,或者配置好", configFileName, "配置文件")
	}

	return nodes.ReturnNodeData(r, c), nil
}
