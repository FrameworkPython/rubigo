package main

import (
	"fmt"

	"github.com/farshadnobody/Rubingo/rubingo"
)

func main() {
	client, _ := rubingo.NewClient("my_session")

	// ═══════════════════════════════════════════════════════
	// تنظیم آدرس
	// ═══════════════════════════════════════════════════════
	client.SetProxy("http://user:pass@gate.decodo.com:10001")

	// ═══════════════════════════════════════════════════════
	// فعال / غیرفعال
	// ═══════════════════════════════════════════════════════
	client.EnableProxy()
	client.DisableProxy()
	fmt.Println("فعال؟", client.IsProxyEnabled())

	// ═══════════════════════════════════════════════════════
	// استثنا کردن
	// ═══════════════════════════════════════════════════════
	client.ExcludeFromProxy(rubingo.ProxyOpUpload)
	client.ExcludeFromProxy(rubingo.ProxyOpDownload)
	fmt.Println("آپلود استثنا؟", client.IsExcludedFromProxy(rubingo.ProxyOpUpload))

	// ═══════════════════════════════════════════════════════
	// برگرداندن به حالت عادی
	// ═══════════════════════════════════════════════════════
	client.IncludeInProxy(rubingo.ProxyOpUpload)
	fmt.Println("آپلود استثنا؟", client.IsExcludedFromProxy(rubingo.ProxyOpUpload))

	// ═══════════════════════════════════════════════════════
	// عملیات‌های قابل استثنا
	// ═══════════════════════════════════════════════════════
	_ = rubingo.ProxyOpRefreshURL // گرفتن URL جدید
	_ = rubingo.ProxyOpGetDCs     // گرفتن لیست سرورها
	_ = rubingo.ProxyOpAPI        // درخواست‌های API
	_ = rubingo.ProxyOpUpload     // آپلود فایل
	_ = rubingo.ProxyOpDownload   // دانلود فایل
	_ = rubingo.ProxyOpWebSocket  // اتصال WebSocket
}
