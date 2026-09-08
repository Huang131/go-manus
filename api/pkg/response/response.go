package response

import (
	"errors"
	"net/http"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// TotalResponse 带总数的响应结构
type TotalResponse struct {
	Code  int         `json:"code"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
	Total int         `json:"total"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

// SuccessWithMsg 带消息的成功响应
func SuccessWithMsg(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  msg,
		Data: data,
	})
}

// SuccessWithTotal 带总数的成功响应
func SuccessWithTotal(c *gin.Context, data interface{}, total int) {
	c.JSON(http.StatusOK, TotalResponse{
		Code:  0,
		Msg:   "success",
		Data:  data,
		Total: total,
	})
}

// Error 错误响应
func Error(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, Response{
		Code: 400,
		Msg:  msg,
		Data: nil,
	})
}

// FromError 将业务错误映射为统一响应。
func FromError(c *gin.Context, err error) {
	if err == nil {
		Success(c, nil)
		return
	}

	var ae *apperr.Error
	if errors.As(err, &ae) {
		c.JSON(ae.Status(), Response{
			Code: ae.Code(),
			Msg:  ae.Msg,
			Data: nil,
		})
		return
	}

	c.JSON(http.StatusInternalServerError, Response{
		Code: http.StatusInternalServerError,
		Msg:  "internal server error",
		Data: nil,
	})
}
