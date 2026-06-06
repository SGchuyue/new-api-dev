package model

import (
	"bytes"
	"context"
	// json operations use common.Marshal/Unmarshal
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var minioClient *minio.Client
var minioBucket string
var minioPrefix string
var minioEnabled bool

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

// ArchiveRecord 归档记录（上传到 MinIO 的 JSON 格式）
type ArchiveRecord struct {
	RequestId    string `json:"request_id"`
	UserId       int    `json:"user_id"`
	Username     string `json:"username"`
	ModelName    string `json:"model_name"`
	CreatedAt    int64  `json:"created_at"`
	RequestBody  string `json:"request_body"`
	ResponseBody string `json:"response_body"`
}

// InitLogStorage 初始化日志存储层
func InitLogStorage() {
	endpoint := common.GetEnvOrDefaultString("MINIO_ENDPOINT", "")
	if endpoint == "" {
		minioEnabled = false
		common.SysLog("log storage: MySQL backend (gzip compressed)")
		return
	}

	accessKey := common.GetEnvOrDefaultString("MINIO_ACCESS_KEY", "")
	secretKey := common.GetEnvOrDefaultString("MINIO_SECRET_KEY", "")
	useSSL := common.GetEnvOrDefaultBool("MINIO_USE_SSL", false)
	minioBucket = common.GetEnvOrDefaultString("MINIO_BUCKET", "conversation-logs")
	minioPrefix = common.GetEnvOrDefaultString("MINIO_PREFIX", "")
	var err error
	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		minioEnabled = false
		common.SysLog(fmt.Sprintf("log storage: MinIO init failed: %v, falling back to MySQL", err))
		return
	}

	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, minioBucket)
	if err != nil {
		minioEnabled = false
		common.SysLog(fmt.Sprintf("log storage: MinIO bucket check failed: %v, falling back to MySQL", err))
		return
	}
	if !exists {
		err = minioClient.MakeBucket(ctx, minioBucket, minio.MakeBucketOptions{})
		if err != nil {
			minioEnabled = false
			common.SysLog(fmt.Sprintf("log storage: MinIO create bucket failed: %v, falling back to MySQL", err))
			return
		}
	}

	minioEnabled = true
	common.SysLog(fmt.Sprintf("log storage: MySQL + MinIO (bucket: %s, endpoint: %s)", minioBucket, endpoint))
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

// GetConversationLog 查询对话日志（先 MySQL，后 MinIO，自动解压）
func GetConversationLog(requestId string) (*ConversationRecord, error) {
	var log ConversationLog
	err := LOG_DB.Where("request_id = ?", requestId).First(&log).Error
	if err == nil {
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

	// MySQL 没有，尝试从 MinIO 读取
	if minioEnabled {
		return getConversationLogFromMinIO(requestId)
	}

	return nil, err
}

// getConversationLogFromMinIO 从 MinIO 读取归档数据
func getConversationLogFromMinIO(requestId string) (*ConversationRecord, error) {
	objectKey := getObjectKey(requestId)
	ctx := context.Background()

	obj, err := minioClient.GetObject(ctx, minioBucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("minio get failed: %w", err)
	}
	defer obj.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(obj); err != nil {
		return nil, fmt.Errorf("minio read failed: %w", err)
	}

	// MinIO 存的是 gzip 压缩后的 JSON，先解压
	decompressed := common.GzipDecompress(buf.String())

	var record ArchiveRecord
	if err := common.Unmarshal([]byte(decompressed), &record); err != nil {
		return nil, fmt.Errorf("minio unmarshal failed: %w", err)
	}

	return &ConversationRecord{
		RequestId:    record.RequestId,
		UserId:       record.UserId,
		Username:     record.Username,
		ModelName:    record.ModelName,
		CreatedAt:    record.CreatedAt,
		RequestBody:  record.RequestBody,
		ResponseBody: record.ResponseBody,
	}, nil
}

// getObjectKey 生成 MinIO 对象路径: YYYY/MM/DD/{request_id}.json.gz
func getObjectKey(requestId string) string {
	var path string
	if len(requestId) >= 8 {
		year := requestId[:4]
		month := requestId[4:6]
		day := requestId[6:8]
		path = fmt.Sprintf("%s/%s/%s/%s.json.gz", year, month, day, requestId)
	} else {
		path = fmt.Sprintf("unknown/%s.json.gz", requestId)
	}
	if minioPrefix != "" {
		return minioPrefix + "/" + path
	}
	return path
}


// cleanupStaleArchivedRecords removes records that were marked archived but not deleted (crash recovery)
func cleanupStaleArchivedRecords() {
	totalCleaned := 0
	for {
		var staleLogs []ConversationLog
		result := LOG_DB.Where("archived = 1").Limit(500).Find(&staleLogs)
		if result.Error != nil || len(staleLogs) == 0 {
			break
		}
		successCount := 0
		for i := range staleLogs {
			if err := LOG_DB.Delete(&staleLogs[i]).Error; err != nil {
				common.SysLog(fmt.Sprintf("cleanup stale archived record failed: request_id=%s, error=%v", staleLogs[i].RequestId, err))
			} else {
				successCount++
			}
		}
		totalCleaned += successCount
		if successCount == 0 {
			common.SysLog("cleanup: no progress made, stopping to avoid infinite loop")
			break
		}
		if len(staleLogs) < 500 {
			break
		}
	}
	if totalCleaned > 0 {
		common.SysLog(fmt.Sprintf("cleanup: removed %d stale archived records", totalCleaned))
	}
}

// ArchiveOldConversationLogs 归档超过 retentionDays 天的 conversation_logs 到 MinIO
// 返回：归档成功数量，删除成功数量，错误
func ArchiveOldConversationLogs(retentionDays int, batchLimit int) (archived int, deleted int, err error) {
	if !minioEnabled {
		return 0, 0, fmt.Errorf("minio not enabled")
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays).Unix()
	ctx := context.Background()

	var logs []ConversationLog
	err = LOG_DB.Where("created_at < ? AND archived = 0", cutoff).
		Order("id ASC").
		Limit(batchLimit).
		Find(&logs).Error
	if err != nil {
		return 0, 0, fmt.Errorf("query old logs failed: %w", err)
	}

	for i := range logs {
		// 构建归档数据
		record := ArchiveRecord{
			RequestId:    logs[i].RequestId,
			UserId:       logs[i].UserId,
			Username:     logs[i].Username,
			ModelName:    logs[i].ModelName,
			CreatedAt:    logs[i].CreatedAt,
			RequestBody:  logs[i].GetDecompressedRequestBody(),
			ResponseBody: logs[i].GetDecompressedResponseBody(),
		}

		data, err := common.Marshal(record)
		if err != nil {
			common.SysLog(fmt.Sprintf("archive marshal failed: request_id=%s, error=%v", logs[i].RequestId, err))
			continue
		}

		// gzip 压缩后上传
		compressed := common.GzipCompress(string(data))
		objectKey := getObjectKey(logs[i].RequestId)
		reader := bytes.NewReader([]byte(compressed))

		_, err = minioClient.PutObject(ctx, minioBucket, objectKey, reader, int64(len(compressed)),
			minio.PutObjectOptions{ContentType: "application/gzip"})
		if err != nil {
			common.SysLog(fmt.Sprintf("archive upload failed: request_id=%s, error=%v", logs[i].RequestId, err))
			continue
		}
		// mark as archived
		if err := LOG_DB.Model(&logs[i]).Update("archived", 1).Error; err != nil {
			common.SysLog(fmt.Sprintf("archive mark failed: request_id=%s, error=%v", logs[i].RequestId, err))
			continue
		}

		// update meta table
		if err := DB.Model(&ConversationLogMeta{}).Where("request_id = ?", logs[i].RequestId).
			Updates(map[string]interface{}{"is_archived": 1, "archived_at": time.Now().Unix()}).Error; err != nil {
			common.SysLog(fmt.Sprintf("archive meta update failed: request_id=%s, error=%v", logs[i].RequestId, err))
		}

		// delete MySQL record
		if err := LOG_DB.Delete(&logs[i]).Error; err != nil {
			common.SysLog(fmt.Sprintf("archive delete failed: request_id=%s, error=%v", logs[i].RequestId, err))
			continue
		}
		deleted++
		archived++
	}

	return archived, deleted, nil
}
