package model

// File 结构体表示文件信息
type File struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	Path            string `json:"path"`
	Name            string `json:"name"`
	Mime            string `json:"mime"`
	Size            int64  `json:"size"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	Duration        int    `json:"duration"`
	NumThumbnails   int    `json:"numThumbnails"`
	AnimatedPreview bool   `json:"animatedPreview"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

// Avatar 结构体表示用户的头像信息
type Avatar struct {
	File
}

// Artist 结构体表示用户信息
type Artist struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Username       string `json:"username"`
	Status         string `json:"status"`
	Role           string `json:"role"`
	FollowedBy     bool   `json:"followedBy"`
	Following      bool   `json:"following"`
	Friend         bool   `json:"friend"`
	Premium        bool   `json:"premium"`
	CreatorProgram bool   `json:"creatorProgram"`
	Locale         string `json:"locale"`
	SeenAt         string `json:"seenAt"`
	Avatar         Avatar `json:"avatar"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

// Tag 结构体表示标签信息
type Tag struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Sensitive bool   `json:"sensitive"`
}

// Result 结构体表示单个结果条目
type Result struct {
	ID              string      `json:"id"`
	Slug            string      `json:"slug"`
	Title           string      `json:"title"`
	Body            string      `json:"body"`
	Status          string      `json:"status"`
	Rating          string      `json:"rating"`
	Private         bool        `json:"private"`
	Unlisted        bool        `json:"unlisted"`
	Thumbnail       int         `json:"thumbnail"`
	EmbedUrl        interface{} `json:"embedUrl"` // 可以是 nil 或具体类型
	Liked           bool        `json:"liked"`
	NumLikes        int         `json:"numLikes"`
	NumViews        int         `json:"numViews"`
	NumComments     int         `json:"numComments"`
	File            File        `json:"file"`
	CustomThumbnail interface{} `json:"customThumbnail"` // 可以是 nil 或具体类型
	User            Artist      `json:"user"`
	Tags            []Tag       `json:"tags"`
	CreatedAt       string      `json:"createdAt"`
	UpdatedAt       string      `json:"updatedAt"`
	FileUrl         string      `json:"fileUrl"`
}

// PageDataRoot 结构体表示整个 JSON 数据
type PageDataRoot struct {
	Count   int      `json:"count"`
	Limit   int      `json:"limit"`
	Page    int      `json:"page"`
	Results []Result `json:"results"`
}

// -----------------------------------------------------------------------------------------------------------------

var (
	VideoDefinitionMap = map[string]int{
		"Source":  99999,
		"1080":    1080,
		"720":     720,
		"540":     540,
		"480":     480,
		"360":     360,
		"240":     240,
		"144":     144,
		"108":     108,
		"96":      96,
		"preview": 1,
	}
)

// VideoSrc 结构体表示源链接信息
type VideoSrc struct {
	View     string `json:"view"`
	Download string `json:"download"`
}

// Video 结构体表示单个视频条目
type Video struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Src       VideoSrc `json:"src"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
	Type      string   `json:"type"`
}
