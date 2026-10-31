package main

import (
	"fmt"
	"log"

	"github.com/farshadnobody/Rubingo/rubingo"
)

func main() {
	client, err := rubingo.NewClient("my_session")
	if err != nil {
		log.Fatal(err)
	}

	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}

	if err := client.Start(""); err != nil {
		log.Fatal(err)
	}

	// ═══════════════════════════════════════════════════════
	// لیست پروکسی‌ها
	// ═══════════════════════════════════════════════════════
	proxies := []string{
		"socks5://proxy1.example.com:1080",
		"socks5://proxy2.example.com:1080",
		"http://proxy3.example.com:8080",
	}
	currentProxy := 0

	// ═══════════════════════════════════════════════════════
	// تابع تغییر پروکسی
	// ═══════════════════════════════════════════════════════
	switchProxy := func() {
		currentProxy = (currentProxy + 1) % len(proxies)
		client.SetProxy(proxies[currentProxy])
		fmt.Println("پروکسی جدید:", proxies[currentProxy])
	}

	// ═══════════════════════════════════════════════════════
	// فعال کردن اولین پروکسی
	// ═══════════════════════════════════════════════════════
	client.SetProxy(proxies[0])
	client.EnableProxy()

	// ═══════════════════════════════════════════════════════
	// هندلر پیام
	// ═══════════════════════════════════════════════════════
	client.OnMessage(func(u *rubingo.Update) {
		text := u.Text()

		if text == "/proxy on" {
			client.EnableProxy()
			u.Reply("پروکسی فعال شد ✅")

		} else if text == "/proxy off" {
			client.DisableProxy()
			u.Reply("پروکسی غیرفعال شد ❌")

		} else if text == "/proxy switch" {
			switchProxy()
			u.Reply(fmt.Sprintf("پروکسی عوض شد: %s", proxies[currentProxy]))

		} else if text == "/proxy status" {
			status := "غیرفعال ❌"
			if client.IsProxyEnabled() {
				status = fmt.Sprintf("فعال ✅ - %s", proxies[currentProxy])
			}
			u.Reply(fmt.Sprintf("وضعیت پروکسی: %s", status))
		}
	})

	fmt.Println("ربات شروع شد...")
	fmt.Println("دستورات: /proxy on, /proxy off, /proxy switch, /proxy status")

	client.Run()
}
