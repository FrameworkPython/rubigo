package main

import (
	"fmt"
	"log"

	"github.com/farshadnobody/Rubingo/rubingo"
)

func main() {
	// ═══════════════════════════════════════════════════════
	// ساخت کلاینت با پروکسی
	// ═══════════════════════════════════════════════════════
	client, err := rubingo.NewClient("my_session",
		rubingo.WithProxy("http://user:pass@gate.decodo.com:10001"),
		rubingo.WithProxyEnabled(true),
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

	// ارسال پیام (از طریق پروکسی)
	result, err := client.SendMessage(rubingo.SendMessageOptions{
		ObjectGUID: "me",
		Text:       "این پیام از طریق پروکسی ارسال شد 🔒",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Message sent:", result.MessageID())

	client.Disconnect()
}
