package top

import (
	"context"
	"encoding/json"
	"fmt"
)

type TradeOrderVo struct {
	PageRequest *PageRequest `json:"page_request"`
}

type PageRequest struct {
	PageSize int `json:"page_size"`
	PageNum  int `json:"page_num"`
}

type TradeOrderDTO struct {
	RefundFee              string `json:"refund_fee"`
	ItemPicURL             string `json:"item_pic_url"`
	OrderID                string `json:"order_id"`
	ItemTitle              string `json:"item_title"`
	PayTime                string `json:"pay_time"` // 可以改为 int64
	StateDesc              string `json:"state_desc"`
	PartConfirmFee         string `json:"part_confirm_fee"`
	ActualPaidFee          string `json:"actual_paid_fee"`
	GmtCreate              string `json:"gmt_create"` // 可以改为 int64
	ItemID                 string `json:"item_id"`
	EstimateCommission     string `json:"estimate_commission"`
	EstimateCommissionRate string `json:"estimate_commission_rate"`
	StateCode              int    `json:"state_code"`
	ItemPrice              string `json:"item_price"`
	EndTime                string `json:"end_time"` // 可以改为 int64
	ItemImageURL           string `json:"item_image_url"`
	SubPublisherID         string `json:"sub_publisher_id"`
}

func (c *Client) QueryAllPagedOrders(
	ctx context.Context,
	apiName string,
	baseParams Parameters,
	listPath []string, // 例如：["alibaba_idle_affiliate_cps_order_query_response", "result", "model_list"]
	pageSize int,
) ([]*TradeOrderDTO, error) {

	var allItems []*TradeOrderDTO
	pageNo := 1

	for {
		// 添加分页参数
		baseParams["trade_order_vo"] = TradeOrderVo{
			PageRequest: &PageRequest{
				PageSize: pageSize,
				PageNum:  pageNo,
			},
		}

		// 每次都要复制一份参数防止引用问题
		paramsCopy := Parameters{}
		for k, v := range baseParams {
			paramsCopy[k] = v
		}

		// 发起请求
		resp, err := c.DoJson(ctx, apiName, paramsCopy)
		if err != nil {
			return nil, err
		}
		fmt.Println(resp)
		// 取出订单列表字段
		result := resp
		for _, key := range listPath {
			result = result.Get(key)
		}

		if result == nil {
			fmt.Println("数据为空")
			break
		}

		list := result.Get("result")
		items, err := list.Array()
		if err != nil {
			fmt.Println("列表数据有误或者没有数据")
			break // 说明没有更多了
		}

		// 把每项加入 allItems
		for i := range items {
			itemBytes, err := json.Marshal(items[i])
			if err != nil {
				continue
			}
			var dto TradeOrderDTO
			if err := json.Unmarshal(itemBytes, &dto); err != nil {
				continue
			}
			allItems = append(allItems, &dto)
		}

		// 如果返回的条数 < pageSize，就说明到末尾了
		nextPage, _ := result.Get("next_page").Bool()
		if !nextPage {
			break
		}
		pageNo++
	}

	return allItems, nil
}
