---
layout: default
title: شروع سریع
permalink: /docs/quickstart/
---

# شروع سریع

سه روش مختلف برای شروع — از ساده‌ترین تا کنترل کامل.

---

## روش ۱ — ساده‌ترین حالت

کتابخانه خودش ورود یا ثبت‌نام رو مدیریت می‌کنه:

```go
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

    client.Connect()
    client.Start("") // اگر سشن نداشته باشد، خودکار می‌پرسد

    result, _ := client.SendMessage(rubingo.SendMessageOptions{
        ObjectGUID: "me",
        Text:       "سلام! این پیام از Go ارسال شده 🚀",
    })
    fmt.Println("Message sent:", result.MessageID())
    client.Disconnect()
}
```

---

## روش ۲ — با شماره از پیش تعریف‌شده

```go
client.Start("989912345678") // دیگر در ترمینال نمی‌پرسد
```

---

## روش ۳ — کنترل کامل ورود

مناسب برای سرور و API:

```go
sendResult, _ := client.SendCodeExt(phoneNumber, "")

signInResult, _ := client.SignInExt(
    code,
    phoneNumber,
    sendResult.PhoneCodeHash,
    sendResult.TmpSession,
)

client.ApplySignInResult(signInResult)
fmt.Println("GUID:", signInResult.UserGUID)
```

> **نکته:** شماره تلفن باید با کد کشور باشه: `989912345678` — بدون `+` یا `00`
