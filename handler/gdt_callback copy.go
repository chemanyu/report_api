package handlers

import (
	"report_api/service"

	"github.com/gin-gonic/gin"
)

type GdtCallbackHandler struct {
	CommonHandler
}

var GdtallbackApiHandler = new(GdtCallbackHandler)

func init() {
	GdtallbackApiHandler.postMapping("upload_gdt_excel", uploadGdtExcel) // 新增Excel上传接口
}

func uploadGdtExcel(ctx *gin.Context) {
	service.UploadGdtExcelFiles(ctx)
	// service层已经处理了响应，这里不需要再返回JSON
}
