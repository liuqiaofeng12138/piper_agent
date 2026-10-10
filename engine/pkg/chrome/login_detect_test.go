package chrome

import "testing"

func TestPageNeedsManualLogin(t *testing.T) {
	if !pageNeedsManualLogin(`<title>Sina Visitor System</title>`, "https://weibo.com/u/1") {
		t.Fatal("visitor page")
	}
	if pageNeedsManualLogin(`"islogin":1`, "https://weibo.com/u/5720474518") {
		t.Fatal("islogin on profile should not need login wait")
	}
	if !pageLooksLoggedInForScrape(`"islogin":1 Feed_wrap 置顶 toolbar_repost`, "https://weibo.com/u/5720474518") {
		t.Fatal("logged in profile")
	}
	if pageNeedsManualLogin(`Feed_wrap detail_wbtext`, "https://weibo.com/u/5720474518") {
		t.Fatal("feed visible")
	}
	if !pageNeedsManualLogin(`Feed_wrap default_avatar_male`, "https://weibo.com/u/5720474518") {
		t.Fatal("guest shell should need login")
	}
}
