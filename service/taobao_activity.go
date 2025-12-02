package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"report_api/core"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

const (
	TaobaoAPIURL = "http://gw.api.taobao.com/router/rest"
)

// TaobaoActivityRequest 淘宝活动报表请求参数
type TaobaoActivityRequest struct {
	EventId   string `json:"event_id" form:"event_id"`     // CPA活动id，必填
	BizDate   string `json:"biz_date" form:"biz_date"`     // 日期(yyyyMMdd)，必填
	PageNo    int    `json:"page_no" form:"page_no"`       // 分页页码，从1开始，默认1
	QueryType int    `json:"query_type" form:"query_type"` // 查询类型，1-推广 2-拉新，必填
	PageSize  int    `json:"page_size" form:"page_size"`   // 分页大小，默认10
	Pid       string `json:"pid" form:"pid"`               // 推广位id
}

// VegasCpaReportDTO 数据统计对象
type VegasCpaReportDTO struct {
	Union30dLxUv int    `json:"union_30d_lx_uv"` // 近30天拉新奖励计算用户量
	RewardAmount string `json:"reward_amount"`   // 奖励金额
	RelationId   int    `json:"relation_id"`     // 代理id
	BizDate      string `json:"biz_date"`        // 统计日期
	Pid          string `json:"pid"`             // 推广位id
	QueryType    int    `json:"query_type"`      // 查询类型：1推广 2拉新
	ExtInfo      string `json:"ext_info"`        // 活动扩展信息
}

// TaobaoActivityResponse 淘宝活动报表响应
type TaobaoActivityResponse struct {
	TbkDgCpaActivityReportResponse struct {
		RequestId string `json:"request_id"`
		Result    struct {
			Data struct {
				Results struct {
					VegasCpaReportDTO []VegasCpaReportDTO `json:"vegas_cpa_report_d_t_o"`
				} `json:"results"`
			} `json:"data"`
		} `json:"result"`
	} `json:"tbk_dg_cpa_activity_report_response"`
}

// ExtInfoParsed ext_info解析后的结构
type ExtInfoParsed struct {
	Crowd1RewardUv  string  `json:"crowd1_reward_uv"`
	Crowd2RewardUv  string  `json:"crowd2_reward_uv"`
	Crowd3RewardUv  string  `json:"crowd3_reward_uv"`
	Crowd4RewardUv  string  `json:"crowd4_reward_uv"`
	Crowd5RewardUv  string  `json:"crowd5_reward_uv"`
	AccountDrawRate float64 `json:"account_draw_rate"`
	DrawRate        float64 `json:"draw_rate"`
	UpdateTime      string  `json:"update_time"`
}

// GetActivityReport 获取淘宝活动报表（支持Excel批量查询）
func GetActivityReport(ctx *gin.Context) {
	// 获取URL参数
	bizDate := ctx.Query("biz_date")
	queryTypeStr := ctx.Query("query_type")

	// 参数校验
	if bizDate == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "biz_date is required"})
		return
	}

	queryType := 1 // 默认值
	if queryTypeStr != "" {
		if qt, err := strconv.Atoi(queryTypeStr); err == nil {
			queryType = qt
		}
	}

	// 获取上传的Excel文件
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Excel file is required"})
		return
	}

	// 验证文件扩展名
	if !strings.HasSuffix(file.Filename, ".xlsx") && !strings.HasSuffix(file.Filename, ".xls") {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Only .xlsx and .xls files are accepted"})
		return
	}

	// 保存上传的文件
	uploadPath := fmt.Sprintf("uploaded_taobao_%s", file.Filename)
	if err := ctx.SaveUploadedFile(file, uploadPath); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save file: %v", err)})
		return
	}

	// 处理Excel并批量查询
	resultFile, err := processExcelAndQuery(uploadPath, bizDate, queryType)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to process: %v", err)})
		return
	}

	// 返回Excel文件下载
	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=taobao_result_%s.xlsx", time.Now().Format("20060102150405")))
	ctx.File(resultFile)
}

// processExcelAndQuery 处理Excel文件并批量查询
func processExcelAndQuery(filePath, bizDate string, queryType int) (string, error) {
	// 读取Excel文件
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open excel: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return "", fmt.Errorf("no sheets found in excel file")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return "", fmt.Errorf("failed to read rows: %w", err)
	}

	if len(rows) < 2 {
		return "", fmt.Errorf("excel file has no data rows")
	}

	// 查找pid列索引
	header := rows[0]
	pidCol := -1
	for i, col := range header {
		if strings.TrimSpace(col) == "pid" {
			pidCol = i
			break
		}
	}

	if pidCol == -1 {
		return "", fmt.Errorf("pid column not found in excel")
	}

	// 创建结果Excel
	resultFile := excelize.NewFile()
	resultSheet := "Sheet1"

	// 设置表头
	resultFile.SetCellValue(resultSheet, "A1", "pid")
	resultFile.SetCellValue(resultSheet, "B1", "biz_date")
	resultFile.SetCellValue(resultSheet, "C1", "符合奖励要求的累计用户数")
	resultFile.SetCellValue(resultSheet, "D1", "奖励金额")
	resultFile.SetCellValue(resultSheet, "E1", "人群1结算奖励uv")
	resultFile.SetCellValue(resultSheet, "F1", "人群2结算奖励uv")
	resultFile.SetCellValue(resultSheet, "G1", "人群3结算奖励uv")
	resultFile.SetCellValue(resultSheet, "H1", "人群4结算奖励uv")
	resultFile.SetCellValue(resultSheet, "I1", "人群5结算奖励uv")
	resultFile.SetCellValue(resultSheet, "J1", "账号总开奖率")
	resultFile.SetCellValue(resultSheet, "K1", "开奖率")
	resultFile.SetCellValue(resultSheet, "L1", "更新时间")

	// 遍历每一行，获取pid并查询
	rowIndex := 2
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) <= pidCol {
			continue
		}

		pid := strings.TrimSpace(row[pidCol])
		if pid == "" {
			continue
		}

		// 构建请求参数
		req := TaobaoActivityRequest{
			EventId:   "3654363", // 默认值
			BizDate:   bizDate,
			QueryType: queryType,
			PageNo:    1,
			PageSize:  10,
			Pid:       pid,
		}

		// 调用淘宝API
		result, err := callTaobaoActivityAPI(req)
		if err != nil {
			// 记录错误但继续处理
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("A%d", rowIndex), pid)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("B%d", rowIndex), bizDate)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("C%d", rowIndex), fmt.Sprintf("Error: %v", err))
			rowIndex++
			continue
		}

		// 提取数据
		if len(result.TbkDgCpaActivityReportResponse.Result.Data.Results.VegasCpaReportDTO) > 0 {
			data := result.TbkDgCpaActivityReportResponse.Result.Data.Results.VegasCpaReportDTO[0]

			// 解析ext_info
			extInfo := parseExtInfo(data.ExtInfo)

			// 填充Excel行
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("A%d", rowIndex), data.Pid)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("B%d", rowIndex), data.BizDate)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("C%d", rowIndex), data.Union30dLxUv)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("D%d", rowIndex), data.RewardAmount)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("E%d", rowIndex), extInfo.Crowd1RewardUv)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("F%d", rowIndex), extInfo.Crowd2RewardUv)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("G%d", rowIndex), extInfo.Crowd3RewardUv)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("H%d", rowIndex), extInfo.Crowd4RewardUv)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("I%d", rowIndex), extInfo.Crowd5RewardUv)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("J%d", rowIndex), extInfo.AccountDrawRate)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("K%d", rowIndex), extInfo.DrawRate)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("L%d", rowIndex), extInfo.UpdateTime)
		} else {
			// 没有数据
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("A%d", rowIndex), pid)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("B%d", rowIndex), bizDate)
			resultFile.SetCellValue(resultSheet, fmt.Sprintf("C%d", rowIndex), "No data")
		}

		rowIndex++
	}

	// 保存结果文件
	outputPath := fmt.Sprintf("taobao_result_%s.xlsx", time.Now().Format("20060102150405"))
	if err := resultFile.SaveAs(outputPath); err != nil {
		return "", fmt.Errorf("failed to save result file: %w", err)
	}

	return outputPath, nil
}

// parseExtInfo 解析ext_info JSON字符串
func parseExtInfo(extInfoStr string) ExtInfoParsed {
	var result ExtInfoParsed

	if extInfoStr == "" {
		return result
	}

	var extData map[string]interface{}
	if err := json.Unmarshal([]byte(extInfoStr), &extData); err != nil {
		return result
	}

	// 提取各个字段
	if v, ok := extData["人群1结算奖励uv"].(string); ok {
		result.Crowd1RewardUv = v
	}
	if v, ok := extData["人群2结算奖励uv"].(string); ok {
		result.Crowd2RewardUv = v
	}
	if v, ok := extData["人群3结算奖励uv"].(string); ok {
		result.Crowd3RewardUv = v
	}
	if v, ok := extData["人群4结算奖励uv"].(string); ok {
		result.Crowd4RewardUv = v
	}
	if v, ok := extData["人群5结算奖励uv"].(string); ok {
		result.Crowd5RewardUv = v
	}
	if v, ok := extData["账号总开奖率（奖励计算用)"].(float64); ok {
		result.AccountDrawRate = v
	}
	if v, ok := extData["开奖率"].(float64); ok {
		result.DrawRate = v
	}
	if v, ok := extData["更新时间"].(string); ok {
		result.UpdateTime = v
	}

	return result
}

// callTaobaoActivityAPI 调用淘宝客活动报表API
func callTaobaoActivityAPI(req TaobaoActivityRequest) (*TaobaoActivityResponse, error) {
	// 从配置中获取AppKey和AppSecret
	config := core.GetConfig()
	appKey := config.TAOBAO_APP_KEY
	appSecret := config.TAOBAO_APP_SECRET

	// 构建公共参数
	params := make(map[string]string)
	params["method"] = "taobao.tbk.dg.cpa.activity.report"
	params["app_key"] = appKey
	params["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
	params["format"] = "json"
	params["v"] = "2.0"
	params["sign_method"] = "md5"
	params["partner_id"] = "top-apitools"

	// 构建业务参数
	params["event_id"] = req.EventId
	params["biz_date"] = req.BizDate
	params["query_type"] = fmt.Sprintf("%d", req.QueryType)
	params["page_no"] = fmt.Sprintf("%d", req.PageNo)
	params["page_size"] = fmt.Sprintf("%d", req.PageSize)
	if req.Pid != "" {
		params["pid"] = req.Pid
	}

	// 生成签名
	sign := generateSign(params, appSecret)
	params["sign"] = sign

	// 构建URL
	apiURL := buildURL(TaobaoAPIURL, params)

	// 发送HTTP请求
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 解析响应
	var result TaobaoActivityResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &result, nil
}

// generateSign 生成淘宝API签名（MD5算法）
// 签名步骤：
// 1. 对所有API请求参数（包括公共参数和业务参数，但除去sign参数和byte[]类型的参数），根据参数名称的ASCII码表的顺序排序
// 2. 将排序好的参数名和参数值拼接在一起，如：bar2foo1foo bar3foobar4
// 3. 把拼接好的字符串采用utf-8编码，使用签名算法对编码后的字节流进行摘要
// 4. 将摘要得到的字节流结果使用十六进制表示，如：hex("helloworld".getBytes("utf-8")) = "68656C6C6F776F726C64"
func generateSign(params map[string]string, appSecret string) string {
	// 1. 按照参数名ASCII码排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 拼接参数名和参数值
	var builder strings.Builder
	builder.WriteString(appSecret)
	for _, k := range keys {
		builder.WriteString(k)
		builder.WriteString(params[k])
	}
	builder.WriteString(appSecret)

	// 3. MD5摘要
	hash := md5.Sum([]byte(builder.String()))

	// 4. 转换为十六进制大写字符串
	return strings.ToUpper(hex.EncodeToString(hash[:]))
}

// buildURL 构建完整的请求URL
func buildURL(baseURL string, params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	return baseURL + "?" + values.Encode()
}
