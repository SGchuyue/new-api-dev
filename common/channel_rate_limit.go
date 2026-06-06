package common

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var channelRpmNoRedisWarnOnce sync.Once

const channelRpmKeyPrefix = "channel_rpm:"

var channelRpmLua = redis.NewScript(`
local current = tonumber(redis.call('GET', KEYS[1])) or 0
if current >= tonumber(ARGV[1]) then
    return 0
end
local new_count = redis.call('INCR', KEYS[1])
redis.call('EXPIRE', KEYS[1], 62)
return 1
`)

// AllowAndRecordChannelRPM 检查并记录渠道+模型的 RPM（原子操作）
// model 为空时按渠道总量限制
func AllowAndRecordChannelRPM(channelId int, model string, rpmLimit int) bool {
	if rpmLimit <= 0 {
		return true
	}
	if !RedisEnabled {
		channelRpmNoRedisWarnOnce.Do(func() {
			SysLog("warning: channel RPM limit configured but Redis not enabled, limit not enforced for all channels")
		})
		return true
	}
	return allowAndRecordChannelRPMRedis(channelId, model, rpmLimit)
}

func allowAndRecordChannelRPMRedis(channelId int, model string, rpmLimit int) bool {
	ctx := context.Background()
	rdb := RDB
	key := getChannelRpmKey(channelId, model)

	result, err := channelRpmLua.Run(ctx, rdb, []string{key}, rpmLimit).Int64()
	if err != nil {
		SysLog(fmt.Sprintf("failed to check/increment channel rpm counter: channel_id=%d model=%s, error=%v", channelId, model, err))
		return true
	}
	return result == 1
}

// CheckChannelRPM 检查渠道+模型的 RPM（只读，用于选渠道时预判）
// model 为空时按渠道总量限制
func CheckChannelRPM(channelId int, model string, rpmLimit int) bool {
	if rpmLimit <= 0 {
		return true
	}
	if !RedisEnabled {
		return true
	}

	ctx := context.Background()
	rdb := RDB
	key := getChannelRpmKey(channelId, model)

	count, err := rdb.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		SysLog(fmt.Sprintf("failed to get channel rpm counter: channel_id=%d model=%s, error=%v", channelId, model, err))
		return true
	}
	return count < rpmLimit
}

// getChannelRpmKey 生成 key
// model 为空: channel_rpm:{channelId}:{minuteTimestamp}
// model 非空: channel_rpm:{channelId}:{model}:{minuteTimestamp}
func getChannelRpmKey(channelId int, model string) string {
	minuteTimestamp := time.Now().Unix() / 60
	if model != "" {
		return fmt.Sprintf("%s%d:%s:%d", channelRpmKeyPrefix, channelId, model, minuteTimestamp)
	}
	return fmt.Sprintf("%s%d:%d", channelRpmKeyPrefix, channelId, minuteTimestamp)
}
