---
layout: default
title: ارسال پیام
permalink: /docs/messaging/
---

# ارسال پیام

متن ساده، Markdown، ریپلای، پیام خودحذف و ویرایش.

---

## پیام ساده

```go
client.SendMessage(rubingo.SendMessageOptions{
    ObjectGUID: "u0xxxxxxxxxx",
    Text:       "سلام!",
})
```

---

## با Markdown

```go
client.SendMessage(rubingo.SendMessageOptions{
    ObjectGUID: "g0xxxxxxxxxx",
    Text:       "**متن بولد** و __ایتالیک__ و `کد` اینجا",
    ParseMode:  "markdown",
})
```

---

## ریپلای به پیام

```go
client.SendMessage(rubingo.SendMessageOptions{
    ObjectGUID:       "u0xxxxxxxxxx",
    Text:             "جواب پیامت اینه!",
    ReplyToMessageID: "123456789",
})
```

---

## پیام خودحذف

```go
client.SendMessage(rubingo.SendMessageOptions{
    ObjectGUID: "me",
    Text:       "این پیام ۳۰ ثانیه دیگه پاک می‌شه!",
    AutoDelete: 30, // ثانیه
})
```

---

## ویرایش پیام

```go
client.EditMessage("g0xxxxxxxxxx", "message_id", "متن جدید")
```

---

## حذف پیام

```go
client.DeleteMessages("g0xxxxxxxxxx", []string{"msg_id_1", "msg_id_2"})
```

---

## پین کردن پیام

```go
client.SetPinMessage("g0xxxxxxxxxx", "message_id", "Pin")
```

---

## فوروارد پیام

```go
client.ForwardMessages("g0source", "g0target", []string{"msg_id_1"})
```

> برای `ObjectGUID` می‌تونی از `"me"` یا `"self"` به جای GUID خودت استفاده کنی.
