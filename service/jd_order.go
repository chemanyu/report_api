package service

import (
	"crypto/md5"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/xuri/excelize/v2"
)

// 创建专用的JD API HTTP客户端，超时时间较长
func createJdHttpClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:      10,
		IdleConnTimeout:   30 * time.Second,
		DisableKeepAlives: false,
	}
	return &http.Client{
		Transport: tr,
		Timeout:   60 * time.Second, // JD API专用，60秒超时
	}
}

// JD API专用HTTP GET请求，带重试机制
func jdHttpGet(urlpath string, resp_exec func(content []byte) error) error {
	client := createJdHttpClient()

	req, err := http.NewRequest("GET", urlpath, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Request attempt failed, retrying in 2 seconds: %v\n", err)
		return fmt.Errorf("request failed: %w", err)
	}

	content, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		fmt.Printf("Request attempt failed, retrying in 2 seconds: %v\n", err)
		return fmt.Errorf("request failed: %w", err)
	}

	// 成功读取响应，执行回调
	return resp_exec(content)
}

// JD API 配置
const (
	JD_APP_KEY      = "634228adb0f410c705aaefe2e194cf2b"
	JD_APP_SECRET   = "ee62ec0fab8143f38b8cfa5453cfa2a1" // 请替换为您的JD应用密钥
	JD_ACCESS_TOKEN = ""                                 // 请替换为您的JD访问令牌
)

// 需要过滤的推广位ID列表
var targetPositionIds = map[int64]bool{
	3102289741: true,
	3102289660: true,
	3102289811: true,
	3102289782: true,
	3102289791: true,
}

// JD API 响应结构体
type JdOrderResponse struct {
	JdUnionOpenOrderRowQueryResponse struct {
		Code        string `json:"code"`
		QueryResult string `json:"queryResult"` // 注意：这是一个JSON字符串，需要二次解析
	} `json:"jd_union_open_order_row_query_responce"`
}

// JD API 查询结果结构体（用于解析queryResult字符串）
type JdQueryResult struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestId string `json:"requestId"`
	HasMore   bool   `json:"hasMore"`
	Data      []struct {
		OrderId     int64   `json:"orderId"`
		OrderTime   string  `json:"orderTime"`
		FinishTime  string  `json:"finishTime"`
		ModifyTime  string  `json:"modifyTime"`
		PositionId  int64   `json:"positionId"`
		Account     string  `json:"account"`
		SkuName     string  `json:"skuName"`
		ActualFee   float64 `json:"actualFee"`
		EstimateFee float64 `json:"estimateFee"`
		ValidCode   int     `json:"validCode"`
	} `json:"data"`
}

// JD订单数据结构用于Excel导出
type JdOrderData struct {
	OrderId    int64  `json:"orderId"`
	OrderTime  string `json:"orderTime"`
	FinishTime string `json:"finishTime"`
	ModifyTime string `json:"modifyTime"`
	PositionId int64  `json:"positionId"`
	Account    string `json:"account"`
}

func GetJdOrder(ctx *gin.Context) {
	// 获取查询参数
	startTimeStr := ctx.Query("startTime")
	endTimeStr := ctx.Query("endTime")
	pageSize := ctx.DefaultQuery("pageSize", "200")

	if startTimeStr == "" || endTimeStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "startTime and endTime are required"})
		return
	}
	// startTimeStr, _ = url.QueryUnescape(startTimeStr)
	// endTimeStr, _ = url.QueryUnescape(endTimeStr)

	// 处理特殊的24:00:00时间格式，将其转换为23:59:59
	endTimeStr = strings.Replace(endTimeStr, "24:00:00", "23:59:59", -1)

	// 解析时间
	startTime, err := time.Parse("2006-01-02 15:04:05", startTimeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid startTime format: %s, expected: 2006-01-02 15:04:05 (e.g., 2025-09-23 14:00:00)", startTimeStr),
		})
		return
	}

	endTime, err := time.Parse("2006-01-02 15:04:05", endTimeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid endTime format: %s, expected: 2006-01-02 15:04:05 (e.g., 2025-09-23 23:59:59). Note: use 23:59:59 instead of 24:00:00", endTimeStr),
		})
		return
	}

	var allOrders []JdOrderData

	// 按小时循环时间段
	currentTime := startTime
	for currentTime.Before(endTime) {
		// 计算当前小时的结束时间
		hourEndTime := currentTime.Add(time.Hour)
		if hourEndTime.After(endTime) {
			hourEndTime = endTime
		}

		currentStartStr := currentTime.Format("2006-01-02 15:04:05")
		currentEndStr := hourEndTime.Format("2006-01-02 15:04:05")

		// 循环调用API，type分别为1、2、3
		for orderType := 1; orderType <= 3; orderType++ {
			orders, err := fetchJdOrders(currentStartStr, currentEndStr, pageSize, orderType)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch orders for type %d, time %s-%s: %v", orderType, currentStartStr, currentEndStr, err)})
				return
			}
			allOrders = append(allOrders, orders...)

			// 在不同类型的API调用之间添加延迟
			if orderType < 3 {
				time.Sleep(1 * time.Second)
			}
		}

		// 移动到下一个小时
		currentTime = hourEndTime
	}

	if len(allOrders) == 0 {
		ctx.JSON(http.StatusOK, gin.H{"message": "No orders found", "count": 0})
		return
	}

	// 导出到Excel
	filePath := "jd_orders.xlsx"
	err = exportJdOrdersToExcel(allOrders, filePath)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export orders to Excel"})
		return
	}

	// 返回文件下载
	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", "attachment; filename=jd_orders.xlsx")
	ctx.File(filePath)
}

// 获取JD订单数据
func fetchJdOrders(startTime, endTime, pageSize string, orderType int) ([]JdOrderData, error) {
	var allOrders []JdOrderData
	pageIndex := 1
	hasMore := true

	for hasMore {
		orders, hasMoreData, err := fetchJdOrdersPage(startTime, endTime, pageSize, pageIndex, orderType)
		if err != nil {
			return nil, err
		}

		allOrders = append(allOrders, orders...)
		hasMore = hasMoreData
		pageIndex++

		// 防止无限循环，最多获取100页
		if pageIndex > 100 {
			break
		}

		// 在分页请求之间添加短暂延迟
		if hasMore {
			time.Sleep(500 * time.Millisecond)
		}
	}

	return allOrders, nil
}

// 获取单页JD订单数据
func fetchJdOrdersPage(startTime, endTime, pageSize string, pageIndex, orderType int) ([]JdOrderData, bool, error) {
	// JD API 参数
	method := "jd.union.open.order.row.query"
	version := "1.0"
	timestamp := time.Now().Format("2006-01-02 15:04:05.000-0700")

	// 构建请求参数
	orderReq := map[string]interface{}{
		"orderReq": map[string]interface{}{
			"pageIndex": pageIndex,
			"pageSize":  pageSize,
			"startTime": startTime,
			"endTime":   endTime,
			"type":      orderType,
		},
	}

	paramJsonBytes, _ := jsoniter.Marshal(orderReq)
	paramJson := string(paramJsonBytes)

	// 构建签名参数
	params := map[string]string{
		"access_token":      JD_ACCESS_TOKEN,
		"app_key":           JD_APP_KEY,
		"method":            method,
		"v":                 version,
		"timestamp":         timestamp,
		"360buy_param_json": paramJson,
	}

	// 生成签名
	sign := generateJdSign(params, JD_APP_SECRET)

	// 构建请求URL
	apiUrl := "https://api.jd.com/routerjson"
	values := url.Values{}
	values.Set("access_token", JD_ACCESS_TOKEN)
	values.Set("app_key", JD_APP_KEY)
	values.Set("method", method)
	values.Set("v", version)
	values.Set("sign", sign)
	values.Set("360buy_param_json", paramJson)
	values.Set("timestamp", timestamp)

	requestUrl := apiUrl + "?" + values.Encode()

	fmt.Println("Request URL:", requestUrl)

	// 发送HTTP请求
	var response JdOrderResponse
	err := jdHttpGet(requestUrl, func(content []byte) error {
		fmt.Println("Response Content:", string(content))
		return jsoniter.Unmarshal(content, &response)
	})

	if err != nil {
		return nil, false, err
	}

	// 检查响应状态
	if response.JdUnionOpenOrderRowQueryResponse.Code != "0" {
		return nil, false, fmt.Errorf("JD API error: code=%s", response.JdUnionOpenOrderRowQueryResponse.Code)
	}

	// 二次解析queryResult字符串
	var queryResult JdQueryResult
	err = jsoniter.Unmarshal([]byte(response.JdUnionOpenOrderRowQueryResponse.QueryResult), &queryResult)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse queryResult: %w", err)
	}

	if queryResult.Code != 200 {
		return nil, false, fmt.Errorf("JD API query error: code=%d, message=%s",
			queryResult.Code,
			queryResult.Message)
	}

	// 转换数据格式，只保留指定的PositionId
	var orders []JdOrderData
	for _, item := range queryResult.Data {
		// 只处理目标推广位ID的订单
		if !targetPositionIds[item.PositionId] {
			continue
		}

		order := JdOrderData{
			OrderId:    item.OrderId,
			OrderTime:  item.OrderTime,
			FinishTime: item.FinishTime,
			ModifyTime: item.ModifyTime,
			PositionId: item.PositionId,
			Account:    item.Account,
		}
		orders = append(orders, order)
	}

	return orders, queryResult.HasMore, nil
}

// 生成JD API签名
func generateJdSign(params map[string]string, appSecret string) string {
	// 1. 将所有请求参数按照字母先后顺序排列
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 把所有参数名和参数值进行拼接
	var signStr strings.Builder
	for _, k := range keys {
		signStr.WriteString(k)
		signStr.WriteString(params[k])
	}

	// 3. 把appSecret夹在字符串的两端
	finalStr := appSecret + signStr.String() + appSecret
	fmt.Println("String to Sign:", finalStr)

	// 4. 使用MD5进行加密，再转化成大写
	hash := md5.Sum([]byte(finalStr))
	return strings.ToUpper(fmt.Sprintf("%x", hash))
}

// 导出JD订单到Excel
func exportJdOrdersToExcel(orders []JdOrderData, filePath string) error {
	f := excelize.NewFile()
	sheetName := "JD Orders"
	f.NewSheet(sheetName)

	// 写入表头
	headers := []string{
		"订单ID", "下单时间", "完成时间", "修改时间", "推广位ID", "账户信息",
	}
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(sheetName, cell, header)
	}

	// 写入数据
	for rowIndex, order := range orders {
		row := rowIndex + 2 // 从第二行开始写数据
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), order.OrderId)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), order.OrderTime)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), order.FinishTime)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), order.ModifyTime)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), order.PositionId)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), order.Account)
	}

	// 删除默认的Sheet1
	f.DeleteSheet("Sheet1")

	// 保存文件
	if err := f.SaveAs(filePath); err != nil {
		return err
	}
	return nil
}
