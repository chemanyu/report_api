package service

import (
	"crypto/md5"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
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
	JD_APP_KEY      = "5c9b5dc2edb53e58b37d94027f27a6d7"
	JD_APP_SECRET   = "de67c89669df4ade93faf34aa8168a8e" // 请替换为您的JD应用密钥
	JD_ACCESS_TOKEN = ""                                 // 请替换为您的JD访问令牌
)

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
		OrderId          int64   `json:"orderId"`
		OrderTime        string  `json:"orderTime"`
		FinishTime       string  `json:"finishTime"`
		ModifyTime       string  `json:"modifyTime"`
		UnionId          int64   `json:"unionId"`
		SkuId            int64   `json:"skuId"`
		SkuName          string  `json:"skuName"`
		Price            float64 `json:"price"`
		FinalRate        float64 `json:"finalRate"`
		EstimateCosPrice float64 `json:"estimateCosPrice"`
		EstimateFee      float64 `json:"estimateFee"`
		ActualCosPrice   float64 `json:"actualCosPrice"`
		ActualFee        float64 `json:"actualFee"`
		ValidCode        int     `json:"validCode"`
		PositionId       int64   `json:"positionId"`
		Pid              string  `json:"pid"`
		Account          string  `json:"account"`
	} `json:"data"`
}

// JD奖励订单API响应结构体
type JdBonusOrderResponse struct {
	JdUnionOpenOrderBonusQueryResponse struct {
		Code        string `json:"code"`
		QueryResult string `json:"queryResult"` // JSON字符串，需要二次解析
	} `json:"jd_union_open_order_bonus_query_responce"` // 注意：京东API返回的是responce不是response
}

// JD奖励订单查询结果（用于解析queryResult字符串）
type JdBonusQueryResult struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestId string `json:"requestId"`
	Data      []struct {
		UnionId          int64   `json:"unionId"`          // 联盟ID
		BonusInvalidCode string  `json:"bonusInvalidCode"` // 无效状态码
		BonusInvalidText string  `json:"bonusInvalidText"` // 无效状态文案
		PayPrice         float64 `json:"payPrice"`         // 实际支付金额
		EstimateCosPrice float64 `json:"estimateCosPrice"` // 预估计费金额
		EstimateFee      float64 `json:"estimateFee"`      // 预估佣金
		ActualCosPrice   float64 `json:"actualCosPrice"`   // 实际计费金额
		ActualFee        float64 `json:"actualFee"`        // 实际佣金
		OrderTime        int64   `json:"orderTime"`        // 下单时间(时间戳)
		FinishTime       int64   `json:"finishTime"`       // 完成时间(时间戳)
		PositionId       int64   `json:"positionId"`       // 推广位ID
		OrderId          int64   `json:"orderId"`          // 订单号
		BonusState       int     `json:"bonusState"`       // 奖励状态 0:无效，1:有效
		BonusText        string  `json:"bonusText"`        // 奖励状态文案
		SkuName          string  `json:"skuName"`          // 商品名称
		CommissionRate   float64 `json:"commissionRate"`   // 佣金比例(%)
		SubUnionId       string  `json:"subUnionId"`       // 子联盟ID
		Pid              string  `json:"pid"`              // pid
		Ext1             string  `json:"ext1"`             // 推荐生成推广链接时传入的扩展字段
		UnionAlias       string  `json:"unionAlias"`       // 母站长标称
		SubSideRate      float64 `json:"subSideRate"`      // 分成比例(单位:%)
		SubsidyRate      float64 `json:"subsidyRate"`      // 补贴比例(单位:%)
		FinalRate        float64 `json:"finalRate"`        // 最终分佣比例(%) =分成比例+佣金比例
		ActivityName     string  `json:"activityName"`     // 活动名称
		ParentId         int64   `json:"parentId"`         // 主单的订单号
		SkuId            int64   `json:"skuId"`            // skuId
		EstimateBonusFee float64 `json:"estimateBonusFee"` // 预估奖励金额
		ActualBonusFee   float64 `json:"actualBonusFee"`   // 实际奖励金额
		OrderState       int     `json:"orderState"`       // 奖励订单状态 1:已完成，2:已付款，3:待付款
		OrderText        string  `json:"orderText"`        // 奖励订单状态文案
		SortValue        string  `json:"sortValue"`        // 排序值，分页查询时用
		ActivityId       int64   `json:"activityId"`       // 奖励活动ID
		ChannelId        int64   `json:"channelId"`        // 渠道关系ID
		ItemId           string  `json:"itemId"`           // 联盟商品ID
		Id               int64   `json:"id"`               // 奖励订单ID
		Account          string  `json:"account"`          // 账户ID
	} `json:"data"`
}

// JD订单数据结构用于Excel导出
type JdOrderData struct {
	OrderId          int64   `json:"orderId"`
	OrderTime        string  `json:"orderTime"`
	FinishTime       string  `json:"finishTime"`
	ModifyTime       string  `json:"modifyTime"`
	UnionId          int64   `json:"unionId"`
	SkuId            int64   `json:"skuId"`
	SkuName          string  `json:"skuName"`
	Price            float64 `json:"price"`
	FinalRate        float64 `json:"finalRate"`
	EstimateCosPrice float64 `json:"estimateCosPrice"`
	EstimateFee      float64 `json:"estimateFee"`
	ActualCosPrice   float64 `json:"actualCosPrice"`
	ActualFee        float64 `json:"actualFee"`
	ValidCode        int     `json:"validCode"`
	PositionId       int64   `json:"positionId"`
	Pid              string  `json:"pid"`
	Account          string  `json:"account"`
}

// JD奖励订单数据结构用于Excel导出
type JdBonusOrderData struct {
	UnionId          int64   `json:"unionId"`
	BonusInvalidCode string  `json:"bonusInvalidCode"`
	BonusInvalidText string  `json:"bonusInvalidText"`
	PayPrice         float64 `json:"payPrice"`
	EstimateCosPrice float64 `json:"estimateCosPrice"`
	EstimateFee      float64 `json:"estimateFee"`
	ActualCosPrice   float64 `json:"actualCosPrice"`
	ActualFee        float64 `json:"actualFee"`
	OrderTime        string  `json:"orderTime"`
	FinishTime       string  `json:"finishTime"`
	PositionId       int64   `json:"positionId"`
	OrderId          int64   `json:"orderId"`
	BonusState       int     `json:"bonusState"`
	BonusText        string  `json:"bonusText"`
	SkuName          string  `json:"skuName"`
	CommissionRate   float64 `json:"commissionRate"`
	SubUnionId       string  `json:"subUnionId"`
	Pid              string  `json:"pid"`
	Ext1             string  `json:"ext1"`
	UnionAlias       string  `json:"unionAlias"`
	SubSideRate      float64 `json:"subSideRate"`
	SubsidyRate      float64 `json:"subsidyRate"`
	FinalRate        float64 `json:"finalRate"`
	ActivityName     string  `json:"activityName"`
	ParentId         int64   `json:"parentId"`
	SkuId            int64   `json:"skuId"`
	EstimateBonusFee float64 `json:"estimateBonusFee"`
	ActualBonusFee   float64 `json:"actualBonusFee"`
	OrderState       int     `json:"orderState"`
	OrderText        string  `json:"orderText"`
	SortValue        string  `json:"sortValue"`
	ActivityId       int64   `json:"activityId"`
	ChannelId        int64   `json:"channelId"`
	ItemId           string  `json:"itemId"`
	Id               int64   `json:"id"`
	Account          string  `json:"account"`
}

func GetJdOrder(ctx *gin.Context) {
	// 获取查询参数
	startTimeStr := ctx.Query("startTime")
	endTimeStr := ctx.Query("endTime")
	pageSize := ctx.DefaultQuery("pageSize", "200")
	orderType := ctx.Query("type") // 1-3

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
		//for orderType := 1; orderType <= 3; orderType++ {
		size, _ := strconv.Atoi(pageSize)
		typeInt, _ := strconv.Atoi(orderType)
		orders, err := fetchJdOrders(currentStartStr, currentEndStr, size, typeInt)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch orders for type %d, time %s-%s: %v", 1, currentStartStr, currentEndStr, err)})
			return
		}
		allOrders = append(allOrders, orders...)

		// 在不同类型的API调用之间添加延迟
		// if orderType < 3 {
		// 	time.Sleep(1 * time.Second)
		// }
		//}

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
func fetchJdOrders(startTime, endTime string, pageSize, orderType int) ([]JdOrderData, error) {
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
func fetchJdOrdersPage(startTime, endTime string, pageSize, pageIndex, orderType int) ([]JdOrderData, bool, error) {
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
		//fmt.Println("Response Content:", string(content))
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
		order := JdOrderData{
			OrderId:          item.OrderId,
			OrderTime:        item.OrderTime,
			FinishTime:       item.FinishTime,
			ModifyTime:       item.ModifyTime,
			UnionId:          item.UnionId,
			SkuId:            item.SkuId,
			SkuName:          item.SkuName,
			Price:            item.Price,
			FinalRate:        item.FinalRate,
			EstimateCosPrice: item.EstimateCosPrice,
			EstimateFee:      item.EstimateFee,
			ActualCosPrice:   item.ActualCosPrice,
			ActualFee:        item.ActualFee,
			ValidCode:        item.ValidCode,
			PositionId:       item.PositionId,
			Pid:              item.Pid,
			Account:          item.Account,
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
	//fmt.Println("String to Sign:", finalStr)

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
		"订单ID", "下单时间", "完成时间", "修改时间", "联盟ID", "商品ID", "商品名称",
		"商品价格", "佣金比例", "预估计费金额", "预估佣金", "实际计费金额", "实际佣金",
		"有效码", "推广位ID", "推广位", "账户信息",
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
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), order.UnionId)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), order.SkuId)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), order.SkuName)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), order.Price)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), order.FinalRate)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), order.EstimateCosPrice)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), order.EstimateFee)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), order.ActualCosPrice)
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), order.ActualFee)
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), order.ValidCode)
		f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), order.PositionId)
		f.SetCellValue(sheetName, fmt.Sprintf("P%d", row), order.Pid)
		f.SetCellValue(sheetName, fmt.Sprintf("Q%d", row), order.Account)
	}

	// 删除默认的Sheet1
	f.DeleteSheet("Sheet1")

	// 保存文件
	if err := f.SaveAs(filePath); err != nil {
		return err
	}
	return nil
}

// GetJdBonusOrder 获取JD奖励订单
func GetJdBonusOrder(ctx *gin.Context) {
	// 获取查询参数
	optType := ctx.DefaultQuery("optType", "1")     // 时间类型，1:下单时间，2:更新时间
	startTimeStr := ctx.Query("startTime")          // 开始时间，支持时间戳(毫秒)或日期格式
	endTimeStr := ctx.Query("endTime")              // 结束时间，支持时间戳(毫秒)或日期格式
	pageSize := ctx.DefaultQuery("pageSize", "100") // 每页数量，上限100
	activityId := ctx.Query("activityId")           // 奖励活动ID，可选

	// 验证必填参数
	if startTimeStr == "" || endTimeStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "startTime and endTime are required"})
		return
	}

	// 解析开始时间（支持时间戳或日期格式）
	var startTime int64
	var err error
	if len(startTimeStr) <= 13 && strings.Contains(startTimeStr, "0") {
		// 时间戳格式
		startTime, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid startTime format"})
			return
		}
	} else {
		// 日期格式
		t, err := time.Parse("2006-01-02 15:04:05", startTimeStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid startTime format, expected: 2006-01-02 15:04:05 or timestamp"})
			return
		}
		startTime = t.UnixNano() / 1e6 // 转换为毫秒
	}

	// 解析结束时间（支持时间戳或日期格式）
	var endTime int64
	if len(endTimeStr) <= 13 && strings.Contains(endTimeStr, "0") {
		// 时间戳格式
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid endTime format"})
			return
		}
	} else {
		// 日期格式
		t, err := time.Parse("2006-01-02 15:04:05", endTimeStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid endTime format, expected: 2006-01-02 15:04:05 or timestamp"})
			return
		}
		endTime = t.UnixNano() / 1e6 // 转换为毫秒
	}

	pageSizeInt, _ := strconv.Atoi(pageSize)
	optTypeInt, _ := strconv.Atoi(optType)

	var allOrders []JdBonusOrderData

	// 按600秒（10分钟）间隔循环时间段（京东API限制时间范围不超过600秒）
	currentTime := startTime
	for currentTime < endTime {
		// 计算当前区间的结束时间（最多600秒）
		intervalEndTime := currentTime + 600000 // 600秒 = 600000毫秒
		if intervalEndTime > endTime {
			intervalEndTime = endTime
		}

		//fmt.Printf("Fetching bonus orders for time range: %d - %d\n", currentTime, intervalEndTime)

		// 获取当前时间区间的奖励订单数据
		orders, err := fetchJdBonusOrdersWithPagination(optTypeInt, currentTime, intervalEndTime, pageSizeInt, activityId)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to fetch bonus orders for time %d-%d: %v", currentTime, intervalEndTime, err),
			})
			return
		}
		allOrders = append(allOrders, orders...)

		// 移动到下一个时间区间
		currentTime = intervalEndTime

		// 在不同时间区间的API调用之间添加延迟，避免请求过快
		// if currentTime < endTime {
		// 	time.Sleep(500 * time.Millisecond)
		// }
	}

	if len(allOrders) == 0 {
		ctx.JSON(http.StatusOK, gin.H{"message": "No bonus orders found", "count": 0})
		return
	}

	// 导出到Excel
	filePath := "jd_bonus_orders.xlsx"
	err = exportJdBonusOrdersToExcel(allOrders, filePath)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export bonus orders to Excel"})
		return
	}

	// 返回文件下载
	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", "attachment; filename=jd_bonus_orders.xlsx")
	ctx.File(filePath)
}

// fetchJdBonusOrdersWithPagination 获取JD奖励订单数据（带分页）
func fetchJdBonusOrdersWithPagination(optType int, startTime, endTime int64, pageSize int, activityId string) ([]JdBonusOrderData, error) {
	var allOrders []JdBonusOrderData
	pageNo := 1
	sortValue := ""

	// 循环获取所有分页数据
	for {
		orders, nextSortValue, err := fetchJdBonusOrdersPage(optType, startTime, endTime, pageNo, pageSize, sortValue, activityId)
		if err != nil {
			return nil, err
		}

		allOrders = append(allOrders, orders...)

		// 如果没有返回sortValue或没有数据，说明已经是最后一页
		if nextSortValue == "" || len(orders) == 0 {
			break
		}

		// 使用返回的sortValue进行下一页查询
		sortValue = nextSortValue
		pageNo++

		// 防止无限循环，最多获取100页
		if pageNo > 100 {
			break
		}

		// 在分页请求之间添加短暂延迟
		//time.Sleep(300 * time.Millisecond)
	}

	return allOrders, nil
}

// fetchJdBonusOrdersPage 获取单页JD奖励订单数据
func fetchJdBonusOrdersPage(optType int, startTime, endTime int64, pageNo, pageSize int, sortValue, activityId string) ([]JdBonusOrderData, string, error) {
	// JD API 参数
	method := "jd.union.open.order.bonus.query"
	version := "1.0"
	timestamp := time.Now().Format("2006-01-02 15:04:05.000-0700")
	log.Println("start,end:", startTime, endTime)
	// 构建请求参数
	orderReq := map[string]interface{}{
		"orderReq": map[string]interface{}{
			"optType":   optType,
			"startTime": startTime,
			"endTime":   endTime,
			"pageNo":    pageNo,
			"pageSize":  pageSize,
		},
	}

	// 添加可选参数
	if sortValue != "" {
		orderReq["orderReq"].(map[string]interface{})["sortValue"] = sortValue
	}
	if activityId != "" {
		activityIdInt, _ := strconv.ParseInt(activityId, 10, 64)
		orderReq["orderReq"].(map[string]interface{})["activityId"] = activityIdInt
	}

	paramJsonBytes, _ := jsoniter.Marshal(orderReq)
	paramJson := string(paramJsonBytes)

	// 构建签名参数
	params := map[string]string{
		"app_key":           JD_APP_KEY,
		"method":            method,
		"timestamp":         timestamp,
		"format":            "json",
		"v":                 version,
		"sign_method":       "md5",
		"360buy_param_json": paramJson,
	}

	// 如果有access_token则添加
	if JD_ACCESS_TOKEN != "" {
		params["access_token"] = JD_ACCESS_TOKEN
	}

	// 生成签名
	sign := generateJdSign(params, JD_APP_SECRET)

	// 构建请求URL
	apiUrl := "https://api.jd.com/routerjson"
	values := url.Values{}
	values.Set("app_key", JD_APP_KEY)
	values.Set("method", method)
	values.Set("timestamp", timestamp)
	values.Set("format", "json")
	values.Set("v", version)
	values.Set("sign_method", "md5")
	values.Set("sign", sign)
	values.Set("360buy_param_json", paramJson)

	if JD_ACCESS_TOKEN != "" {
		values.Set("access_token", JD_ACCESS_TOKEN)
	}

	requestUrl := apiUrl + "?" + values.Encode()

	//fmt.Println("Bonus Order Request URL:", requestUrl)

	// 发送HTTP请求
	var response JdBonusOrderResponse
	err := jdHttpGet(requestUrl, func(content []byte) error {
		//:", string(content))
		return jsoniter.Unmarshal(content, &response)
	})

	if err != nil {
		return nil, "", err
	}

	//log.Println("response.JdUnionOpenOrderBonusQueryResponse.Code: ", response)
	// 检查响应状态
	if response.JdUnionOpenOrderBonusQueryResponse.Code != "0" {
		return nil, "", fmt.Errorf("JD Bonus API error: code=%s", response.JdUnionOpenOrderBonusQueryResponse.Code)
	}

	// 二次解析queryResult字符串
	var queryResult JdBonusQueryResult
	err = jsoniter.Unmarshal([]byte(response.JdUnionOpenOrderBonusQueryResponse.QueryResult), &queryResult)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse bonus queryResult: %w", err)
	}

	if queryResult.Code != 200 {
		return nil, "", fmt.Errorf("JD Bonus API query error: code=%d, message=%s",
			queryResult.Code,
			queryResult.Message)
	}

	// 转换数据格式
	var orders []JdBonusOrderData
	var nextSortValue string
	for _, item := range queryResult.Data {
		// 将时间戳转换为日期时间字符串
		orderTime := ""
		finishTime := ""
		if item.OrderTime > 0 {
			orderTime = time.Unix(item.OrderTime/1000, 0).Format("2006-01-02 15:04:05")
		}
		if item.FinishTime > 0 {
			finishTime = time.Unix(item.FinishTime/1000, 0).Format("2006-01-02 15:04:05")
		}

		order := JdBonusOrderData{
			UnionId:          item.UnionId,
			BonusInvalidCode: item.BonusInvalidCode,
			BonusInvalidText: item.BonusInvalidText,
			PayPrice:         item.PayPrice,
			EstimateCosPrice: item.EstimateCosPrice,
			EstimateFee:      item.EstimateFee,
			ActualCosPrice:   item.ActualCosPrice,
			ActualFee:        item.ActualFee,
			OrderTime:        orderTime,
			FinishTime:       finishTime,
			PositionId:       item.PositionId,
			OrderId:          item.OrderId,
			BonusState:       item.BonusState,
			BonusText:        item.BonusText,
			SkuName:          item.SkuName,
			CommissionRate:   item.CommissionRate,
			SubUnionId:       item.SubUnionId,
			Pid:              item.Pid,
			Ext1:             item.Ext1,
			UnionAlias:       item.UnionAlias,
			SubSideRate:      item.SubSideRate,
			SubsidyRate:      item.SubsidyRate,
			FinalRate:        item.FinalRate,
			ActivityName:     item.ActivityName,
			ParentId:         item.ParentId,
			SkuId:            item.SkuId,
			EstimateBonusFee: item.EstimateBonusFee,
			ActualBonusFee:   item.ActualBonusFee,
			OrderState:       item.OrderState,
			OrderText:        item.OrderText,
			SortValue:        item.SortValue,
			ActivityId:       item.ActivityId,
			ChannelId:        item.ChannelId,
			ItemId:           item.ItemId,
			Id:               item.Id,
			Account:          item.Account,
		}
		orders = append(orders, order)

		// 保存最后一条记录的sortValue用于下一页查询
		if item.SortValue != "" {
			nextSortValue = item.SortValue
		}
	}

	return orders, nextSortValue, nil
}

// exportJdBonusOrdersToExcel 导出JD奖励订单到Excel
func exportJdBonusOrdersToExcel(orders []JdBonusOrderData, filePath string) error {
	f := excelize.NewFile()
	sheetName := "JD Bonus Orders"
	f.NewSheet(sheetName)

	// 写入表头
	headers := []string{
		"联盟ID", "无效状态码", "无效状态文案", "实际支付金额", "预估计费金额", "预估佣金",
		"实际计费金额", "实际佣金", "下单时间", "完成时间", "推广位ID", "订单号",
		"奖励状态", "奖励状态文案", "商品名称", "佣金比例(%)", "子联盟ID", "PID",
		"扩展字段", "母站长标称", "分成比例(%)", "补贴比例(%)", "最终分佣比例(%)",
		"活动名称", "主单订单号", "SkuID", "预估奖励金额", "实际奖励金额",
		"订单状态", "订单状态文案", "排序值", "奖励活动ID", "渠道关系ID",
		"联盟商品ID", "奖励订单ID", "账户ID",
	}

	for i, header := range headers {
		col := string(rune('A' + i))
		if i >= 26 {
			col = string(rune('A'+(i/26-1))) + string(rune('A'+(i%26)))
		}
		cell := fmt.Sprintf("%s1", col)
		f.SetCellValue(sheetName, cell, header)
	}

	// 写入数据
	for rowIndex, order := range orders {
		row := rowIndex + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), order.UnionId)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), order.BonusInvalidCode)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), order.BonusInvalidText)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), order.PayPrice)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), order.EstimateCosPrice)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), order.EstimateFee)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), order.ActualCosPrice)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), order.ActualFee)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), order.OrderTime)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), order.FinishTime)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), order.PositionId)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), order.OrderId)
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), order.BonusState)
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), order.BonusText)
		f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), order.SkuName)
		f.SetCellValue(sheetName, fmt.Sprintf("P%d", row), order.CommissionRate)
		f.SetCellValue(sheetName, fmt.Sprintf("Q%d", row), order.SubUnionId)
		f.SetCellValue(sheetName, fmt.Sprintf("R%d", row), order.Pid)
		f.SetCellValue(sheetName, fmt.Sprintf("S%d", row), order.Ext1)
		f.SetCellValue(sheetName, fmt.Sprintf("T%d", row), order.UnionAlias)
		f.SetCellValue(sheetName, fmt.Sprintf("U%d", row), order.SubSideRate)
		f.SetCellValue(sheetName, fmt.Sprintf("V%d", row), order.SubsidyRate)
		f.SetCellValue(sheetName, fmt.Sprintf("W%d", row), order.FinalRate)
		f.SetCellValue(sheetName, fmt.Sprintf("X%d", row), order.ActivityName)
		f.SetCellValue(sheetName, fmt.Sprintf("Y%d", row), order.ParentId)
		f.SetCellValue(sheetName, fmt.Sprintf("Z%d", row), order.SkuId)
		f.SetCellValue(sheetName, fmt.Sprintf("AA%d", row), order.EstimateBonusFee)
		f.SetCellValue(sheetName, fmt.Sprintf("AB%d", row), order.ActualBonusFee)
		f.SetCellValue(sheetName, fmt.Sprintf("AC%d", row), order.OrderState)
		f.SetCellValue(sheetName, fmt.Sprintf("AD%d", row), order.OrderText)
		f.SetCellValue(sheetName, fmt.Sprintf("AE%d", row), order.SortValue)
		f.SetCellValue(sheetName, fmt.Sprintf("AF%d", row), order.ActivityId)
		f.SetCellValue(sheetName, fmt.Sprintf("AG%d", row), order.ChannelId)
		f.SetCellValue(sheetName, fmt.Sprintf("AH%d", row), order.ItemId)
		f.SetCellValue(sheetName, fmt.Sprintf("AI%d", row), order.Id)
		f.SetCellValue(sheetName, fmt.Sprintf("AJ%d", row), order.Account)
	}

	// 删除默认的Sheet1
	f.DeleteSheet("Sheet1")

	// 保存文件
	if err := f.SaveAs(filePath); err != nil {
		return err
	}
	return nil
}
