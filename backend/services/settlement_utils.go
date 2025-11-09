package services

import (
	"strconv"
	"strings"
)

// calculateRmbAmount 将积分根据房间倍率转换成人民币金额
func calculateRmbAmount(chipAmount int, chipRate string) float64 {
	parts := strings.Split(chipRate, ":")
	if len(parts) != 2 {
		return 0
	}

	chipPart, err1 := strconv.ParseFloat(parts[0], 64)
	rmbPart, err2 := strconv.ParseFloat(parts[1], 64)

	if err1 != nil || err2 != nil || chipPart == 0 {
		return 0
	}

	return float64(chipAmount) * rmbPart / chipPart
}
