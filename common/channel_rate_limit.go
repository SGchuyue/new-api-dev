package common

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// channelRpmKeyPrefix Redis key 前缀，用于渠道级别 RPM 计数
const channelRpmKeyPrefix = "channel_rpm:"

// CheckChannelRPM 检查渠道是否达到 RPM 限制
// 返回 true 表示允许请求，false 表示已达到限制
// 如果 rpmLimit 为 0，表示不限制，始终返回 true
func CheckChannelRPM(channelId int, rpmLimit int) bool {
	if rpmLimit <= 0 {
		return true
	}

	if RedisEnabled {
		return checkChannelRPMRedis(channelId, rpmLimit)
	}

	return true
}

// IncrementChannelRPM 对渠道的 RPM 计数器 +1
// 在请求成功分发到渠道后调用
func IncrementChannelRPM(channelId int, rpmLimit int) {
	if rpmLimit <= 0 {
		return
	}

	if RedisEnabled {
		incrementChannelRPMRedis(channelId, rpmLimit)
	}
}

// checkChannelRPMRedis 使用 Redis 固定窗口算法检查渠道 RPM
// key 格式: channel_rpm:{channel_id}:{当前分钟时间戳}
func checkChannelRPMRedis(channelId int, rpmLimit int) bool {
	ctx := context.Background()
	rdb := RDB
	key := getChannelRpmKey(channelId)

	count, err := rdb.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		// Redis 错误，放行请求
		SysLog(fmt.Sprintf("failed to get channel rpm counter: channel_id=%d, error=%v", channelId, err))
		return true
	}

	if count >= rpmLimit {
		return false
	}

	return true
}

// incrementChannelRPMRedis 使用 Redis INCR + EXPIRE 实现固定窗口计数
func incrementChannelRPMRedis(channelId int, rpmLimit int) {
	ctx := context.Background()
	rdb := RDB
	key := getChannelRpmKey(channelId)

	pipe := rdb.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 62*time.Second) // 多给 2 秒余量，避免边界问题

	_, err := pipe.Exec(ctx)
	if err != nil {
		SysLog(fmt.Sprintf("failed to increment channel rpm counter: channel_id=%d, error=%v", channelId, err))
		return
	}

	count := incrCmd.Val()
	if count == 1 {
		// 首次设置，确保过期时间为 60 秒（覆盖可能已存在的 key）
		rdb.Expire(ctx, key, 62*time.Second)
	}

	if count > int64(rpmLimit) {
		SysLog(fmt.Sprintf("channel %d rpm exceeded: count=%d, limit=%d", channelId, count, rpmLimit))
	}
}

// getChannelRpmKey 生成 Redis key
// key 格式: channel_rpm:{channel_id}:{当前分钟时间戳}
func getChannelRpmKey(channelId int) string {
	// 使用分钟级时间戳作为窗口标识
	minuteTimestamp := time.Now().Unix() / 60
	return fmt.Sprintf("%s%d:%d", channelRpmKeyPrefix, channelId, minuteTimestamp)
}
