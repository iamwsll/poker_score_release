package utils

import (
	"math/rand"
	"time"
)

// GenerateRoomCode 生成6位数字房间号
func GenerateRoomCode() string {
	// 使用当前时间作为随机数种子
	rand.Seed(time.Now().UnixNano())
	
	// 生成100000-999999之间的随机数
	code := rand.Intn(900000) + 100000
	
	return string(rune(code/100000+'0')) +
		string(rune((code/10000)%10+'0')) +
		string(rune((code/1000)%10+'0')) +
		string(rune((code/100)%10+'0')) +
		string(rune((code/10)%10+'0')) +
		string(rune(code%10+'0'))
}

