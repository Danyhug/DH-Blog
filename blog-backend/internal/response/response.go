package response

import (
	"github.com/gin-gonic/gin"
)

type AjaxResult struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func Success() AjaxResult {
	return AjaxResult{
		Code: 1,
		Msg:  "success",
	}
}

func SuccessWithData(data interface{}) AjaxResult {
	return AjaxResult{
		Code: 1,
		Msg:  "success",
		Data: data,
	}
}

func Error(msg string) AjaxResult {
	return AjaxResult{
		Code: 0,
		Msg:  msg,
	}
}

func FailWithCode(c *gin.Context, code int, msg string) {
	c.JSON(code, AjaxResult{
		Code: 0,
		Msg:  msg,
	})
}

type PageResult struct {
	Total int64       `json:"total"`
	List  interface{} `json:"list"`
}

func Page(total, curr int64, list interface{}) PageResult {
	return PageResult{
		Total: total,
		List:  list,
	}
}
