package handlers

import (
	"report_api/service"

	"github.com/gin-gonic/gin"
)

type TapbaoReportApiHandler struct {
	CommonHandler
}

var GetTaobaoReportApiHandler = new(TapbaoReportApiHandler)

func init() {
	GetTaobaoReportApiHandler.postMapping("activity", getActivityReport) // 福利购CPA
}

func getActivityReport(ctx *gin.Context) {
	service.GetActivityReport(ctx)
	// service层已经处理了响应，这里不需要再返回JSON
}
