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
	client, err := rubingo.NewClient("my_session",
		rubingo.WithProxy("http://user:pass@gate.decodo.com:10001"),
		rubingo.WithProxyEnabled(true),

		// این عملیات‌ها بدون پروکسی اجرا می‌شوند
		rubingo.WithProxyExclude(
			rubingo.ProxyOpRefreshURL, // دریافت URL از سرور روبیکا
			rubingo.ProxyOpGetDCs,     // دریافت لیست سرورها
			rubingo.ProxyOpWebSocket,  // وقتی client.Start() یا client.Run() صدا می‌زنی، یه اتصال WebSocket دائمی برقرار می‌شه که پیام‌های جدید از طریقش دریافت می‌شن. این عملیات مربوط به همون اتصال WSS هست. اگه استثنا کنی، اتصال WebSocket مستقیم برقرار می‌شه بدون پروکسی.
			rubingo.ProxyOpAPI,        // هر بار که یه متد API صدا می‌زنی — SendMessage، SendCodeExt، SignInExt، GetChats و هر چیز دیگه‌ای — درخواست HTTP از این مسیر رد می‌شه. اگه استثنا کنی، همه این متدها مستقیم (بدون پروکسی) به سرور روبیکا وصل می‌شن.
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}

	// ارسال کد (از پروکسی استفاده می‌کند)
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

	// ورود (از پروکسی استفاده می‌کند)
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

	client.ApplySignInResult(signInResult)

	// ارسال پیام (بدون پروکسی - چون ProxyOpAPI استثنا شده)
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
