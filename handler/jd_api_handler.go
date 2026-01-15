package handlers

import (
	"report_api/service"

	"github.com/gin-gonic/gin"
)

type JdReportApiHandler struct {
	CommonHandler
}

var GetJdReportApiHandler = new(JdReportApiHandler)
var GetJdShApiHandler = new(JdReportApiHandler)

func init() {
	GetJdReportApiHandler.getMapping("jd_order", getJdOrder)            // 获取佣金
	GetJdReportApiHandler.getMapping("jd_bonus_order", getJdBonusOrder) // 获取奖励订单
	GetJdShApiHandler.getMapping("jd_sh", getJdSh)                      // 获取推广链接
	GetJdShApiHandler.postMapping("jd_sh_batch", batchProcessJdSh)      // 批量处理
}

func getJdOrder(ctx *gin.Context) {
	service.GetJdOrder(ctx)
	// service层已经处理了响应，这里不需要再返回JSON
}

func getJdBonusOrder(ctx *gin.Context) {
	service.GetJdBonusOrder(ctx)
	// service层已经处理了响应，这里不需要再返回JSON
}

func getJdSh(ctx *gin.Context) {
	service.GetJdShPromotion(ctx)
}

func batchProcessJdSh(ctx *gin.Context) {
	service.BatchProcessJdShPromotion(ctx)
}
