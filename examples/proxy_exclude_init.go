package main

import (
	"fmt"
	"log"

	"github.com/farshadnobody/Rubingo/rubingo"
)

func main() {
	// ═══════════════════════════════════════════════════════
	// سناریو: پروکسی خارج از ایران
	// مشکل: سرور getdcmess.iranlms.ir فقط از داخل ایران جواب می‌دهد
	// راه‌حل: گرفتن URL بدون پروکسی، بقیه با پروکسی
	// ═══════════════════════════════════════════════════════
	client, err := rubingo.NewClient("my_session",
		rubingo.WithProxy("http://user:pass@gate.decodo.com:10001"),
		rubingo.WithProxyEnabled(true),
		rubingo.WithProxyExclude(
			rubingo.ProxyOpRefreshURL, // گرفتن URL جدید → بدون پروکسی
			rubingo.ProxyOpGetDCs,     // گرفتن لیست سرورها → بدون پروکسی
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}

	// GetDCs بدون پروکسی اجرا می‌شود ✅
	// بقیه درخواست‌ها با پروکسی ✅

	if err := client.Start(""); err != nil {
		log.Fatal(err)
	}

	// این پیام از طریق پروکسی ارسال می‌شود
	result, err := client.SendMessage(rubingo.SendMessageOptions{
		ObjectGUID: "me",
		Text:       "سلام از آلمان! 🇩🇪",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Message sent:", result.MessageID())

	client.Disconnect()
}
