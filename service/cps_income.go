package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	top "report_api/common/topsdk-go"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func GetCpsIncome(c *gin.Context) {
	client, err := top.NewClient("35026206", "8e7c9f532cac60b5baf0445eb0a3ab0e",
		top.WithSession("50000001826x672mspeYfgw6rWYjhitm4irSBE13493b04wFxuBJvvrus1enPitWidLM"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create client"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	params := top.Parameters{}

	listPath := []string{
		"alibaba_idle_affiliate_cps_income_details_query_response",
		"result",
	}

	// 从请求链接中获取参数
	billState, _ := strconv.Atoi(c.Query("billState")) // 获取 billState 参数
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))   // 获取 pageSize 参数
	incomes, err := client.QueryAllPagedIncomes(ctx, "alibaba.idle.affiliate.cps.income.details.query", params, listPath, billState, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Incomes"})
		return
	}

	if len(incomes) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No Incomes found"})
		return
	}

	// 导出到 Excel
	filePath := "incoms.xlsx"
	err = exportIncomeToExcel(incomes, filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export orders to Excel"})
		return
	}

	// 提供文件下载
	c.File(filePath)

	// 删除临时文件
	defer os.Remove(filePath)
}

func exportIncomeToExcel(incomes []*top.IncomeDetailDTO, filePath string) error {
	f := excelize.NewFile()
	sheetName := "Incomes"
	f.NewSheet(sheetName)

	// 写入表头
	headers := []string{
		"Bill ID",  // 对应 income.BillID
		"佣金状态描述",   // 对应 income.BillInfo.StateText
		"结算佣金",     // 对应 income.BillInfo.SettleAmountText
		"预估佣金",     // 对应 income.BillInfo.AssessAmountText
		"二级推广者id",  // 对应 income.BillInfo.SubPublisherID
		"状态",       // 0待发放 1待到账 2已到账 3已失效
		"记账时间",     // 对应 income.BillInfo.AccountingTime
		"销账时间",     // 对应 income.BillInfo.AccountingWriteOffTime
		"商品图",      // 对应 income.TradeOrderDTO.ItemPicURL
		"商品id",     // 对应 income.TradeOrderDTO.ItemID
		"付款时间",     // 对应 income.TradeOrderDTO.PayTime
		"商品标题",     // 对应 income.TradeOrderDTO.ItemTitle
		"订单id",     // 对应 income.TradeOrderDTO.OrderID
		"订单状态描述",   // 对应 income.TradeOrderDTO.StateDesc
		"确收金额",     // 对应 income.TradeOrderDTO.PartConfirmFee
		"实付金额",     // 对应 income.TradeOrderDTO.ActualPaidFee
		"创建时间",     // 对应 income.TradeOrderDTO.GmtCreate
		"订单状态",     // 对应 income.TradeOrderDTO.StateCode
		"订单结束时间",   // 对应 income.TradeOrderDTO.EndTime
		"未加密订单号",   // 对应 income.TradeOrderDTO.PlainOrderId
		"deeplink", // 对应 income.Deeplink
	}
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(sheetName, cell, header)
	}

	// 写入数据
	for rowIndex, income := range incomes {
		row := rowIndex + 2 // 从第二行开始写数据
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), income.BillID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), income.BillInfo.StateText)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), income.BillInfo.SettleAmountText)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), income.BillInfo.AssessAmountText)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), income.BillInfo.SubPublisherID)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), income.BillInfo.State)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), income.BillInfo.AccountingTime)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), income.TradeOrderDTO.ItemPicURL)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), income.TradeOrderDTO.ItemID)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), income.TradeOrderDTO.PayTime)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), income.TradeOrderDTO.ItemTitle)
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), income.TradeOrderDTO.OrderID)
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), income.TradeOrderDTO.StateDesc)
		f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), income.TradeOrderDTO.PartConfirmFee)
		f.SetCellValue(sheetName, fmt.Sprintf("P%d", row), income.TradeOrderDTO.ActualPaidFee)
		f.SetCellValue(sheetName, fmt.Sprintf("Q%d", row), income.TradeOrderDTO.GmtCreate)
		f.SetCellValue(sheetName, fmt.Sprintf("R%d", row), income.PlainBillId)
		f.SetCellValue(sheetName, fmt.Sprintf("S%d", row), income.Deeplink)
	}

	// 保存文件
	if err := f.SaveAs(filePath); err != nil {
		return err
	}
	return nil
}
