package handlers

import (
	"report_api/service"

	"github.com/gin-gonic/gin"
)

type KsCallbackHandler struct {
	CommonHandler
}

var KsCallbackApiHandler = new(KsCallbackHandler)

func init() {
	KsCallbackApiHandler.postMapping("upload_ks_excel", uploadKsExcel) // 新增Excel上传接口
}

func uploadKsExcel(ctx *gin.Context) {
	service.UploadKsExcelFiles(ctx)
	// service层已经处理了响应，这里不需要再返回JSON
}
