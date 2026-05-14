package common

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"strings"
)

const GzipPrefix = "gz:"

// GzipCompress 压缩字符串，base64 编码确保存储为合法 UTF-8
// 短于 256 字节的不压缩（压缩后反而更大）
// 已压缩的（gz: 前缀）跳过
func GzipCompress(s string) string {
	if s == "" || len(s) < 256 || strings.HasPrefix(s, GzipPrefix) {
		return s
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(s)); err != nil {
		return s
	}
	if err := gz.Close(); err != nil {
		return s
	}
	// base64 编码，确保是合法 UTF-8 字符串，可存入 MySQL utf8mb4 列
	compressed := GzipPrefix + base64.StdEncoding.EncodeToString(buf.Bytes())
	if len(compressed) < len(s) {
		return compressed
	}
	return s
}

// GzipDecompress 解压字符串，兼容未压缩数据
// 自动处理 base64 编码（新格式）和原始二进制（旧格式）
func GzipDecompress(s string) string {
	if s == "" || !strings.HasPrefix(s, GzipPrefix) {
		return s
	}
	data := s[len(GzipPrefix):]
	// 尝试 base64 解码（新格式）
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		// 旧格式：原始二进制（不应出现在正常流程中）
		decoded = []byte(data)
	}
	gz, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return s
	}
	defer gz.Close()
	decompressed, err := io.ReadAll(gz)
	if err != nil {
		return s
	}
	return string(decompressed)
}

// IsGzipCompressed 检查字符串是否是 gzip 压缩的
func IsGzipCompressed(s string) bool {
	return strings.HasPrefix(s, GzipPrefix)
}
