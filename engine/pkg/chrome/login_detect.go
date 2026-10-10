package chrome

import "strings"

// pageHasWeiboFeedContent true when real post text or mymblog JSON appeared (not guest shell).
func pageHasWeiboFeedContent(html string) bool {
	if strings.Contains(html, "detail_wbtext") {
		return true
	}
	if strings.Contains(html, `"text_raw"`) && strings.Contains(html, `"mymblog"`) {
		return true
	}
	if strings.Contains(html, `ajax/statuses/mymblog`) && strings.Contains(html, `"ok":1`) {
		return true
	}
	return false
}

// pageLooksLoggedInForScrape 已登录且目标页可抓取（比 detail_wbtext 更宽松，适配 SPA / 多标签）。
func pageLooksLoggedInForScrape(html, locationURL string) bool {
	loc := strings.ToLower(locationURL)
	lower := strings.ToLower(html)
	if strings.Contains(loc, "passport.weibo.com") || strings.Contains(lower, "sina visitor system") {
		return false
	}
	if pageHasWeiboFeedContent(html) {
		return true
	}
	if !strings.Contains(loc, "weibo.com") {
		return false
	}
	if strings.Contains(loc, "about:blank") {
		return false
	}
	if strings.Contains(loc, "weibo.com/u/") {
		if strings.Contains(html, `"islogin":1`) || strings.Contains(html, `"islogin":true`) {
			return true
		}
		if strings.Contains(html, "toolbar_repost") || strings.Contains(html, "head-info") || strings.Contains(html, "head-info_time") {
			return true
		}
		if strings.Contains(html, "Feed_wrap") && !strings.Contains(hGuestOnly(html), "default_avatar_male") {
			return true
		}
		if strings.Contains(html, "关注") && strings.Contains(html, "粉丝") && len(html) > 50_000 {
			return true
		}
	}
	return false
}

func hGuestOnly(html string) string { return html }

// pageNeedsManualLogin returns true when the current page looks like a login / visitor wall.
func pageNeedsManualLogin(html, locationURL string) bool {
	h := html
	loc := strings.ToLower(locationURL)
	lower := strings.ToLower(h)

	if strings.Contains(loc, "passport.weibo.com") {
		return true
	}
	if strings.Contains(lower, "sina visitor system") || strings.Contains(lower, "visitor/visitor") {
		return true
	}
	if strings.Contains(h, `"islogin":0`) || strings.Contains(h, `"islogin":false`) {
		return true
	}

	// weibo.com：仅有 Feed 骨架 / 默认头像占位 ≠ 已登录可抓 feed。
	if strings.Contains(loc, "weibo.com") {
		if pageLooksLoggedInForScrape(h, loc) {
			return false
		}
		if strings.Contains(h, "Feed_wrap") || strings.Contains(h, "default_avatar_male") {
			return true
		}
		return true
	}

	if strings.Contains(h, `"islogin":1`) || strings.Contains(h, `"islogin":true`) {
		return false
	}
	if strings.Contains(h, "detail_wbtext") || strings.Contains(h, "feed_list_item") {
		return false
	}
	if strings.Contains(h, "$CONFIG") && strings.Contains(h, `"uid"`) && strings.Contains(h, `"screen_name"`) {
		return false
	}
	if strings.Contains(lower, "passport.weibo.com") {
		return true
	}
	return false
}
