package main

import (
	"fmt"
	"log"

	"github.com/farshadnobody/Rubingo/rubingo"
)

func main() {
	// ساخت کلاینت
	client, err := rubingo.NewClient("my_session")
	if err != nil {
		log.Fatal(err)
	}

	// اتصال
	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}

	// شروع (لاگین یا ثبت‌نام)
	if err := client.Start(""); err != nil {
		log.Fatal(err)
	}

	// ارسال پیام
	result, err := client.SendMessage(rubingo.SendMessageOptions{
		ObjectGUID: "me",
		Text:       "سلام! این پیام از **Go** ارسال شده 🚀",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Message sent:", result.MessageID())

	// قطع اتصال
	client.Disconnect()
}
