package service

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// UploadExcelFiles 处理两个Excel文件上传
func UploadGdtExcelFiles(ctx *gin.Context) {
	// 获取上传的两个文件
	form, err := ctx.MultipartForm()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to parse multipart form: %v", err)})
		return
	}

	files := form.File["files"] // 期望客户端使用 "files" 作为字段名上传多个文件
	if len(files) != 2 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Expected 2 Excel files, but received %d files. Please upload exactly 2 Excel files.", len(files)),
		})
		return
	}

	// 保存上传的文件
	var dataSourcePath, clickDataPath string

	for i, file := range files {
		// 验证文件扩展名
		if !strings.HasSuffix(file.Filename, ".xlsx") && !strings.HasSuffix(file.Filename, ".xls") {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("File %d (%s) is not an Excel file. Only .xlsx and .xls files are accepted.", i+1, file.Filename),
			})
			return
		}

		// 保存文件到临时目录
		savePath := fmt.Sprintf("uploaded_excel_%d_%s", i+1, file.Filename)
		if err := ctx.SaveUploadedFile(file, savePath); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to save file %d: %v", i+1, err),
			})
			return
		}

		// 第一个文件作为数据源表，第二个文件作为打点表
		if i == 0 {
			dataSourcePath = savePath
		} else {
			clickDataPath = savePath
		}
	}

	// 处理Excel文件并执行回调
	result, err := processGdtExcelAndCallback(dataSourcePath, clickDataPath)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to process Excel files: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Successfully processed Excel files and executed callbacks",
		"result":  result,
	})
}

// 数据源记录结构
type DataGdtSourceRecord struct {
	AdId          string
	CallbackParam string
	AdvertiserId  string
	CampaignId    string
	IdfaSum       string
	Oaid          string
	OaidSum       string
	Imei          string
	ImeiSum       string
	ReqId         string
}

// 打点数据结构
type ClickGdtDataRecord struct {
	AdId       string
	ClickCount int
}

// processGdtExcelAndCallback 处理Excel文件并执行回调
func processGdtExcelAndCallback(dataSourcePath, clickDataPath string) (map[string]interface{}, error) {
	// 1. 读取数据源表（图一）
	dataSourceMap, err := readGdtDataSourceExcel(dataSourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read data source: %w", err)
	}

	// 2. 读取打点表（图二）
	clickDataList, err := readGdtClickDataExcel(clickDataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read click data: %w", err)
	}

	// 3. 根据打点表的广告ID和点击数，获取对应的callback_param并执行回调
	totalCallbacks := 0
	successCallbacks := 0
	failedCallbacks := 0
	var callbackDetails []map[string]interface{}

	for _, clickData := range clickDataList {
		adId := clickData.AdId
		clickCount := clickData.ClickCount

		// 从数据源中获取该广告ID的所有记录
		records, exists := dataSourceMap[adId]
		if !exists {
			callbackDetails = append(callbackDetails, map[string]interface{}{
				"ad_id":       adId,
				"click_count": clickCount,
				"status":      "skipped",
				"reason":      "ad_id not found in data source",
			})
			continue
		}

		// 获取指定数量的callback_param
		count := clickCount
		if count > len(records) {
			count = len(records) // 如果请求的数量超过可用记录，使用全部可用记录
		}

		callbackResults := []string{}
		for i := 0; i < count; i++ {
			record := records[i]
			// 执行回调
			success := executeGdtCallback(record)
			totalCallbacks++
			if success {
				successCallbacks++
				callbackResults = append(callbackResults, fmt.Sprintf("Success: %s", record.CallbackParam))
			} else {
				failedCallbacks++
				callbackResults = append(callbackResults, fmt.Sprintf("Failed: %s", record.CallbackParam))
			}
		}

		callbackDetails = append(callbackDetails, map[string]interface{}{
			"ad_id":          adId,
			"click_count":    clickCount,
			"executed_count": count,
			"status":         "completed",
			"callbacks":      callbackResults,
		})
	}

	result := map[string]interface{}{
		"total_callbacks":   totalCallbacks,
		"success_callbacks": successCallbacks,
		"failed_callbacks":  failedCallbacks,
		"details":           callbackDetails,
	}

	return result, nil
}

// readDataSourceExcel 读取数据源表（图一）
func readGdtDataSourceExcel(filePath string) (map[string][]DataGdtSourceRecord, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in data source file")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("data source file has no data rows")
	}

	// 解析表头，找到各列的索引
	header := rows[0]
	adIdCol := findGdtColumnIndex(header, "ad_id")
	callbackParamCol := findGdtColumnIndex(header, "callback_param")
	advertiserIdCol := findGdtColumnIndex(header, "advertiser_id")
	campaignIdCol := findGdtColumnIndex(header, "campaign_id")
	idfaSumCol := findGdtColumnIndex(header, "idfa_sum")
	oaidCol := findGdtColumnIndex(header, "oaid")
	oaidSumCol := findGdtColumnIndex(header, "oaid_sum")
	imeiCol := findGdtColumnIndex(header, "imei")
	imeiSumCol := findGdtColumnIndex(header, "imei_sum")
	reqIdCol := findGdtColumnIndex(header, "req_id")

	if adIdCol == -1 || callbackParamCol == -1 {
		return nil, fmt.Errorf("required columns (ad_id, callback_param) not found in data source")
	}

	// 按 ad_id 分组存储数据
	dataMap := make(map[string][]DataGdtSourceRecord)

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) <= adIdCol || len(row) <= callbackParamCol {
			continue
		}

		// 清理ad_id中的空白字符（包括制表符、换行符、空格等）
		adId := strings.TrimSpace(row[adIdCol])
		adId = strings.ReplaceAll(adId, "\t", "")
		adId = strings.ReplaceAll(adId, "\n", "")
		adId = strings.ReplaceAll(adId, "\r", "")

		record := DataGdtSourceRecord{
			AdId:          adId,
			CallbackParam: row[callbackParamCol],
		}

		if advertiserIdCol != -1 && len(row) > advertiserIdCol {
			record.AdvertiserId = row[advertiserIdCol]
		}
		if campaignIdCol != -1 && len(row) > campaignIdCol {
			record.CampaignId = row[campaignIdCol]
		}
		if idfaSumCol != -1 && len(row) > idfaSumCol {
			record.IdfaSum = row[idfaSumCol]
		}
		if oaidCol != -1 && len(row) > oaidCol {
			record.Oaid = row[oaidCol]
		}
		if oaidSumCol != -1 && len(row) > oaidSumCol {
			record.OaidSum = row[oaidSumCol]
		}
		if imeiCol != -1 && len(row) > imeiCol {
			record.Imei = row[imeiCol]
		}
		if imeiSumCol != -1 && len(row) > imeiSumCol {
			record.ImeiSum = row[imeiSumCol]
		}
		record.ReqId = row[reqIdCol]

		dataMap[record.AdvertiserId] = append(dataMap[record.AdvertiserId], record)
	}

	return dataMap, nil
}

// readClickDataExcel 读取打点表（图二）
func readGdtClickDataExcel(filePath string) ([]ClickGdtDataRecord, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in click data file")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("click data file has no data rows")
	}

	// 解析表头
	header := rows[0]
	adIdCol := findColumnIndex(header, "广告id", "广告ID", "ad_id")
	clickCountCol := findColumnIndex(header, "点击数", "点击次数", "click_count")

	if adIdCol == -1 || clickCountCol == -1 {
		return nil, fmt.Errorf("required columns (广告id, 点击数) not found in click data")
	}

	var clickDataList []ClickGdtDataRecord

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) <= adIdCol || len(row) <= clickCountCol {
			continue
		}

		clickCount, err := strconv.Atoi(row[clickCountCol])
		if err != nil {
			// 如果转换失败，跳过这一行
			continue
		}

		// 清理ad_id中的空白字符（包括制表符、换行符、空格等）
		adId := strings.TrimSpace(row[adIdCol])
		adId = strings.ReplaceAll(adId, "\t", "")
		adId = strings.ReplaceAll(adId, "\n", "")
		adId = strings.ReplaceAll(adId, "\r", "")

		record := ClickGdtDataRecord{
			AdId:       adId,
			ClickCount: clickCount,
		}

		clickDataList = append(clickDataList, record)
	}

	return clickDataList, nil
}

// findColumnIndex 查找列索引（支持多个可能的列名）
func findGdtColumnIndex(header []string, possibleNames ...string) int {
	for i, col := range header {
		for _, name := range possibleNames {
			if strings.TrimSpace(col) == name {
				return i
			}
		}
	}
	return -1
}

// executeCallback 执行回调操作
func executeGdtCallback(record DataGdtSourceRecord) bool {
	// 构建回调URL
	baseURL := "https://ad-ocpx.zhltech.net/track/62904f0109"

	// 构建查询参数
	params := url.Values{}
	params.Set("ms_task", "meishutest")
	params.Set("ms_place", "1")
	params.Set("ms_channel", "gdt")
	params.Set("callback_param", record.CallbackParam) // 使用传入的callback_param
	params.Set("advertiser_id", record.AdvertiserId)
	params.Set("campaign_id", record.CampaignId)
	params.Set("ad_id", record.AdId)
	params.Set("oaid", record.Oaid)
	params.Set("oaid_sum", record.OaidSum) // 使用数据源表的oaid_sum
	if record.IdfaSum != "" {
		params.Set("os", "ios")
		params.Set("muid", record.IdfaSum) // 使用数据源表的idfa_sum
	} else {
		params.Set("os", "android")
		params.Set("muid", record.Imei)
	}
	params.Set("req_id", record.ReqId+"-20260115") // 使用数据源表的req_id
	params.Set("debug", "1")
	params.Set("transformType", "49")

	callbackURL := baseURL + "?" + params.Encode()

	fmt.Printf("Executing callback: %s\n", callbackURL)

	// 发送HTTP GET请求
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(callbackURL)
	if err != nil {
		fmt.Printf("Callback failed for %s: %v\n", record.CallbackParam, err)
		return false
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("Callback success for %s: status %d\n", record.CallbackParam, resp.StatusCode)
		return true
	}

	fmt.Printf("Callback failed for %s: status %d\n", record.CallbackParam, resp.StatusCode)
	return false
}
