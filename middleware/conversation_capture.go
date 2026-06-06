package middleware

import (
	"bytes"
	"io"
	"sync"

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
		// Update context on each Write so RecordConsumeLog can read it
		w.ctx.Set("log_response_body", w.buf.String())
	}
	return w.ResponseWriter.Write(b)
}

// ConversationCapture captures request and response bodies for conversation logging.
// Stores them in gin context as "log_request_body" and "log_response_body".
func ConversationCapture() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Capture request body
		if c.Request != nil && c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, int64(maxCaptureBodySize+1)))
			if err == nil && len(bodyBytes) > 0 {
				if len(bodyBytes) > maxCaptureBodySize {
					bodyBytes = bodyBytes[:maxCaptureBodySize]
				}
				c.Set("log_request_body", string(bodyBytes))
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// Get buffer from pool
		buf := bufPool.Get().(*bytes.Buffer)
		buf.Reset()

		// Wrap response writer
		capture := &captureResponseWriter{
			ResponseWriter: c.Writer,
			buf:            buf,
			ctx:            c,
		}
		c.Writer = capture

		c.Next()

		// Return buffer to pool
		buf.Reset()
		bufPool.Put(buf)
	}
}
