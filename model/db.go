package model

// VideoData 视频数据
type VideoData struct {
	Video *Result
	Files []*Video
}

// Data 存储视频数据
type Data struct {
	VideoMap map[string]VideoData
}
