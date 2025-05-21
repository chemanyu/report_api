package uapar

import (
	"strings"
)

const OS_TYPE_UNKNOWN = 0 //未知
const OS_TYPE_ANDROID = 1 //安卓
const OS_TYPE_IOS = 2     //苹果
const OS_TYPE_MAC = 3     //MAC 电脑
const OS_TYPE_WINDOWS = 4 //微软电脑

const BROWSER_TYPE_OUTHER = 0 //其他
const BROWSER_TYPE_WEIXIN = 1 //微信
const BROWSER_TYPE_QQ = 2     //QQ
const BROWSER_TYPE_DOUYIN = 3 //抖音

// 系统标识
var OS_RULE = map[string]int{"windows": OS_TYPE_WINDOWS, "macintosh": OS_TYPE_MAC, "android": OS_TYPE_ANDROID, "cpu iphone os": OS_TYPE_IOS}

// 浏览器标识
var BROWSER_RULE = map[string]int{"micromessenger": BROWSER_TYPE_WEIXIN, "mqqbrowser": BROWSER_TYPE_QQ, "aweme": BROWSER_TYPE_DOUYIN}

// 蜘蛛标识
var BOT_RULE = []string{"googlebot", "baiduspider", "yahoo! slurp", "msnbot", "sosospider", "yodaoBot", "sogou web spider", "fast-webcrawler", "gaisbot", "ia_archiver", "altavista", "lycos_spider", "inktomi slurp", "googlebot-mobile", "360spider", "haosouspider", "sogou news spider", "youdaobot", "bingbot", "yisouspider", "easouspider", "jikespider", "sogou blog"}

// 解析操作系统类型
func ParseOsByUA(ua string) int {
	if ua != "" {
		ua = strings.ToLower(ua)

		for k, v := range OS_RULE {
			if strings.Index(ua, k) > -1 {
				return v
			}
		}
	}
	return OS_TYPE_UNKNOWN
}

// 解析浏览器类型
func ParseBrowserByUA(ua string) int {
	if ua != "" {
		ua = strings.ToLower(ua)
		for k, v := range BROWSER_RULE {
			if strings.Index(ua, k) > -1 {
				return v
			}
		}
	}
	return BROWSER_TYPE_OUTHER
}

// 过滤蜘蛛流量
func FilterBot(ua string) bool {
	if ua != "" {
		ua = strings.ToLower(ua)
		for _, v := range BOT_RULE {
			if strings.Index(ua, v) > -1 {
				return true
			}
		}
	}
	return false
}
