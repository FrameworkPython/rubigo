package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

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

	// ارسال کد
	phoneNumber := "989912345678"
	sendResult, err := client.SendCodeExt(phoneNumber, "")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("TmpSession:", sendResult.TmpSession)
	fmt.Println("PhoneCodeHash:", sendResult.PhoneCodeHash)
	fmt.Println("Status:", sendResult.Status)

	// دریافت کد از کاربر
	fmt.Print("Enter verification code: ")
	reader := bufio.NewReader(os.Stdin)
	code, _ := reader.ReadString('\n')
	code = strings.TrimSpace(code)

	// ورود
	signInResult, err := client.SignInExt(
		code,
		phoneNumber,
		sendResult.PhoneCodeHash,
		sendResult.TmpSession,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Auth:", signInResult.Auth)
	fmt.Println("UserGUID:", signInResult.UserGUID)

	// ✅ به‌روزرسانی کلاینت با Auth جدید
	client.ApplySignInResult(signInResult)

	// ارسال پیام
	msgResult, err := client.SendMessage(rubingo.SendMessageOptions{
		ObjectGUID: "me",
		Text:       "سلام!",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Message sent:", msgResult.MessageID())

	client.Disconnect()
}
