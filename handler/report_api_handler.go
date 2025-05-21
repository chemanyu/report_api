package handlers

import (
	"net/http"
	"report_api/service"

	"github.com/gin-gonic/gin"
)

type ReportApiHandler struct {
	CommonHandler
}

var GetReportApiHandler = new(ReportApiHandler)
var GetCpsOrderApiHandler = new(ReportApiHandler)
var GetCpsIncomeApiHandler = new(ReportApiHandler)
var GetCpsUserApiHandler = new(ReportApiHandler)

func init() {
	GetReportApiHandler.postMapping("report_api", getAppDetail)
	GetCpsOrderApiHandler.getMapping("xianyu_cps_order", getCpsOrder)    // 获取订单
	GetCpsIncomeApiHandler.getMapping("xianyu_cps_income", getCpsIncome) // 获取佣金
	GetCpsUserApiHandler.getMapping("xianyu_cps_user", getCpsUser)       // 获取佣金
}

func getAppDetail(ctx *gin.Context) {

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "report_api retrieved successfully",
	})
}

func getCpsOrder(ctx *gin.Context) {
	service.GetCpsOrder(ctx)
}

func getCpsIncome(ctx *gin.Context) {
	service.GetCpsIncome(ctx)
}

func getCpsUser(ctx *gin.Context) {
	service.GetCpsUser(ctx)
}
