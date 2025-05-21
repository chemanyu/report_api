package top

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type UserDetailVo struct {
	QueryUserDate string       `json:"query_user_date"`
	PageRequest   *PageRequest `json:"page_request"`
}

type UserDetailDTO struct {
	UserId              string `json:"user_id"`
	DayOneUserRetention bool   `json:"day_one_user_retention"`
	CardCouponNewUser   bool   `json:"card_coupon_new_user"`
	SubPublisherId      string `json:"sub_publisher_id"`
}

func (c *Client) QueryAllPagedUsers(
	ctx context.Context,
	apiName string,
	baseParams Parameters,
	listPath []string,
	userDate string,
	pageSize int,
) ([]*UserDetailDTO, error) {

	if _, ok := ctx.Deadline(); !ok {
		// 如果上层没有设置超时，就设置一个默认的
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	var allItems []*UserDetailDTO
	pageNo := 1

	for {
		// 添加分页参数
		baseParams["user_action_log_query_params"] = UserDetailVo{
			QueryUserDate: userDate,
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
		fmt.Println(paramsCopy)
		// 取出用户列表字段
		result := resp
		for _, key := range listPath {
			result = result.Get(key)
		}
		if result == nil {
			fmt.Println("数据为空")
			break
		}

		list := result.Get("result").Get("user_action_log_d_t_o")
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
			var dto UserDetailDTO
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
