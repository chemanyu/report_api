package service

import (
	"crypto/md5"
	"fmt"
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

const (
	JD_SH_APP_KEY    = "5c0a2824512572fba5881e3f3014989c"
	JD_SH_APP_SECRET = "0458ca722df540ac9f4c20bae9c9f848" // 请替换为您的JD应用密钥
)

// JD推广商品API请求参数
type JdShPromotionRequest struct {
	Account    string `json:"account" form:"account"`       // 已授权的媒体侧账户ID，必填
	TaskId     int64  `json:"taskId" form:"taskId"`         // 深海任务id（联盟侧获取）
	MaterialId string `json:"materialId" form:"materialId"` // 推广物料url，必填
	SceneId    int    `json:"sceneId" form:"sceneId"`       // 场景ID，必填
	Ext1       string `json:"ext1" form:"ext1"`             // 系统扩展参数，可选
	Pid        string `json:"pid" form:"pid"`               // 联盟子推客身份标识，可选
	CouponUrl  string `json:"couponUrl" form:"couponUrl"`   // 惠券领取链接，可选
	GiftKey    string `json:"giftKey" form:"giftKey"`       // 礼金批次号，可选
	ChannelId  int    `json:"channelId" form:"channelId"`   // 渠道关系ID，可选
}

// JD推广商品API响应结构
type JdShPromotionResponse struct {
	JdUnionOpenShPromotionGetResponse struct {
		Code      string `json:"code"`
		GetResult string `json:"getResult"` // JSON字符串，需要二次解析
	} `json:"jd_union_open_sh_promotion_get_responce"` // 注意：京东API的字段名拼写错误，是responce不是response
}

// JD推广商品查询结果
type JdShPromotionResult struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		ClickUrl             string `json:"clickUrl"`             // 推广链接
		ImpressionMonitorUrl string `json:"impressionMonitorUrl"` // 曝光监测链接
		ClickMonitorUrl      string `json:"clickMonitorUrl"`      // 点击监测链接
		AppUrl               string `json:"appUrl"`               // app呼起链接
	} `json:"data"`
}

// GetJdShPromotion 获取京东深海推广链接
func GetJdShPromotion(ctx *gin.Context) {
	var req JdShPromotionRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid parameters: %v", err)})
		return
	}

	// 参数校验
	if req.Account == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "account is required"})
		return
	}
	if req.MaterialId == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "materialId is required"})
		return
	}
	if req.SceneId == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "sceneId is required"})
		return
	}

	// 调用JD API
	result, err := fetchJdShPromotion(req, JD_SH_APP_KEY, JD_SH_APP_SECRET)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get promotion: %v", err)})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

// fetchJdShPromotion 调用京东深海推广API
func fetchJdShPromotion(req JdShPromotionRequest, appKey, secript string) (*JdShPromotionResult, error) {
	// JD API 参数
	method := "jd.union.open.sh.promotion.get"
	version := "1.0"
	timestamp := time.Now().Format("2006-01-02 15:04:05.000-0700")

	// 构建业务参数
	promotionReq := map[string]interface{}{
		"getCodeByTaskIdReq": map[string]interface{}{
			"account":    req.Account,
			"materialId": req.MaterialId,
			"taskId":     req.TaskId,
			"sceneId":    req.SceneId,
		},
	}

	// 添加可选参数
	innerReq := promotionReq["getCodeByTaskIdReq"].(map[string]interface{})
	if req.Ext1 != "" {
		innerReq["ext1"] = req.Ext1
	}
	if req.Pid != "" {
		innerReq["pid"] = req.Pid
	}
	if req.CouponUrl != "" {
		innerReq["couponUrl"] = req.CouponUrl
	}
	if req.GiftKey != "" {
		innerReq["giftCouponKey"] = req.GiftKey
	}
	if req.ChannelId != 0 {
		innerReq["channelId"] = req.ChannelId
	}

	paramJsonBytes, _ := jsoniter.Marshal(promotionReq)
	paramJson := string(paramJsonBytes)

	app_key := JD_SH_APP_KEY
	if appKey != "" {
		app_key = appKey
	}
	secret := JD_SH_APP_SECRET
	if secript != "" {
		secret = secript
	}
	// 构建签名参数
	params := map[string]string{
		"access_token":      "",
		"app_key":           app_key,
		"method":            method,
		"v":                 version,
		"timestamp":         timestamp,
		"360buy_param_json": paramJson,
	}

	// 生成签名
	sign := generateJdShSign(params, secret)

	// 构建请求URL
	apiUrl := "https://api.jd.com/routerjson"
	values := url.Values{}
	values.Set("access_token", "")
	values.Set("app_key", app_key)
	values.Set("method", method)
	values.Set("v", version)
	values.Set("sign", sign)
	values.Set("360buy_param_json", paramJson)
	values.Set("timestamp", timestamp)

	requestUrl := apiUrl + "?" + values.Encode()

	fmt.Println("Request URL:", requestUrl)

	// 发送HTTP请求
	var response JdShPromotionResponse
	err := jdHttpGet(requestUrl, func(content []byte) error {
		fmt.Println("Response Content:", string(content))
		return jsoniter.Unmarshal(content, &response)
	})

	if err != nil {
		return nil, err
	}

	// 检查响应状态
	if response.JdUnionOpenShPromotionGetResponse.Code != "0" {
		return nil, fmt.Errorf("JD API error: code=%s", response.JdUnionOpenShPromotionGetResponse.Code)
	}

	// 二次解析getResult字符串
	var result JdShPromotionResult
	err = jsoniter.Unmarshal([]byte(response.JdUnionOpenShPromotionGetResponse.GetResult), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse getResult: %w", err)
	}

	if result.Code != 200 {
		return nil, fmt.Errorf("JD API query error: code=%d, message=%s", result.Code, result.Message)
	}

	return &result, nil
}

// generateJdShSign 生成JD API签名（与jd_order.go中的generateJdSign逻辑相同）
func generateJdShSign(params map[string]string, appSecret string) string {
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

// BatchProcessJdShPromotion 批量处理京东深海推广（Excel上传）
func BatchProcessJdShPromotion(ctx *gin.Context) {
	// 获取sceneId参数
	appKey := ctx.PostForm("appKey")
	secretkey := ctx.PostForm("secretkey")
	taskId := ctx.PostForm("taskId")

	// 获取上传的文件
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
		return
	}

	// 验证文件扩展名
	if !strings.HasSuffix(file.Filename, ".xlsx") && !strings.HasSuffix(file.Filename, ".xls") {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "只支持 .xlsx 和 .xls 文件"})
		return
	}

	// 保存上传的文件
	uploadPath := fmt.Sprintf("uploaded_jd_sh_%s", file.Filename)
	if err := ctx.SaveUploadedFile(file, uploadPath); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败"})
		return
	}

	// 处理Excel并生成结果
	resultPath, err := processJdShExcel(uploadPath, 1, taskId, appKey, secretkey)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("处理失败: %v", err)})
		return
	}

	// 返回结果文件
	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=jd_sh_result_%s.xlsx", time.Now().Format("20060102150405")))
	ctx.File(resultPath)
}

// processJdShExcel 处理Excel文件并批量调用API
func processJdShExcel(filePath string, sceneId int, taskId, appKey, secretkey string) (string, error) {
	// 读取Excel文件
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return "", fmt.Errorf("打开Excel失败: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return "", fmt.Errorf("Excel文件中没有工作表")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return "", fmt.Errorf("读取行失败: %w", err)
	}

	if len(rows) < 2 {
		return "", fmt.Errorf("Excel文件没有数据行")
	}

	fmt.Printf("读取到 %d 行数据\n", len(rows))

	// 创建结果Excel
	resultFile := excelize.NewFile()
	resultSheet := "Sheet1"

	// 设置表头
	resultFile.SetCellValue(resultSheet, "A1", "任务ID")
	resultFile.SetCellValue(resultSheet, "B1", "平台户ID")
	resultFile.SetCellValue(resultSheet, "C1", "原始物料")
	resultFile.SetCellValue(resultSheet, "D1", "推广链接")
	resultFile.SetCellValue(resultSheet, "E1", "曝光监测地址")
	resultFile.SetCellValue(resultSheet, "F1", "点击监测地址")
	resultFile.SetCellValue(resultSheet, "G1", "app呼起链接")
	resultFile.SetCellValue(resultSheet, "H1", "status")

	// 处理每一行数据（跳过表头）
	rowIndex := 2
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 2 {
			continue
		}

		account := strings.TrimSpace(row[0])
		materialId := strings.TrimSpace(row[1])
		taskIdStr := taskId

		if account == "" || taskIdStr == "" || materialId == "" {
			continue
		}

		parsedTaskId, err := strconv.ParseInt(taskIdStr, 10, 64)
		if err != nil {
			// 记录错误但继续处理
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("A%d", rowIndex), taskIdStr)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("B%d", rowIndex), account)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("C%d", rowIndex), materialId)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("H%d", rowIndex), "taskId格式错误")
			rowIndex++
			continue
		}

		// 构建请求参数
		req := JdShPromotionRequest{
			Account:    account,
			MaterialId: materialId,
			TaskId:     parsedTaskId,
			SceneId:    sceneId,
		}

		// 调用API
		fmt.Println("req: ", req)
		result, err := fetchJdShPromotion(req, appKey, secretkey)
		if err != nil {
			// 记录错误
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("A%d", rowIndex), taskIdStr)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("B%d", rowIndex), account)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("C%d", rowIndex), materialId)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("H%d", rowIndex), fmt.Sprintf("错误: %v", err))
			rowIndex++
			continue
		}

		// 写入结果
		resultFile.SetCellValue(resultSheet, fmt.Sprintf("A%d", rowIndex), taskIdStr)
		resultFile.SetCellValue(resultSheet, fmt.Sprintf("B%d", rowIndex), account)
		resultFile.SetCellValue(resultSheet, fmt.Sprintf("C%d", rowIndex), materialId)
		resultFile.SetCellValue(resultSheet, fmt.Sprintf("D%d", rowIndex), result.Data.ClickUrl)
		resultFile.SetCellValue(resultSheet, fmt.Sprintf("E%d", rowIndex), result.Data.ImpressionMonitorUrl)
		resultFile.SetCellValue(resultSheet, fmt.Sprintf("F%d", rowIndex), result.Data.ClickMonitorUrl)
		resultFile.SetCellValue(resultSheet, fmt.Sprintf("G%d", rowIndex), result.Data.AppUrl)
		resultFile.SetCellValue(resultSheet, fmt.Sprintf("H%d", rowIndex), "成功")

		rowIndex++

		// 添加延迟，避免API频率限制
		time.Sleep(200 * time.Millisecond)
	}

	// 保存结果文件
	outputPath := fmt.Sprintf("jd_sh_result_%s.xlsx", time.Now().Format("20060102150405"))
	if err := resultFile.SaveAs(outputPath); err != nil {
		return "", fmt.Errorf("保存结果文件失败: %w", err)
	}

	return outputPath, nil
}
