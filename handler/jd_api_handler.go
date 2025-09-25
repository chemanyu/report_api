package handlers

import (
	"report_api/service"

	"github.com/gin-gonic/gin"
)

type JdReportApiHandler struct {
	CommonHandler
}

var GetJdReportApiHandler = new(JdReportApiHandler)

func init() {
	GetJdReportApiHandler.getMapping("jd_order", getJdOrder) // 获取佣金
}

func getJdOrder(ctx *gin.Context) {
	service.GetJdOrder(ctx)
	// service层已经处理了响应，这里不需要再返回JSON
}
