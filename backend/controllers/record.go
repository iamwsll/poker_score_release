package controllers

import (
	"poker_score_backend/services"
	"poker_score_backend/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// RecordController 战绩统计控制器
type RecordController struct {
	recordService *services.RecordService
}

// NewRecordController 创建战绩统计控制器
func NewRecordController(recordService *services.RecordService) *RecordController {
	return &RecordController{
		recordService: recordService,
	}
}

// GetTonightRecords 获取今晚战绩
func (ctrl *RecordController) GetTonightRecords(c *gin.Context) {
	// 获取用户ID
	userID, _ := c.Get("user_id")

	// 获取时间参数
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	var startTime, endTime *time.Time
	if startTimeStr != "" {
		t, err := time.Parse(time.RFC3339, startTimeStr)
		if err == nil {
			startTime = &t
		}
	}
	if endTimeStr != "" {
		t, err := time.Parse(time.RFC3339, endTimeStr)
		if err == nil {
			endTime = &t
		}
	}

	// 查询战绩
	records, err := ctrl.recordService.GetTonightRecords(userID.(uint), startTime, endTime)
	if err != nil {
		utils.InternalServerError(c, "查询战绩失败")
		return
	}

	utils.Success(c, records)
}

