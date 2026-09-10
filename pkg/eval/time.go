package eval

import (
	"time"
)

// TimeNowMs 获取当前 Unix 时间戳 (毫秒)
func TimeNowMs() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// TimeNowSecs 获取当前 Unix 时间戳 (秒)
func TimeNowSecs() int64 {
	return time.Now().Unix()
}

// TimeNowMicros 获取当前 Unix 时间戳 (微秒)
func TimeNowMicros() int64 {
	return time.Now().UnixNano() / int64(time.Microsecond)
}

// TimeSleepMs 暂停执行指定毫秒数
func TimeSleepMs(ms int64) {
	if ms > 0 {
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
}

// TimeFormatNow 格式化当前系统时间为可读字符串
func TimeFormatNow() string {
	return time.Now().Format("2006-01-02 15:04:05.000")
}
