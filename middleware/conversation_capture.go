package middleware

import (
	"bytes"
	"io"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const maxCaptureBodySize = 512 * 1024 // 512KB

var bufPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

type captureResponseWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
	ctx *gin.Context
}

func (w *captureResponseWriter) Write(b []byte) (int, error) {
	if w.buf.Len() < maxCaptureBodySize {
		remaining := maxCaptureBodySize - w.buf.Len()
		if len(b) > remaining {
			_, _ = w.buf.Write(b[:remaining])
		} else {
			_, _ = w.buf.Write(b)
		}
		w.ctx.Set("log_response_body", w.buf.String())
	}
	return w.ResponseWriter.Write(b)
}

func ConversationCapture() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Capture request body via BodyStorage
		// BodyStorage is cached by Distribute, but position is at the end
		// so we must Seek(0) before reading
		if bodyStorage, err := common.GetBodyStorage(c); err == nil {
			bodyStorage.Seek(0, io.SeekStart)
			bodyBytes, err := io.ReadAll(io.LimitReader(bodyStorage, int64(maxCaptureBodySize+1)))
			if err == nil && len(bodyBytes) > 0 {
				if len(bodyBytes) > maxCaptureBodySize {
					bodyBytes = bodyBytes[:maxCaptureBodySize]
				}
				c.Set("log_request_body", string(bodyBytes))
			}
			// Seek back to start for downstream handlers
			bodyStorage.Seek(0, io.SeekStart)
		}

		buf := bufPool.Get().(*bytes.Buffer)
		buf.Reset()

		capture := &captureResponseWriter{
			ResponseWriter: c.Writer,
			buf:            buf,
			ctx:            c,
		}
		c.Writer = capture

		c.Next()

		buf.Reset()
		bufPool.Put(buf)
	}
}
