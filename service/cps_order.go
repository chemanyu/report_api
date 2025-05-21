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

func GetCpsOrder(c *gin.Context) {
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
		"alibaba_idle_affiliate_cps_order_query_response",
		"result",
	}

	pageSize, _ := strconv.Atoi(c.Query("pageSize")) // 获取 pageSize 参数
	orders, err := client.QueryAllPagedOrders(ctx, "alibaba.idle.affiliate.cps.order.query", params, listPath, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}

	if len(orders) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No orders found"})
		return
	}

	// 导出到 Excel
	filePath := "orders.xlsx"
	err = exportOrdersToExcel(orders, filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export orders to Excel"})
		return
	}

	// 提供文件下载
	c.File(filePath)

	// 删除临时文件
	defer os.Remove(filePath)
}

func exportOrdersToExcel(orders []*top.TradeOrderDTO, filePath string) error {
	f := excelize.NewFile()
	sheetName := "Orders"
	f.NewSheet(sheetName)

	// 写入表头
	headers := []string{
		"退款金额", "商品图", "订单id", "商品标题", "付款时间",
		"订单状态描述", "确收金额", "实际付款金额", "创建时间",
		"商品id", "预估佣金", "预估费率",
		"订单状态码", "商品价格", "订单结束时间", "itemImageUrl", "子推广者id",
	}
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(sheetName, cell, header)
	}

	// 写入数据
	for rowIndex, order := range orders {
		row := rowIndex + 2 // 从第二行开始写数据
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), order.RefundFee)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), order.ItemPicURL)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), order.OrderID)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), order.ItemTitle)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), order.PayTime)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), order.StateDesc)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), order.PartConfirmFee)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), order.ActualPaidFee)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), order.GmtCreate)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), order.ItemID)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), order.EstimateCommission)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), order.EstimateCommissionRate)
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), order.StateCode)
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), order.ItemPrice)
		f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), order.EndTime)
		f.SetCellValue(sheetName, fmt.Sprintf("P%d", row), order.ItemImageURL)
		f.SetCellValue(sheetName, fmt.Sprintf("Q%d", row), order.SubPublisherID)
	}

	// 保存文件
	if err := f.SaveAs(filePath); err != nil {
		return err
	}
	return nil
}
