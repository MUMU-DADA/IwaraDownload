package date

import "time"

// GetLastDayOfMonth 获取指定月份的最后一天
func GetLastDayOfMonth(t time.Time) time.Time {
	nextMonth := t.AddDate(0, 1, 0)                                    // 增加一个月
	lastDay := nextMonth.AddDate(0, 0, -nextMonth.Day())               // 减去当月天数得到最后一天
	return lastDay.Add(time.Hour*23 + time.Minute*59 + time.Second*59) // 设置时间为最后一秒
}
