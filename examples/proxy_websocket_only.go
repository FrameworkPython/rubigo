package main

import (
	"fmt"
	"log"

	"github.com/farshadnobody/Rubingo/rubingo"
)

func main() {
	// ═══════════════════════════════════════════════════════
	// سناریو: می‌خواهیم فقط WebSocket از پروکسی باشد
	// چون ISP ما WebSocket را فیلتر می‌کند
	// ولی API و فایل‌ها مستقیم سریع‌تر هستند
	// ═══════════════════════════════════════════════════════
	client, err := rubingo.NewClient("my_session",
		rubingo.WithProxy("http://user:pass@gate.decodo.com:10001"),
		rubingo.WithProxyEnabled(true),
		rubingo.WithProxyExclude(
			rubingo.ProxyOpRefreshURL,
			rubingo.ProxyOpGetDCs,
			rubingo.ProxyOpAPI,
			rubingo.ProxyOpUpload,
			rubingo.ProxyOpDownload,
			// ProxyOpWebSocket استثنا نشده → از پروکسی استفاده می‌کند
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}

	if err := client.Start(""); err != nil {
		log.Fatal(err)
	}

	// هندلر پیام
	client.OnMessage(func(u *rubingo.Update) {
		// این پیام از طریق WebSocket (با پروکسی) دریافت شد
		fmt.Println("پیام جدید:", u.Text())

		// ولی پاسخ از طریق API (بدون پروکسی) ارسال می‌شود
		u.Reply("دریافت شد!")
	})

	fmt.Println("ربات شروع شد...")
	fmt.Println("WebSocket: با پروکسی 🔒")
	fmt.Println("API: بدون پروکسی ⚡")

	client.Run()
}
