package middleware

import (
	"github.com/QuantumNous/new-api/common"
)

// CheckChannelRPM 检查渠道是否达到 RPM 限制
// 返回 true 表示允许请求，false 表示已达到限制
// 如果 rpmLimit 为 0，表示不限制，始终返回 true
// 委托给 common 包的实现
func CheckChannelRPM(channelId int, rpmLimit int) bool {
	return common.CheckChannelRPM(channelId, rpmLimit)
}

// IncrementChannelRPM 对渠道的 RPM 计数器 +1
// 在请求成功分发到渠道后调用
// 委托给 common 包的实现
func IncrementChannelRPM(channelId int, rpmLimit int) {
	common.IncrementChannelRPM(channelId, rpmLimit)
}
