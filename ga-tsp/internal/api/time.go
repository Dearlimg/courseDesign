package api

import "time"

// timeNowNanos 返回当前时间的纳秒数，用作默认随机种子。
func timeNowNanos() int64 { return time.Now().UnixNano() }
