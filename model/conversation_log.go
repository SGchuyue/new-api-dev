package model

import (
	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
)

// ConversationLog stores full request/response bodies separately from logs.other
type ConversationLog struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	RequestId    string `json:"request_id" gorm:"type:varchar(64);index;default:''"`
	UserId       int    `json:"user_id" gorm:"index"`
	Username     string `json:"username" gorm:"type:varchar(64);index;default:''"`
	ModelName    string `json:"model_name" gorm:"type:varchar(128);index;default:''"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint;index"`
	RequestBody  string `json:"request_body" gorm:"type:text"`
	ResponseBody string `json:"response_body" gorm:"type:text"`
}

func RecordConversationLog(c *gin.Context, userId int, modelName string) {
	if c == nil {
		return
	}
	requestBody := c.GetString("log_request_body")
	responseBody := c.GetString("log_response_body")
	if requestBody == "" && responseBody == "" {
		return
	}
	// 提取 context 数据，避免 goroutine 中访问已释放的 context
	requestId := c.GetString(common.RequestIdKey)
	username := c.GetString("username")
	timestamp := common.GetTimestamp()

	log := &ConversationLog{
		RequestId:    requestId,
		UserId:       userId,
		Username:     username,
		ModelName:    modelName,
		CreatedAt:    timestamp,
		RequestBody:  requestBody,
		ResponseBody: responseBody,
	}
	// 异步写入，不阻塞 API 响应
	go func() {
		if err := LOG_DB.Create(log).Error; err != nil {
			common.SysLog("failed to record conversation log: " + err.Error())
		}
	}()
}
