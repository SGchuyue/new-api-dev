package model

import (
	"fmt"
	"github.com/QuantumNous/new-api/common"
)

// ConversationRecord 对话日志统一数据结构
type ConversationRecord struct {
	RequestId    string
	UserId       int
	Username     string
	ModelName    string
	CreatedAt    int64
	RequestBody  string
	ResponseBody string
}

// SaveConversationLog 保存对话日志（MySQL + gzip 压缩）
func SaveConversationLog(record *ConversationRecord) error {
	log := &ConversationLog{
		RequestId:    record.RequestId,
		UserId:       record.UserId,
		Username:     record.Username,
		ModelName:    record.ModelName,
		CreatedAt:    record.CreatedAt,
		RequestBody:  common.GzipCompress(record.RequestBody),
		ResponseBody: common.GzipCompress(record.ResponseBody),
	}
	return LOG_DB.Create(log).Error
}

// GetConversationLog 查询对话日志（自动解压）
func GetConversationLog(requestId string) (*ConversationRecord, error) {
	var log ConversationLog
	err := LOG_DB.Where("request_id = ?", requestId).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &ConversationRecord{
		RequestId:    log.RequestId,
		UserId:       log.UserId,
		Username:     log.Username,
		ModelName:    log.ModelName,
		CreatedAt:    log.CreatedAt,
		RequestBody:  log.GetDecompressedRequestBody(),
		ResponseBody: log.GetDecompressedResponseBody(),
	}, nil
}

// InitLogStorage 初始化日志存储层
// 当前: MySQL + gzip 压缩
// 预留: 配置 MINIO_ENDPOINT 后可切换到 MinIO
func InitLogStorage() {
	minioEndpoint := common.GetEnvOrDefaultString("MINIO_ENDPOINT", "")
	if minioEndpoint != "" {
		common.SysLog("log storage: MinIO configured but not yet implemented, using MySQL + gzip")
	} else {
		common.SysLog("log storage: MySQL backend (gzip compressed)")
	}
}

// BatchCompressOldConversationLogs 批量压缩旧的未压缩对话日志
// 注意：无事务保护，但 GzipDecompress 是幂等的（对未压缩数据原样返回）
// 即使进程崩溃导致部分压缩部分未压缩，读取时也不会数据损坏
func BatchCompressOldConversationLogs(batchLimit int) (int, error) {
	var logs []ConversationLog
	err := LOG_DB.Where("request_body != '' AND request_body NOT LIKE ?", "gz:%").
		Limit(batchLimit).Order("id ASC").Find(&logs).Error
	if err != nil {
		return 0, fmt.Errorf("query failed: %w", err)
	}
	if len(logs) == 0 {
		return 0, nil
	}
	compressed := 0
	for i := range logs {
		cb := common.GzipCompress(logs[i].RequestBody)
		cr := common.GzipCompress(logs[i].ResponseBody)
		if cb != logs[i].RequestBody || cr != logs[i].ResponseBody {
			if err := LOG_DB.Model(&logs[i]).Updates(map[string]interface{}{
				"request_body":  cb,
				"response_body": cr,
			}).Error; err != nil {
				common.SysLog(fmt.Sprintf("compress conversation log id=%d failed: %s", logs[i].Id, err.Error()))
			} else {
				compressed++
			}
		}
	}
	return compressed, nil
}

// ============================================================
// MinIO 预留配置（不启用，不影响编译）
// ============================================================

// MinIOConfig MinIO 配置
type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// GetMinIOConfig 获取 MinIO 配置
func GetMinIOConfig() *MinIOConfig {
	return &MinIOConfig{
		Endpoint:  common.GetEnvOrDefaultString("MINIO_ENDPOINT", ""),
		AccessKey: common.GetEnvOrDefaultString("MINIO_ACCESS_KEY", "admin"),
		SecretKey: common.GetEnvOrDefaultString("MINIO_SECRET_KEY", ""),
		Bucket:    common.GetEnvOrDefaultString("MINIO_BUCKET", "conversation-logs"),
		UseSSL:    common.GetEnvOrDefaultBool("MINIO_USE_SSL", false),
	}
}

// ConversationLogIndex 对话日志索引（MinIO 启用时使用）
type ConversationLogIndex struct {
	Id               int    `json:"id" gorm:"primaryKey;autoIncrement"`
	RequestId        string `json:"request_id" gorm:"type:varchar(64);uniqueIndex;default:''"`
	UserId           int    `json:"user_id" gorm:"index"`
	Username         string `json:"username" gorm:"type:varchar(64);index;default:''"`
	ModelName        string `json:"model_name" gorm:"type:varchar(128);index;default:''"`
	CreatedAt        int64  `json:"created_at" gorm:"bigint;index"`
	StorageKey       string `json:"storage_key" gorm:"type:varchar(256);default:''"`
	StorageType      string `json:"storage_type" gorm:"type:varchar(16);default:'mysql'"`
	RequestBodySize  int    `json:"request_body_size" gorm:"default:0"`
	ResponseBodySize int    `json:"response_body_size" gorm:"default:0"`
}
