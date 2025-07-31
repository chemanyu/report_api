package top

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type IncomeDetailVo struct {
	//BillState   int          `json:"bill_state"`
	PageRequest          *PageRequest `json:"page_request"`
	StartCreateTimeStamp int64        `json:"start_create_time_stamp,omitempty"` // 开始创建时间戳
	EndCreateTimeStamp   int64        `json:"end_create_time_stamp,omitempty"`   // 结束创建时间戳
	CreateMonth          string       `json:"create_month,omitempty"`            // 创建月份
	StartUpdateTime      int64        `json:"start_update_time,omitempty"`       // 开始更新时间戳
	EndUpdateTime        int64        `json:"end_update_time,omitempty"`         // 结束更新时间戳
}

type IncomeDetailDTO struct {
	BillInfo      *BillInfo       `json:"bill_info"`
	BillID        string          `json:"bill_id"`
	Deeplink      string          `json:"deeplink"`
	PlainBillId   string          `json:"plain_bill_id"`
	TradeOrderDTO *TradeOrderDTO2 `json:"trade_order_d_t_o"`
}

type BillInfo struct {
	StateText        string `json:"state_text"`
	SettleAmountText string `json:"settle_amount_text"`
	AssessAmountText string `json:"assess_amount_text"`
	SubPublisherID   string `json:"sub_publisher_id"`
	State            int    `json:"state"`
	AccountingTime   string `json:"accounting_time"`
}

type TradeOrderDTO2 struct {
	CouponDiscountFee string `json:"coupon_discount_fee"`
	ItemPicURL        string `json:"item_pic_url"`
	ItemID            string `json:"item_id"`
	PayTime           string `json:"pay_time"`
	ItemTitle         string `json:"item_title"`
	OrderID           string `json:"order_id"`
	StateDesc         string `json:"state_desc"`
	PartConfirmFee    string `json:"part_confirm_fee"`
	ActualPaidFee     string `json:"actual_paid_fee"`
	GmtCreate         string `json:"gmt_create"`
	StateCode         int    `json:"state_code"`
}

func (c *Client) QueryAllPagedIncomes(
	ctx context.Context,
	apiName string,
	paramsReq Parameters,
	baseParams Parameters,
	listPath []string,
	billState int,
	pageSize int,
) ([]*IncomeDetailDTO, error) {

	if _, ok := ctx.Deadline(); !ok {
		// 如果上层没有设置超时，就设置一个默认的
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	var allItems []*IncomeDetailDTO
	pageNo := 1

	for {
		// 添加分页参数
		paramsReq["income_detail_vo"] = IncomeDetailVo{
			//BillState: billState,
			PageRequest: &PageRequest{
				PageSize: pageSize,
				PageNum:  pageNo,
			},
			StartCreateTimeStamp: baseParams["start_create_time_stamp"].(int64),
			EndCreateTimeStamp:   baseParams["end_create_time_stamp"].(int64),
			CreateMonth:          baseParams["create_month"].(string),
			StartUpdateTime:      baseParams["start_update_time"].(int64),
			EndUpdateTime:        baseParams["end_update_time"].(int64),
		}
		fmt.Println(paramsReq)
		// 每次都要复制一份参数防止引用问题
		paramsCopy := Parameters{}
		for k, v := range paramsReq {
			paramsCopy[k] = v
		}

		// 发起请求
		resp, err := c.DoJson(ctx, apiName, paramsCopy)
		if err != nil {
			return nil, err
		}
		fmt.Println(resp)
		// 取出佣金列表字段
		result := resp
		for _, key := range listPath {
			result = result.Get(key)
		}

		if result == nil {
			fmt.Println("数据为空")
			break
		}

		list := result.Get("result").Get("commission_detail_d_t_o")
		fmt.Println(list)
		items, err := list.Array()
		if err != nil {
			fmt.Println("列表数据有误或者没有数据")
			break // 说明没有更多了
		}
		fmt.Println("res", items)
		// 把每项加入 allItems
		for i := range items {
			itemBytes, err := json.Marshal(items[i])
			if err != nil {
				continue
			}
			var dto IncomeDetailDTO
			fmt.Println("json", len(itemBytes))
			if err := json.Unmarshal(itemBytes, &dto); err != nil {
				fmt.Println("err", err)
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
