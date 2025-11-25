package handlers

import (
	"report_api/service"

	"github.com/gin-gonic/gin"
)

type JuliangCallbackHandler struct {
	CommonHandler
}

var CallbackApiHandler = new(JuliangCallbackHandler)

func init() {
	CallbackApiHandler.postMapping("upload_excel", uploadExcel) // 新增Excel上传接口
}

func uploadExcel(ctx *gin.Context) {
	service.UploadExcelFiles(ctx)
	// service层已经处理了响应，这里不需要再返回JSON
}
