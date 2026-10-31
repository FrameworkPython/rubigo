# Rubingo 🚀

**Rubingo** یک کتابخونه Go برای ارتباط با پیام‌رسان **روبیکا** است.  
با این کتابخونه می‌تونی ربات بسازی، پیام بفرستی، فایل آپلود کنی، گروه و کانال مدیریت کنی و خیلی کارهای دیگه.

---

## ✨ قابلیت‌ها

- **احراز هویت آسان** — ورود با شماره تلفن + تأیید OTP (و پشتیبانی از رمز دو مرحله‌ای)
- **ارسال پیام** — متن ساده، فرمت‌بندی Markdown/HTML، ریپلای، پیام خودحذف
- **ارسال فایل** — تصویر، ویدیو، موسیقی، صدا، فایل (از مسیر، URL یا `[]byte`)
- **دریافت آپدیت‌ها** — هندل کردن پیام‌های ورودی به صورت real-time با WebSocket
- **مدیریت گروه/کانال** — اطلاعات، بن/آنبن، ادمین‌ها، فوروارد، حذف پیام
- **مدیریت سشن** — ذخیره سشن در SQLite یا callback سفارشی (برای دیتابیس خودتون)
- **Session String** — صادر/وارد کردن سشن به صورت رشته
- **دانلود فایل** — دانلود مستقیم رسانه‌های دریافتی

---

## 📦 نصب و راه‌اندازی پروژه جدید

اگه می‌خوای یه پروژه جدید بسازی که از Rubingo استفاده کنه، مراحل زیر رو دنبال کن:

**۱. ساخت پوشه پروژه و فایل `go.mod`:**
```bash
mkdir my-rubika-bot
cd my-rubika-bot
go mod init github.com/YOUR_USERNAME/my-rubika-bot
```

> **توضیح:**
> - به جای `YOUR_USERNAME` یوزرنیم گیتهاب خودت رو بنویس (مثلاً `ali123`)
> - به جای `my-rubika-bot` هر اسمی که دوست داری برای پروژه‌ات بذار
> - این اسم **هیچ ربطی به اسم پوشه‌ات نداره** و می‌تونه کاملاً متفاوت باشه
> - فرمت `github.com/...` فقط یه **قرارداد** هست — اگه می‌خوای بقیه بتونن با `go get` پکیجت رو نصب کنن، اسم module باید با آدرس واقعیش روی گیتهاب یکی باشه. وگرنه می‌تونی فقط بنویسی `go mod init mybot` و کافیه

| هدف | دستور |
|-----|-------|
| پروژه شخصی، بدون انتشار | `go mod init mybot` |
| می‌خوای بقیه با `go get` نصب کنن | `go mod init github.com/USERNAME/REPONAME` |

**۲. نصب کتابخونه Rubingo:**
```bash
go get github.com/farshadnobody/Rubingo
```

**۳. تکمیل و دانلود وابستگی‌ها:**
```bash
go mod tidy
```
> این دستور تمام پکیج‌های لازم رو دانلود می‌کنه و فایل `go.sum` رو هم می‌سازه. دیگه نیازی به `go mod download` جداگانه نیست.

**۴. اجرای پروژه:**
```bash
go run .
# یا
go run main.go
```

> **نیازمندی‌ها:** Go 1.20+

---

## ⚡ راه‌اندازی سریع

### روش ۱ — ساده‌ترین حالت (`Start`)

کتابخونه خودش ورود یا ثبت‌نام رو مدیریت می‌کنه:

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

    if err := client.Connect(); err != nil {
        log.Fatal(err)
    }

    // اگر سشن نداشته باشد، خودکار می‌پرسد
    if err := client.Start(""); err != nil {
        log.Fatal(err)
    }

    result, err := client.SendMessage(rubingo.SendMessageOptions{
        ObjectGUID: "me",
        Text:       "سلام! این پیام از **Go** ارسال شده 🚀",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Message sent:", result.MessageID())

    client.Disconnect()
}
```

### روش ۲ — با شماره از پیش تعریف‌شده

```go
// شماره را مستقیماً بدهید تا دیگر در ترمینال نپرسد
if err := client.Start("989912345678"); err != nil {
    log.Fatal(err)
}
```

### روش ۳ — کنترل کامل ورود (مناسب برای سرور/API)

اگه می‌خوای سشن رو در دیتابیس خودت ذخیره کنی:

```go
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

    // ارسال کد تأیید
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

    // اعمال نتیجه ورود روی کلاینت
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
```

---

## 📖 نمونه‌های استفاده

### ارسال پیام با Markdown

```go
client.SendMessage(rubingo.SendMessageOptions{
    ObjectGUID: "g0xxxxxxxxxx", // GUID گروه
    Text:       "**متن بولد** و __ایتالیک__ و `کد` اینجا",
    ParseMode:  "markdown",
})
```

### ریپلای به پیام

```go
client.SendMessage(rubingo.SendMessageOptions{
    ObjectGUID:       "u0xxxxxxxxxx",
    Text:             "جواب پیامت اینه!",
    ReplyToMessageID: "123456789",
})
```

### پیام خودحذف (AutoDelete)

```go
client.SendMessage(rubingo.SendMessageOptions{
    ObjectGUID: "me",
    Text:       "این پیام ۳۰ ثانیه دیگه پاک می‌شه!",
    AutoDelete: 30, // ثانیه
})
```

### ارسال تصویر

```go
client.SendMessage(rubingo.SendMessageOptions{
    ObjectGUID: "g0xxxxxxxxxx",
    FileInline: "/path/to/image.jpg", // مسیر فایل
    Type:       "Image",
    Text:       "توضیحات تصویر",
})
```

### ارسال تصویر از URL

```go
client.SendMessage(rubingo.SendMessageOptions{
    ObjectGUID: "g0xxxxxxxxxx",
    FileInline: "https://example.com/photo.jpg",
    Type:       "Image",
})
```

### ارسال موسیقی

```go
client.SendMessage(rubingo.SendMessageOptions{
    ObjectGUID: "me",
    FileInline: "/path/to/music.mp3",
    Type:       "Music",
    Performer:  "نام خواننده",
})
```

### ارسال ویدیو

```go
client.SendMessage(rubingo.SendMessageOptions{
    ObjectGUID: "g0xxxxxxxxxx",
    FileInline: "/path/to/video.mp4",
    Type:       "Video",
})
```

### فوروارد پیام

```go
client.ForwardMessages("g0source_guid", "g0target_guid", []string{"msg_id_1", "msg_id_2"})
```

### ویرایش پیام

```go
client.EditMessage("g0xxxxxxxxxx", "message_id", "متن جدید")
```

### حذف پیام

```go
client.DeleteMessages("g0xxxxxxxxxx", []string{"msg_id_1", "msg_id_2"})
```

### پین کردن پیام

```go
client.SetPinMessage("g0xxxxxxxxxx", "message_id", "Pin")
```

### بن/آنبن کاربر در گروه

```go
client.BanGroupMember("g0xxxxxxxxxx", "u0xxxxxxxxxx", "Set")   // بن
client.BanGroupMember("g0xxxxxxxxxx", "u0xxxxxxxxxx", "Unset") // آنبن
```

### بلاک/آنبلاک کاربر

```go
client.SetBlockUser("u0xxxxxxxxxx")
client.UnblockUser("u0xxxxxxxxxx")
```

### اطلاعات گروه یا کانال

```go
info, _ := client.GetGroupInfo("g0xxxxxxxxxx")
info, _ := client.GetChannelInfo("c0xxxxxxxxxx")

// یا به صورت عمومی (تشخیص خودکار نوع):
info, _ := client.GetInfo("g0xxxxxxxxxx")
```

### ربات پاسخ‌دهنده — هندل کردن پیام‌های ورودی

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
    client.Start("")

    // هر پیام ورودی
    client.OnMessage(func(u *rubingo.Update) {
        text := u.GetString("message.text")
        senderGUID := u.GetString("message.author_object_guid")
        chatGUID := u.GetString("object_guid")

        fmt.Printf("پیام از %s: %s\n", senderGUID, text)

        client.SendMessage(rubingo.SendMessageOptions{
            ObjectGUID: chatGUID,
            Text:       "پیامت رسید ✅",
        })
    })

    // فقط پیام‌هایی که با /start شروع می‌شوند
    client.OnMessageMatch(`^/start`, func(u *rubingo.Update) {
        chatGUID := u.GetString("object_guid")
        client.SendMessage(rubingo.SendMessageOptions{
            ObjectGUID: chatGUID,
            Text:       "خوش اومدی! 👋",
        })
    })

    // شروع دریافت آپدیت (blocking)
    client.Run()
}
```

### دانلود فایل دریافتی

```go
client.OnMessage(func(u *rubingo.Update) {
    fileInline := u.GetUpdate("message.file_inline")
    if fileInline != nil {
        data, err := client.DownloadFile(fileInline, "saved_file.jpg")
        if err != nil {
            fmt.Println("خطا در دانلود:", err)
            return
        }
        fmt.Printf("فایل دانلود شد: %d بایت\n", len(data))
    }
})
```

### Session String (انتقال سشن بین دستگاه‌ها)

```go
// صادر کردن
sessionStr, err := client.GetSessionString()

// وارد کردن در جای دیگه
client2, _ := rubingo.NewClientFromString(sessionStr)
```

---

## 🗂️ ساختار ریپازیتوری

```
Rubingo/
├── rubingo/
│   └── rubingo.go          # کد اصلی کتابخونه
├── examples/
│   ├── quickstart/
│   │   └── main.go         # مثال ساده با Start
│   ├── manual_login/
│   │   └── main.go         # مثال ورود دستی
│   └── bot/
│       └── main.go         # مثال ربات پاسخ‌دهنده
├── README.md
├── LICENSE
└── go.mod
```

---

## 🛠️ تنظیمات `go.mod` کتابخونه

```
module github.com/farshadnobody/Rubingo

go 1.21

require (
    github.com/gorilla/websocket v1.5.1
    modernc.org/sqlite v1.29.0
)
```

---

## ⚠️ نکات مهم

- شماره تلفن باید با کد کشور باشه: `989912345678` (بدون `+` یا `00`)
- برای `ObjectGUID` می‌تونی از `"me"` یا `"self"` به جای GUID خودت استفاده کنی
- سشن به صورت پیش‌فرض در یک فایل SQLite ذخیره می‌شه و دفعه بعد نیازی به ورود دوباره نیست

---

## 📄 مجوز

MIT License

---

> ساخته شده با ❤️ برای جامعه Go ایران
