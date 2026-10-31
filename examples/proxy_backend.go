package main

import (
	"fmt"
	"log"

	"github.com/farshadnobody/Rubingo/rubingo"
)

// ═══════════════════════════════════════════════════════
// تابع ارسال پیام برای کاربر
// ═══════════════════════════════════════════════════════
func sendMessageToUser(auth, privateKey, userGUID, text string) error {
	// هر درخواست یک کلاینت جدید می‌سازد
	client, err := rubingo.NewClient("temp_session",
		rubingo.WithAuth(auth),
		rubingo.WithPrivateKey(privateKey),

		// پروکسی مشترک برای همه کاربران
		rubingo.WithProxy("http://user:pass@gate.decodo.com:10001"),
		rubingo.WithProxyEnabled(true),

		// گرفتن URL بدون پروکسی (سریع‌تر)
		rubingo.WithProxyExclude(
			rubingo.ProxyOpRefreshURL,
			rubingo.ProxyOpGetDCs,
		),

		// ذخیره سشن در callback (نه فایل)
		rubingo.WithSessionCallback(func(data rubingo.SessionData) {
			// ذخیره در دیتابیس
			fmt.Println("Session saved for:", data.GUID)
		}),
	)
	if err != nil {
		return err
	}

	if err := client.Connect(); err != nil {
		return err
	}

	_, err = client.SendMessage(rubingo.SendMessageOptions{
		ObjectGUID: userGUID,
		Text:       text,
	})

	client.Disconnect()
	return err
}

func main() {
	// شبیه‌سازی درخواست‌های همزمان
	for i := 0; i < 10; i++ {
		go func(id int) {
			err := sendMessageToUser(
				"user_auth_token",
				"user_private_key",
				"u0ABC123",
				fmt.Sprintf("پیام شماره %d", id),
			)
			if err != nil {
				log.Printf("Error sending message %d: %v", id, err)
			} else {
				log.Printf("Message %d sent successfully", id)
			}
		}(i)
	}

	// صبر برای اتمام
	select {}
}
