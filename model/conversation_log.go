package model

import (
	"github.com/QuantumNous/new-api/common"

	"github.com/bytedance/gopkg/util/gopool"

	"github.com/gin-gonic/gin"
)

// ConversationLog stores full request/response bodies
// body 字段使用 gzip 压缩存储，节省 70-80% 空间
type ConversationLog struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	RequestId    string `json:"request_id" gorm:"type:varchar(64);index;default:''"`
	UserId       int    `json:"user_id" gorm:"index"`
	Username     string `json:"username" gorm:"type:varchar(64);index;default:''"`
	ModelName    string `json:"model_name" gorm:"type:varchar(128);index;default:''"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint;index"`
	RequestBody  string `json:"request_body" gorm:"type:longtext"`
	ResponseBody string `json:"response_body" gorm:"type:longtext"`
	Archived     int    `json:"archived" gorm:"type:tinyint;default:0;index"`
}

// ConversationLogMeta lightweight metadata table, stays in MySQL permanently for fast queries
type ConversationLogMeta struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	RequestId    string `json:"request_id" gorm:"type:varchar(64);uniqueIndex;default:''"`
	UserId       int    `json:"user_id" gorm:"index"`
	Username     string `json:"username" gorm:"type:varchar(64);index;default:''"`
	ModelName    string `json:"model_name" gorm:"type:varchar(128);index;default:''"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint;index"`
	RequestSize  int    `json:"request_size" gorm:"default:0"`
	ResponseSize int    `json:"response_size" gorm:"default:0"`
	IsArchived   int    `json:"is_archived" gorm:"type:tinyint;default:0;index"`
	ArchivedAt   int64  `json:"archived_at" gorm:"bigint;default:0"`
}

// GetDecompressedRequestBody 获取解压后的请求体
func (c *ConversationLog) GetDecompressedRequestBody() string {
	return common.GzipDecompress(c.RequestBody)
}

// GetDecompressedResponseBody 获取解压后的响应体
func (c *ConversationLog) GetDecompressedResponseBody() string {
	return common.GzipDecompress(c.ResponseBody)
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
	requestId := c.GetString(common.RequestIdKey)
	username := c.GetString("username")
	timestamp := common.GetTimestamp()

	record := &ConversationRecord{
		RequestId:    requestId,
		UserId:       userId,
		Username:     username,
		ModelName:    modelName,
		CreatedAt:    timestamp,
		RequestBody:  requestBody,
		ResponseBody: responseBody,
	}
	gopool.Go(func() {
		// write main record first (core data)
		if err := SaveConversationLog(record); err != nil {
			common.SysLog("failed to record conversation log: " + err.Error())
			return
		}
		// then write meta (auxiliary index)
		meta := &ConversationLogMeta{
			RequestId:    requestId,
			UserId:       userId,
			Username:     username,
			ModelName:    modelName,
			CreatedAt:    timestamp,
			RequestSize:  len(requestBody),
			ResponseSize: len(responseBody),
		}
		if err := DB.Create(meta).Error; err != nil {
			common.SysLog("failed to record conversation log meta: " + err.Error())
		}
	})
}

// RecordConversationLogFromData 异步写入对话日志，不依赖 gin.Context
func RecordConversationLogFromData(userId int, modelName, requestId, username, requestBody, responseBody string) {
	if requestBody == "" && responseBody == "" {
		return
	}
	timestamp := common.GetTimestamp()
	record := &ConversationRecord{
		RequestId:    requestId,
		UserId:       userId,
		Username:     username,
		ModelName:    modelName,
		CreatedAt:    timestamp,
		RequestBody:  requestBody,
		ResponseBody: responseBody,
	}
	// write main record first (core data), then meta (auxiliary index)
	if err := SaveConversationLog(record); err != nil {
		common.SysLog("failed to record conversation log: " + err.Error())
		return
	}
	meta := &ConversationLogMeta{
		RequestId:    requestId,
		UserId:       userId,
		Username:     username,
		ModelName:    modelName,
		CreatedAt:    timestamp,
		RequestSize:  len(requestBody),
		ResponseSize: len(responseBody),
	}
	if err := DB.Create(meta).Error; err != nil {
		common.SysLog("failed to record conversation log meta: " + err.Error())
	}
}
