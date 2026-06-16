package response

import (
	"net/http"
	errx "storeready_ai/internal/pkg/errors"

	"github.com/gin-gonic/gin"
)

type msgKeyProvider interface {
	MsgKey() string
}

type msgParamsProvider interface {
	MsgParams() map[string]any
}

func WriteOK(c *gin.Context, data any, rid string) {
	c.JSON(http.StatusOK, OK(data, rid))
}

func WriteError(c *gin.Context, err error, rid string) {
	code := errx.CodeOf(err) // errors.Code (int32)
	msg := errx.MessageOf(err, "系统错误")
	status := errx.HTTPStatusByCode(code)
	msgKey, msgParams := resolveErrorI18n(err)

	if msgKey != "" || len(msgParams) > 0 {
		c.JSON(status, FailWithKey(int32(code), msg, msgKey, msgParams, rid))
		return
	}

	c.JSON(status, Fail(int32(code), msg, rid))
}

func AbortError(c *gin.Context, err error, rid string) {
	WriteError(c, err, rid)
	c.Abort()
}

func AbortFail(c *gin.Context, status int, code int32, msg, rid string) {
	c.JSON(status, Fail(code, msg, rid))
	c.Abort()
}

func resolveErrorI18n(err error) (string, map[string]any) {
	if err == nil {
		return "", nil
	}

	var msgKey string
	var msgParams map[string]any

	if v, ok := err.(msgKeyProvider); ok {
		msgKey = v.MsgKey()
	}
	if v, ok := err.(msgParamsProvider); ok {
		msgParams = v.MsgParams()
	}

	return msgKey, msgParams
}
