---
layout: default
title: جدول متدها
permalink: /docs/methods/
---

# جدول متدها

مرجع کامل تمام متدهای موجود در Rubingo.

---

## احراز هویت

| متد | توضیح |
|-----|-------|
| `NewClient(name, ...opts)` | ساخت کلاینت جدید |
| `Connect()` | اتصال به سرور |
| `Start(phone)` | ورود خودکار |
| `SendCodeExt(phone, passKey)` | ارسال کد تأیید |
| `SignInExt(code, phone, hash, tmp)` | ورود با کد |
| `ApplySignInResult(result)` | اعمال نتیجه ورود |
| `Disconnect()` | قطع اتصال |

---

## پیام

| متد | توضیح |
|-----|-------|
| `SendMessage(opts)` | ارسال پیام/فایل |
| `EditMessage(guid, id, text)` | ویرایش پیام |
| `DeleteMessages(guid, ids)` | حذف پیام |
| `ForwardMessages(src, dst, ids)` | فوروارد پیام |
| `SetPinMessage(guid, id, action)` | پین/آنپین |

---

## گروه/کانال

| متد | توضیح |
|-----|-------|
| `GetGroupInfo(guid)` | اطلاعات گروه |
| `GetChannelInfo(guid)` | اطلاعات کانال |
| `GetInfo(guid)` | اطلاعات (خودکار) |
| `BanGroupMember(g, u, action)` | بن/آنبن کاربر |
| `SetBlockUser(guid)` | بلاک کاربر |
| `UnblockUser(guid)` | آنبلاک کاربر |

---

## آپدیت‌ها

| متد | توضیح |
|-----|-------|
| `OnMessage(func)` | هندل تمام پیام‌ها |
| `OnMessageMatch(regex, func)` | هندل با فیلتر regex |
| `Run()` | شروع دریافت (blocking) |

---

## پروکسی

| متد | توضیح |
|-----|-------|
| `WithProxy(url)` | تنظیم پروکسی (option) |
| `WithProxyEnabled(bool)` | وضعیت اولیه (option) |
| `SetProxy(url)` | تغییر آدرس |
| `EnableProxy()` | فعال کردن |
| `DisableProxy()` | غیرفعال کردن |
| `IsProxyEnabled()` | وضعیت فعلی |

---

## سشن

| متد | توضیح |
|-----|-------|
| `GetSessionString()` | صادر کردن سشن |
| `NewClientFromString(str)` | وارد کردن سشن |
| `WithSessionCallback(func)` | callback سفارشی |
