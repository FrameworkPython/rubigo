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

	// ✅ شماره را مستقیماً اینجا بدهید - دیگر در CMD نمی‌پرسد
	if err := client.Start("989912345678"); err != nil {
		log.Fatal(err)
	}

	// ادامه کد...
	result, err := client.SendMessage(rubingo.SendMessageOptions{
		ObjectGUID: "me",
		Text:       "سلام!",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Message sent:", result.MessageID())

	client.Disconnect()
}
