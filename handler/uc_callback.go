package handlers

import (
	"report_api/service"

	"github.com/gin-gonic/gin"
)

type UcCallbackHandler struct {
	CommonHandler
}

var UcCallbackApiHandler = new(GdtCallbackHandler)

func init() {
	HonorCallbackApiHandler.postMapping("upload_uc_excel", uploadUcExcel) // 新增Excel上传接口
}

func uploadUcExcel(ctx *gin.Context) {
	service.UploadUcExcelFiles(ctx)
	// service层已经处理了响应，这里不需要再返回JSON
}
