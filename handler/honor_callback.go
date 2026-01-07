package handlers

import (
	"report_api/service"

	"github.com/gin-gonic/gin"
)

type HonorCallbackHandler struct {
	CommonHandler
}

var HonorCallbackApiHandler = new(GdtCallbackHandler)

func init() {
	HonorCallbackApiHandler.postMapping("upload_honor_excel", uploadHonorExcel) // 新增Excel上传接口
}

func uploadHonorExcel(ctx *gin.Context) {
	service.UploadHonorExcelFiles(ctx)
	// service层已经处理了响应，这里不需要再返回JSON
}
