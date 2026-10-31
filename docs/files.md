---
layout: default
title: ارسال فایل
permalink: /docs/files/
---

# ارسال فایل

تصویر، ویدیو، موسیقی، صدا و فایل — از مسیر، URL یا `[]byte`.

---

## تصویر از مسیر

```go
client.SendMessage(rubingo.SendMessageOptions{
    ObjectGUID: "g0xxxxxxxxxx",
    FileInline: "/path/to/image.jpg",
    Type:       "Image",
    Text:       "توضیحات تصویر",
})
```

---

## تصویر از URL

```go
client.SendMessage(rubingo.SendMessageOptions{
    ObjectGUID: "g0xxxxxxxxxx",
    FileInline: "https://example.com/photo.jpg",
    Type:       "Image",
})
```

---

## ویدیو

```go
client.SendMessage(rubingo.SendMessageOptions{
    ObjectGUID: "g0xxxxxxxxxx",
    FileInline: "/path/to/video.mp4",
    Type:       "Video",
})
```

---

## موسیقی

```go
client.SendMessage(rubingo.SendMessageOptions{
    ObjectGUID: "me",
    FileInline: "/path/to/music.mp3",
    Type:       "Music",
    Performer:  "نام خواننده",
})
```

---

## دانلود فایل دریافتی

```go
client.OnMessage(func(u *rubingo.Update) {
    fileInline := u.GetUpdate("message.file_inline")
    if fileInline != nil {
        data, err := client.DownloadFile(fileInline, "saved_file.jpg")
        if err != nil { return }
        fmt.Printf("دانلود شد: %d بایت\n", len(data))
    }
})
```

---

## انواع فایل

| Type | نوع |
|------|-----|
| `Image` | تصویر (jpg, png, webp, ...) |
| `Video` | ویدیو (mp4, mkv, ...) |
| `Music` | موسیقی (mp3, flac, ...) |
| `Voice` | صدا (ogg, ...) |
| `File` | هر فایل دیگه |
| `Gif` | GIF متحرک |
