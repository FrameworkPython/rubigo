---
layout: default
title: مدیریت سشن
permalink: /docs/session/
---

# مدیریت سشن

ذخیره سشن در SQLite، callback سفارشی و Session String.

---

## پیش‌فرض — SQLite

```go
client, _ := rubingo.NewClient("my_session")
// سشن در فایل my_session.db ذخیره می‌شه
```

---

## Session String

```go
// صادر کردن
sessionStr, err := client.GetSessionString()

// وارد کردن در جای دیگه
client2, _ := rubingo.NewClientFromString(sessionStr)
```

> Session String برای انتقال سشن بین سرورها یا بکاپ گرفتن ایده‌آله.

---

## Callback سفارشی

```go
client, _ := rubingo.NewClient("my_bot",
    rubingo.WithSessionCallback(func(data rubingo.SessionData) {
        // data.Auth, data.GUID, ... رو توی DB خودت ذخیره کن
        fmt.Printf("Session saved: %s\n", data.GUID)
    }),
)
```
