package main

import (
	"fmt"
	"log"

	"github.com/farshadnobody/Rubingo/rubingo"
)

func main() {
	client, err := rubingo.NewClient("my_session",
		rubingo.WithProxy("http://user:pass@gate.decodo.com:10001"),
		rubingo.WithProxyEnabled(true),
		rubingo.WithProxyExclude(
			// ═══════════════════════════════════════════════════════
			// گرفتن URL و سرورها
			// ═══════════════════════════════════════════════════════
			rubingo.ProxyOpRefreshURL, // درخواست به getdcmess.iranlms.ir
			rubingo.ProxyOpGetDCs,     // مشابه بالا

			// ═══════════════════════════════════════════════════════
			// درخواست‌های API
			// ═══════════════════════════════════════════════════════
			rubingo.ProxyOpAPI, // sendMessage, getUserInfo, ...

			// ═══════════════════════════════════════════════════════
			// فایل‌ها
			// ═══════════════════════════════════════════════════════
			rubingo.ProxyOpUpload,   // آپلود عکس، ویدیو، فایل
			rubingo.ProxyOpDownload, // دانلود عکس، ویدیو، فایل

			// ═══════════════════════════════════════════════════════
			// WebSocket
			// ═══════════════════════════════════════════════════════
			rubingo.ProxyOpWebSocket, // دریافت پیام‌های جدید
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	// با این تنظیمات، همه عملیات‌ها بدون پروکسی هستند!
	// معادل DisableProxy است

	fmt.Println("Client created:", client)
}
