package files

import (
	"IwaraDownload/model"
	"os"
	"regexp"
	"strings"
)

// CheckDirOrCreate 检查目录是否存在，不存在则创建
func CheckDirOrCreate(dir string) error {
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

// WriteFile 将数据写入文件。
func WriteFile(filePath string, data []byte) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

// ReadFile 读取文件内容。
func ReadFile(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

// SanitizeFileName 将字符串转换为Windows文件系统支持的文件名格式
func SanitizeFileName(filename string) string {
	// 替换非法字符
	illegalChars := regexp.MustCompile(`[\\/:*?"<>|\r\n]+`)
	sanitized := illegalChars.ReplaceAllString(filename, "_")

	// 进一步确保文件名不是保留名
	reservedNames := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}

	// 检查是否为保留名
	for _, name := range reservedNames {
		if sanitized == name {
			sanitized += "_"
			break
		}
	}

	// 确保文件名不以点或空格开头
	if strings.HasPrefix(sanitized, ".") || strings.HasPrefix(sanitized, " ") {
		sanitized = "_" + sanitized
	}

	// 确保文件名不为空
	if sanitized == "" {
		sanitized = "_"
	}

	return sanitized
}

// CheckFileExists 检查文件是否存在
func CheckFileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

// CheckVideoFileExist 检查指定目录下的视频文件是否存在
func CheckVideoFileExist(baseName string, dirPath string) string {
	for k, _ := range model.VideoDefinitionMap {
		tempName := baseName + " [" + k + "].mp4"
		if CheckFileExists(dirPath + string(os.PathSeparator) + tempName) {
			return tempName
		}
	}
	return ""
}
