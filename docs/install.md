---
layout: default
title: نصب و راه‌اندازی
permalink: /docs/install/
---

# نصب و راه‌اندازی

در ۴ قدم ساده پروژه‌ات رو راه‌اندازی کن. نیاز به Go `1.20+` داری.

---

## ۱. ساخت پوشه پروژه

```bash
mkdir my-rubika-bot
cd my-rubika-bot
go mod init github.com/YOUR_USERNAME/my-rubika-bot
```

| هدف | دستور |
|-----|-------|
| پروژه شخصی، بدون انتشار | `go mod init mybot` |
| می‌خوای بقیه با `go get` نصب کنن | `go mod init github.com/USERNAME/REPONAME` |

---

## ۲. نصب Rubingo

```bash
go get github.com/farshadnobody/Rubingo
```

---

## ۳. تکمیل وابستگی‌ها

```bash
go mod tidy
```

> این دستور تمام پکیج‌های لازم رو دانلود می‌کنه و فایل `go.sum` رو هم می‌سازه.

---

## ۴. اجرای پروژه

```bash
go run .
# یا
go run main.go
```
