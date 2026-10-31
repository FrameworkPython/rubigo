---
layout: default
title: گروه و کانال
permalink: /docs/groups/
---

# گروه و کانال

مدیریت کامل گروه‌ها و کانال‌ها.

---

## دریافت اطلاعات

```go
info, _ := client.GetGroupInfo("g0xxxxxxxxxx")
info, _ := client.GetChannelInfo("c0xxxxxxxxxx")

// تشخیص خودکار نوع:
info, _ := client.GetInfo("g0xxxxxxxxxx")
```

---

## بن و آنبن کاربر

```go
client.BanGroupMember("g0xxxxxxxxxx", "u0xxxxxxxxxx", "Set")   // بن
client.BanGroupMember("g0xxxxxxxxxx", "u0xxxxxxxxxx", "Unset") // آنبن
```

---

## بلاک و آنبلاک

```go
client.SetBlockUser("u0xxxxxxxxxx")
client.UnblockUser("u0xxxxxxxxxx")
```
