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

func GetCpsUser(c *gin.Context) {
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
		"alibaba_idle_affiliate_user_action_log_query_response",
		"result",
	}

	// 从请求链接中获取参数
	queryUserDate := c.Query("query_user_date")      // 获取 billState 参数
	pageSize, _ := strconv.Atoi(c.Query("pageSize")) // 获取 pageSize 参数
	users, err := client.QueryAllPagedUsers(ctx, "alibaba.idle.affiliate.user.action.log.query", params, listPath, queryUserDate, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Users"})
		return
	}

	if len(users) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No Users found"})
		return
	}

	// 导出到 Excel
	filePath := "user.xlsx"
	err = exportUserToExcel(users, filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export users to Excel"})
		return
	}

	// 提供文件下载
	c.File(filePath)

	// 删除临时文件
	defer os.Remove(filePath)
}

func exportUserToExcel(incomes []*top.UserDetailDTO, filePath string) error {
	f := excelize.NewFile()
	sheetName := "Incomes"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}
	f.SetActiveSheet(index)
	fmt.Println(len(incomes))
	// 写入表头
	headers := []string{
		"UserId",     // 加密用户id
		"用户是否次留",     // 用户是否次留（用户在T日和T+1日均有访问行为，T为查询日期）
		"用户是否为卡券新用户", // 用户是否为卡券新用户
		"子推广者id",     // 子推广者id
	}
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(sheetName, cell, header)
	}

	// 写入数据
	for rowIndex, income := range incomes {
		row := rowIndex + 2 // 从第二行开始写数据
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), income.UserId)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), income.DayOneUserRetention)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), income.CardCouponNewUser)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), income.SubPublisherId)
	}

	// 保存文件
	if err := f.SaveAs(filePath); err != nil {
		return err
	}
	return nil
}
