package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/account"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/enums"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"

	_ "modernc.org/sqlite"
)

var (
	TARGET_GROUP_GUID    = "g0DHpOn042b93c4982b5b135365c7bb6"
	BROADCAST_GROUP_GUID = "g0EgTH60e7fac57047e1e8fd03e4ba77"
	MAIN_ADMIN_GUID      = "u0JNjam04cd438e3013ebd0111bc3fb3"
	GITHUB_GUID_URL      = "https://raw.githubusercontent.com/FrameworkPython/BanerGozarGuid/main/README.md"
	GITHUB_SYNC_HOUR     = 23
	GITHUB_SYNC_MINUTE   = 59
	GITHUB_FETCH_RETRIES = 4
	GITHUB_FETCH_TIMEOUT = 15 * time.Second

	IS_BOT_ACTIVE          = true
	WARNING_AFTER_HOURS    = 5.0
	AUTO_CLOSE_AFTER_HOURS = 6.0
	FORWARD_RETRY_DELAY    = 10 * time.Second
	PENDING_EXPIRE_SECONDS = 20
	PROCESSED_CACHE_LIMIT  = 10000
	FORWARD_WORKERS        = 2
	AUTO_DELETE_SECONDS    = 15
	BROADCAST_INTERVAL     = 5 * time.Second
	MAX_MENTIONS_PER_MSG   = 30
	TAG_CHUNK_LIMIT        = 3800
	TAG_SEND_INTERVAL      = 2 * time.Second
	USER_INFO_CONCURRENCY  = 5
	GROUP_FETCH_MAX_ITER   = 500
	GROUP_INFO_CONCURRENCY = 5

	STORAGE_MODE  = "database"
	DATABASE_PATH = "donito_bot.sqlite3"
	USE_DATABASE  = strings.Contains(strings.ToLower(STORAGE_MODE), "database") || strings.Contains(strings.ToLower(STORAGE_MODE), "db") || strings.Contains(strings.ToLower(STORAGE_MODE), "sqlite")

	CONFIRM_WORDS     = map[string]bool{"بله": true, "آره": true, "اوکی": true, "تایید": true, "تائید": true, "باشه": true, "✅": true, "✓": true}
	REJECT_WORDS      = map[string]bool{"نه": true, "خیر": true, "رد": true, "لغو": true, "❌": true, "✗": true}
	TOTAL_STATS_WORDS = map[string]bool{"کل": true, "کلی": true, "all": true, "total": true}
	DUNYTO_VARIATIONS = map[string]bool{"دنیتو": true, "دونیتو": true, "دُنیتو": true}
	REPLY_DELETE_WORDS = map[string]bool{"حذف": true, "پاک": true, "حذف کن": true, "حذف کردن": true, "پاک کن": true, "پاک کردن": true}

	NAME_MAP = map[string]string{
		"فایتر": "Fighter", "اهورا": "Ahoora", "سبحان": "Geshtapo", "یحیی": "Yahya",
		"صبحان": "Geshtapo", "یکتا": "Yekta", "ساره": "Sareh", "مجی": "Moji",
		"گاد فادر": "GodFather", "گادفادر": "GodFather", "فرید": "Farid", "هانتر": "Hunter",
		"هانی": "Hani", "ماتادور": "Matador", "نیلوفر": "Niloofar", "هانیه": "Hanieh",
		"رضا": "Reza", "آتنا": "Atena", "اتنا": "Atena",
	}
	NAME_ID_MAP        = make(map[int]string)
	NEXT_BUILDER_ID    = 1
	NAME_BY_LEN        = make(map[int][]struct{ Name, Mapped string })
	_builder_lower_map = make(map[string]string)
	_builder_lock      sync.RWMutex
	_name_lock         sync.RWMutex

	FINGLISH_REPLACER = strings.NewReplacer(
		"ا", "a", "آ", "a", "ب", "b", "پ", "p", "ت", "t", "ث", "s", "ج", "j",
		"چ", "ch", "ح", "h", "خ", "kh", "د", "d", "ذ", "z", "ر", "r", "ز", "z",
		"ژ", "zh", "س", "s", "ش", "sh", "ص", "s", "ض", "z", "ط", "t", "ظ", "z",
		"ع", "a", "غ", "gh", "ف", "f", "ق", "gh", "ک", "k", "ك", "k", "گ", "g",
		"ل", "l", "م", "m", "ن", "n", "و", "oo", "ه", "h", "ی", "i", "ي", "i",
		"ئ", "e", "؟", "?", "،", ",", "؛", ";",
		"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4",
		"۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
		"َ", "a", "ُ", "o", "ِ", "e", "\u200c", "",
	)
	NORMALIZE_REPLACER = strings.NewReplacer("\u200c", " ", "ي", "ی", "ك", "ک")
	DIGIT_REPLACER     = strings.NewReplacer("۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9")
	CLEAN_NAME_RE      = regexp.MustCompile(`[\x00-\x1f\x7f-\x9f\[\]<>]`)

	START_REGEX               = regexp.MustCompile(`^(?!پایان[\s,،])([^,،]+)[,،]\s*([^,،]+?)(?:\s*[,،]\s*(فور))?\s*$`)
	END_REGEX                 = regexp.MustCompile(`^پایان[\s,،]+(.+?)(?:\s*[,،]?\s*(فور))?\s*$`)
	LIST_GAMES_REGEX          = regexp.MustCompile(`^(?:(?:لیست|نمایش)\s+)?بازی(?:\s+ها)?$`)
	DELETE_GAME_REGEX         = regexp.MustCompile(`^(?:حذف|پاک)(?:\s+کردن)?\s+بازی\s+(\d+)$`)
	DELETE_ALL_REGEX          = regexp.MustCompile(`^(?:حذف|پاک)(?:\s+کردن)?(?:\s+(?:تمام|همه|کل|کلیه|تمامی))?\s+بازی(?:\s+ها)?$`)
	AUTO_STATS_SET_REGEX      = regexp.MustCompile(`^(?:آمار|احصائیه)\s+(?:خودکار|اتوماتیک)\s+(?:در\s+)?ساعت\s+(\d{1,2}:\d{2})$`)
	AUTO_STATS_DISABLE_REGEX  = regexp.MustCompile(`^(?:لغو|حذف|غیرفعال|قطع)\s+(?:(?:آمار\s+(?:خودکار|اتوماتیک))|(?:(?:خودکار|اتوماتیک)\s+آمار))$`)
	SET_AUTO_DELETE_REGEX     = regexp.MustCompile(`^(?:تنظیم|تنطیم)\s+(?:حذف\s+)?خودکار\s+(\d+)$`)
	DISABLE_AUTO_DELETE_REGEX = regexp.MustCompile(`^(?:لغو|حذف|غیرفعال|قطع)\s+(?:حذف\s+)?خودکار$`)
	CLOSE_GROUP_REGEX         = regexp.MustCompile(`^(?:بستن|بسته|قفل)(?:\s+کردن)?(?:\s+(?:گروه|چت|گپ))?$`)
	OPEN_GROUP_REGEX          = regexp.MustCompile(`^(?:باز\s+کردن|وا\s+کردن|آزاد\s+کردن)(?:\s+(?:گروه|چت|گپ))?$`)
	ADD_ADMIN_REGEX           = regexp.MustCompile(`^(?:ادمین\s+کن|افزودن\s+ادمین|ادمین\s+کردن|ادمین\s+اضافه\s+کن)(?:\s+(\S+))?$`)
	REMOVE_ADMIN_REGEX        = regexp.MustCompile(`^(?:عزل\s+ادمین|حذف\s+ادمین|برداشتن\s+ادمین|عزل\s+کردن)(?:\s+(\S+))?$`)
	DELETE_MESSAGES_REGEX     = regexp.MustCompile(`^(?:حذف|پاک)(?:\s+کردن)?\s+پیام\s+(\d+)$`)
	BAN_REGEX                 = regexp.MustCompile(`^(?:بن(?:\s+کن)?|اخراج(?:\s+کن)?|حذف(?:\s+کن)?)(?:\s+(\S+))?$`)
	UNBAN_REGEX               = regexp.MustCompile(`^(?:آنبن(?:\s+کن)?|لغو\s+بن|لغو\s+اخراج|برگردان|لغو\s+کردن|لغو)(?:\s+(\S+))?$`)
	BLOCK_REGEX               = regexp.MustCompile(`^(?:بلاک|مسدود)(?:\s+کردن)?(?:\s+(\S+))?$`)
	UNBLOCK_REGEX             = regexp.MustCompile(`^(?:آنبلاک|آن\s*بلاک|رفع(?:\s+کردن)?(?:\s+بلاک)|لغو(?:\s+کردن)?(?:\s+بلاک)|حذف(?:\s+کردن)?(?:\s+بلاک))(?:\s+(\S+))?$`)
	SPECIAL_REGEX             = regexp.MustCompile(`^(?:ویژه|مخصوص|خاص)(?:\s+(\S+))?$`)
	OBJECT_ID_REGEX           = regexp.MustCompile(`^[ugc]\w+$`)
	STATS_BASIC_REGEX         = regexp.MustCompile(`^(?:آمار|stats|امار|احصائیه|وضعیت)(?!\s+اشتراک)(?:\s|$)`)
	STATS_TOTAL_REGEX         = regexp.MustCompile(`^(?:آمار|stats|امار|احصائیه|وضعیت)\s+(?:کل|کلی|all|total)(?:\s|$)`)
	HELP_REGEX                = regexp.MustCompile(`^(?:راهنما|help|کمک|راهنمایی|راهنمايي)(?:\s|$)`)
	STOP_REGEX                = regexp.MustCompile(`^!stop$`)
	START_BOT_REGEX           = regexp.MustCompile(`^!start$`)
	SET_BACKUP_TIME_REGEX     = regexp.MustCompile(`^(?:تنظیم|تنطیم)\s+(?:زمان\s+)?بکاپ\s+ساعت\s+(\d{1,2}:\d{2})$`)
	ADD_SUB_REGEX             = regexp.MustCompile(`^افزایش\s+اشتراک\s+(\d{4}[/-]\d{1,2}[/-]\d{1,2})$`)
	REDUCE_SUB_REGEX          = regexp.MustCompile(`^کاهش\s+اشتراک\s+(\d{4}[/-]\d{1,2}[/-]\d{1,2})$`)
	SUB_STATUS_REGEX          = regexp.MustCompile(`^وضعیت\s+اشتراک$`)
	CANCEL_SUB_REGEX          = regexp.MustCompile(`^لغو\s+اشتراک$`)
	PV_LIST_GROUPS_REGEX      = regexp.MustCompile(`^مدیریت\s+گروه(?:\s*ها)?$`)
	PV_ADD_GROUP_REGEX        = regexp.MustCompile(`^افزودن\s+گروه\s+([\s\S]+)$`)
	PV_LEAVE_GROUP_REGEX      = regexp.MustCompile(`^لفت\s+گروه\s+(\d+)$`)
	ADD_BUILDER_REGEX         = regexp.MustCompile(`^(?:افزودن|اضافه\s+کردن|ثبت)\s+سازنده\s+(.+?)\s*=\s*(.+?)\s*$`)
	REMOVE_BUILDER_REGEX      = regexp.MustCompile(`^(?:حذف|پاک)(?:\s+کردن)?\s+سازنده\s+(\d+)\s*$`)
	LIST_BUILDERS_REGEX       = regexp.MustCompile(`^(?:(?:لیست|نمایش)\s+سازنده(?:\s+ها)?)$`)
	DELETE_BUILDER_STATS_REGEX = regexp.MustCompile(`^(?:حذف|پاک)(?:\s+کردن)?\s+آمار\s+سازنده\s+(\d+)\s*$`)
	MERGE_BUILDER_REGEX       = regexp.MustCompile(`^ادغام\s+(?:سازنده\s+)?(\S+)\s+در\s+(\S+)\s*$`)
	DELETE_SCENARIO_STATS_REGEX = regexp.MustCompile(`^(?:حذف|پاک)(?:\s+کردن)?\s+آمار\s+سناریو\s+(\d+)\s*$`)
	MERGE_SCENARIO_REGEX      = regexp.MustCompile(`^ادغام\s+سناریو\s+(.+?)\s+در\s+(.+?)\s*$`)
	ADD_SUPER_ADMIN_REGEX     = regexp.MustCompile(`^(?:افزودن|اضافه\s+کردن|انتخاب)\s+جانشین(?:\s+(\S+))?$`)
	REMOVE_SUPER_ADMIN_REGEX  = regexp.MustCompile(`^(?:حذف|پاک|عزل)(?:\s+کردن)?\s+جانشین(?:\s+(\S+))?$`)
	LIST_SUPER_ADMINS_REGEX   = regexp.MustCompile(`^(?:(?:لیست|نمایش)\s+جانشین(?:\s+ها)?)$`)
	BROADCAST_ALL_REGEX       = regexp.MustCompile(`^فور\s+همگانی\s*([\s\S]*)$`)
	CANCEL_BROADCAST_REGEX    = regexp.MustCompile(`^(?:فور\s+لغو|لغو\s+فور|فورلغو|لغوفور)$`)
	TAG_ALL_REGEX             = regexp.MustCompile(`^تگ(?:\s+کردن)?$`)
	PIN_REGEX                 = regexp.MustCompile(`^(?:(?:پین|سنجاق)(?:\s+کن|کردن)?|(?:پین|سنجاق)\s+بزن)$`)
	UNPIN_REGEX               = regexp.MustCompile(`^(?:(?:آن|ان)\s*(?:پین|پن)|(?:آن|ان)پین|(?:لغو|حذف|پاک|برداشتن|رفع|آزاد|ازاد|باز)\s*(?:کردن)?\s*(?:پین|پن)|پین\s*(?:رو|را)?\s*(?:بردار|باز\s*کن|لغو\s*کن))(?:\s+کن)?$`)

	HELP_TEXT = "📖 راهنمای کامل ربات بنرگذار دُنیتو\n\n" +
		"🎮 شروع بازی (کاربران ویژه)\n" +
		"• یک عکس با کپشن زیر بفرستید:\n" +
		"  [سناریو] ، [سازنده] ، فور\n" +
		"• «فور» اختیاری است (فوروارد به گروه رسمی).\n" +
		"• اگر «دنیتو» در سناریو باشد، خودکار به کپشن اضافه می‌شود.\n" +
		"• مثال: مذاکره دنیتو ، هانتر ، فور\n\n" +
		"🆕 بنر با ریپلای (ادمین/ویژه)\n" +
		"• یک عکس بدون کپشن ارسال کنید، سپس روی آن ریپلای بزنید و دستور را بفرستید:\n" +
		"  مثال شروع: بازپرس دنیتو ، فایتر ، فور\n" +
		"  مثال پایان: پایان ، شهر ، فور\n" +
		"• مدت اعتبار پیش‌نمایش ۲۰ ثانیه است.\n" +
		"• برای هر بازی فقط یک پیش‌نمایش با همان اطلاعات پذیرفته می‌شود.\n" +
		"• هم فرستنده عکس و هم ریپلای‌زننده می‌توانند تأیید/رد کنند.\n\n" +
		"✅ تأیید / ❌ رد بنر\n" +
		"• بعد از ارسال عکس، پیش‌نمایشی دریافت می‌کنید.\n" +
		"• با ریپلای روی پیش‌نمایش یکی از کلمات زیر را بفرستید:\n" +
		"  تأیید: بله / آره / اوکی / تایید\n" +
		"  رد: نه / خیر / رد / لغو\n\n" +
		"🏁 پایان بازی\n" +
		"• یک عکس با کپشن زیر بفرستید:\n" +
		"  پایان [برنده] ، فور\n" +
		"• یا با ریپلای روی عکس: پایان [برنده] ، فور\n\n" +
		"📊 آمار و اطلاعات\n" +
		"• آمار روزانه گروه: آمار / stats\n" +
		"• آمار کلی گروه: آمار کل / آمار کلی\n" +
		"• آمار اختصاصی: ریپلای روی کاربر + آمار\n" +
		"• راهنما: راهنما / help\n" +
		"• لیست بازی‌های فعال: لیست بازی‌ها\n" +
		"• حذف یک بازی: حذف بازی 01\n" +
		"• حذف تمام بازی‌ها: حذف تمام بازی‌ها\n\n" +
		"💎 اشتراک (فقط سازنده اصلی)\n" +
		"• افزایش اشتراک: افزایش اشتراک 1405/07/11\n" +
		"• کاهش اشتراک: کاهش اشتراک 1405/07/11\n" +
		"• وضعیت اشتراک: وضعیت اشتراک\n" +
		"• لغو اشتراک: لغو اشتراک\n\n" +
		"🧑‍💻 مدیریت سازنده‌ها (ادمین‌ها)\n" +
		"• افزودن: افزودن سازنده فایتر = Fighter\n" +
		"• حذف از لیست: حذف سازنده 1\n" +
		"• نمایش لیست: لیست سازنده‌ها\n" +
		"• حذف از آمار: حذف آمار سازنده 18\n" +
		"• ادغام با نام: ادغام سازنده Mjtbi در Moji\n" +
		"• ادغام با شماره: ادغام سازنده 18 در 4\n\n" +
		"🎭 مدیریت سناریوها (ادمین‌ها)\n" +
		"• حذف از آمار: حذف آمار سناریو 3\n" +
		"• ادغام با نام: ادغام سناریو بازپرس در بازپرس ➪ دُنیتو\n" +
		"• ادغام با شماره: ادغام سناریو 2 در 1\n\n" +
		"🏷 تگ کردن اعضا (ادمین اصلی و جانشین‌ها)\n" +
		"• تگ کردن (بدون ریپلای): همه اعضا منشن می‌شوند\n" +
		"• تگ کردن (با ریپلای): منشن‌ها روی همان پیام ارسال می‌شوند\n\n" +
		"📌 پین و حذف پیام (ادمین‌ها)\n" +
		"• حذف با ریپلای: ریپلای + «حذف» یا «پاک»\n" +
		"• پین: ریپلای + «پین» یا «سنجاق»\n" +
		"• برداشتن پین: ریپلای + «انپین» / «آنپین» / «لغو پین»\n\n" +
		"📢 فور همگانی (ادمین اصلی و جانشین‌ها)\n" +
		"• ارسال به همه اعضا: فور همگانی متن پیام\n" +
		"• با ریپلای روی یک پیام: فور همگانی\n" +
		"• لغو ارسال: فور لغو یا لغو فور\n\n" +
		"👑 جانشین‌ها (فقط ادمین اصلی)\n" +
		"• افزودن جانشین: افزودن جانشین / افزودن جانشین u0xxx\n" +
		"• حذف جانشین: حذف جانشین / حذف جانشین u0xxx\n" +
		"• لیست جانشین‌ها: لیست جانشین‌ها\n\n" +
		"👑 دستورات مدیریت گروه (ادمین‌ها)\n" +
		"• قفل گروه: قفل / قفل گروه / بستن\n" +
		"• باز کردن گروه: بازکردن / باز کردن گروه / وا کردن\n\n" +
		"🔨 مدیریت کاربران (ادمین‌ها)\n" +
		"• ویژه کردن: ویژه / ویژه u0xxx\n" +
		"• بلاک کردن: بلاک / بلاک u0xxx / مسدود\n" +
		"• آنبلاک کردن: آنبلاک / رفع بلاک / آنبلاک u0xxx\n" +
		"• بن کردن: بن / اخراج\n" +
		"• آنبن + ارسال لینک: آنبن / برگردان / لغو بن\n\n" +
		"⚙️ مدیریت ادمین‌ها (ادمین اصلی و جانشین‌ها)\n" +
		"• افزودن ادمین: ادمین کن / افزودن ادمین u0xxx\n" +
		"• حذف ادمین: عزل ادمین / حذف ادمین\n\n" +
		"🧹 پاکسازی پیام‌ها (ادمین‌ها)\n" +
		"• حذف پیام 20 (حداکثر ۱۰۰۰ پیام آخر)\n\n" +
		"⏰ آمار خودکار (ادمین‌ها)\n" +
		"• تنظیم: آمار خودکار ساعت 00:00\n" +
		"• لغو: لغو آمار خودکار\n\n" +
		"⏱️ حذف خودکار پیام‌ها (ادمین‌ها)\n" +
		"• تنظیم: تنظیم حذف خودکار 15\n" +
		"• لغو: لغو حذف خودکار\n\n" +
		"📦 بکاپ خودکار (ادمین‌ها)\n" +
		"• تنظیم زمان بکاپ: تنظیم بکاپ ساعت 23:59\n\n" +
		"🛑 توقف و راه‌اندازی ربات (ادمین اصلی و جانشین‌ها)\n" +
		"• توقف: !stop\n" +
		"• راه‌اندازی: !start\n"

	PV_HELP_TEXT = "📖 راهنمای مدیریت ربات (پیوی سازنده اصلی)\n\n" +
		"📋 مدیریت گروه‌ها\n" +
		"• لیست گروه‌ها: مدیریت گروه ها\n" +
		"• افزودن گروه: افزودن گروه <link>\n" +
		"  مثال: افزودن گروه https://rubika.ir/joing/XXXX\n" +
		"• لفت دادن گروه با شماره: لفت گروه 1\n\n" +
		"💎 مدیریت اشتراک\n" +
		"• برای تنظیم اشتراک، دستور را در خود گروه ارسال کنید:\n" +
		"  افزایش اشتراک 1405/07/11\n" +
		"  کاهش اشتراک 1405/07/11\n" +
		"  لغو اشتراک\n" +
		"  وضعیت اشتراک\n\n" +
		"ℹ️ سایر دستورات مدیریتی در خود گروه‌ها اجرا می‌شوند."
)

type Game struct {
	ID              string
	StartMsgID      string
	ScenarioCaption string
	Builder         string
	CreatorGUID     string
	ShouldForward   bool
	StartTime       string
	StartMono       time.Time
	ForwardedMsgID  string
	TimeoutWarned   bool
	Closing         bool
	WarnTimer       *time.Timer
	CloseTimer      *time.Timer
}

type BannerData struct {
	ChatID          string
	FileInline      *events.Event
	Caption         string
	FileMeta        map[string]any
	ShouldForward   bool
	ScenarioCaption string
	BuilderEn       string
	Now             string
	CreatorGUID     string
	BuilderFa       string
	UserMsgID       string
	PreviewMsgID    string
	ExpireTimer     *time.Timer
	Signature       string
	AllowedUsers    map[string]bool
	CommandMsgID    string
}

type EndData struct {
	ChatID         string
	GameID         string
	FileInline     *events.Event
	Caption        string
	FileMeta       map[string]any
	ShouldForward  bool
	Winner         string
	Now            string
	CreatorGUID    string
	UserMsgID      string
	PreviewMsgID   string
	ExpireTimer    *time.Timer
	Signature      string
	AllowedUsers   map[string]bool
	CommandMsgID   string
}

type Stats struct {
	TotalGames         int
	DailyGames         int
	BuilderStats       map[string]int
	ScenarioStats      map[string]int
	DailyBuilderStats  map[string]int
	DailyScenarioStats map[string]int
	UserStats          map[string]*UserStat
	DailyUserStats     map[string]*UserStat
	LastReset          string
}

type UserStat struct {
	Games     int
	Scenarios map[string]int
}

var (
	acc *account.Account

	activeGames       = make(map[string]map[string]*Game)
	activeGamesMu     sync.RWMutex
	pendingBanners    = make(map[string]*BannerData)
	pendingBannersMu  sync.RWMutex
	pendingByPreview  = make(map[string]string)
	pendingByPreviewMu sync.RWMutex
	pendingEnds       = make(map[string]*EndData)
	pendingEndsMu     sync.RWMutex
	pendingEndByPreview = make(map[string]string)
	pendingEndByPreviewMu sync.RWMutex
	startMessageIndex = make(map[string]string)
	startMessageIndexMu sync.RWMutex
	processingPreviews = make(map[string]bool)
	processingPreviewsMu sync.Mutex
	botGroups         = make(map[string]map[string]any)
	botGroupsMu       sync.RWMutex
	groupSubscriptions = make(map[string]time.Time)
	groupSubscriptionsMu sync.RWMutex
	lastGroupDisplay []struct {
		GUID        string
		Title       string
		MemberCount int
	}
	lastGroupDisplayMu      sync.RWMutex
	lastGroupDisplaySavedAt string

	gameStats      = NewStats()
	groupGameStats = make(map[string]*Stats)
	statsMu        sync.RWMutex
	blockedUsers   = make(map[string]bool)
	blockedUsersMu sync.RWMutex
	specialUsers   = make(map[string]bool)
	specialUsersMu sync.RWMutex
	adminGuids     = map[string]bool{MAIN_ADMIN_GUID: true}
	adminGuidsMu   sync.RWMutex
	superAdminGuids = make(map[string]bool)
	superAdminGuidsMu sync.RWMutex

	autoStatsEnabled  = false
	autoStatsTime     = ""
	autoStatsHandle   *time.Timer
	lastAutoStatsDate = ""
	autoStatsMu       sync.Mutex

	forwardQueue          chan map[string]any
	activeEvent           = make(chan struct{}, 1)
	dbConnection          *sql.DB
	dbQueue               chan func(*sql.Tx)
	backupTime            = "23:59"
	backupMu              sync.Mutex
	selfGUID              string
	selfGUIDMu            sync.Mutex
	activeBroadcastCancel context.CancelFunc
	activeBroadcastMu     sync.Mutex
)

func NewStats() *Stats {
	return &Stats{
		BuilderStats:       make(map[string]int),
		ScenarioStats:      make(map[string]int),
		DailyBuilderStats:  make(map[string]int),
		DailyScenarioStats: make(map[string]int),
		UserStats:          make(map[string]*UserStat),
		DailyUserStats:     make(map[string]*UserStat),
	}
}

type RingBufferCache struct {
	buffer []string
	size   int
	index  int
	set    map[string]bool
	mu     sync.Mutex
}

func NewRingBufferCache(size int) *RingBufferCache {
	return &RingBufferCache{
		buffer: make([]string, size),
		size:   size,
		set:    make(map[string]bool),
	}
}

func (c *RingBufferCache) Consume(item string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.set[item] {
		return false
	}
	old := c.buffer[c.index]
	if old != "" {
		delete(c.set, old)
	}
	c.buffer[c.index] = item
	c.set[item] = true
	c.index = (c.index + 1) % c.size
	return true
}

var processedMessageIDs = NewRingBufferCache(PROCESSED_CACHE_LIMIT)

func rebuildNameIndex() {
	_name_lock.Lock()
	defer _name_lock.Unlock()
	index := make(map[int][]struct{ Name, Mapped string })
	for name, mapped := range NAME_MAP {
		l := len(name)
		index[l] = append(index[l], struct{ Name, Mapped string }{name, mapped})
	}
	NAME_BY_LEN = index
}

func rebuildBuilderLowerMap() {
	_builder_lock.Lock()
	defer _builder_lock.Unlock()
	_builder_lower_map = make(map[string]string)
	statsMu.RLock()
	for k := range gameStats.BuilderStats {
		_builder_lower_map[strings.ToLower(k)] = k
	}
	statsMu.RUnlock()
	for _, v := range NAME_MAP {
		_builder_lower_map[strings.ToLower(v)] = v
	}
}

func GregorianToJalali(gy, gm, gd int) (jy, jm, jd int) {
	g_d_m := []int{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}
	if gy > 1600 {
		jy = 979
		gy -= 1600
	} else {
		jy = 0
		gy -= 621
	}
	gy2 := gy + 1
	if gm > 2 {
		gy2++
	}
	days := (365 * gy) + ((gy2 + 3) / 4) - ((gy2 + 99) / 100) + ((gy2 + 399) / 400) - 80 + gd + g_d_m[gm-1]
	jy += 33 * (days / 12053)
	days %= 12053
	jy += 4 * (days / 1461)
	days %= 1461
	if days > 365 {
		jy += (days - 1) / 365
		days = (days - 1) % 365
	}
	if days < 186 {
		jm = 1 + (days / 31)
		jd = 1 + (days % 31)
	} else {
		jm = 7 + ((days - 186) / 30)
		jd = 1 + ((days - 186) % 30)
	}
	return
}

func JalaliToGregorian(jy, jm, jd int) (gy, gm, gd int) {
	jy += 1595
	days := -355668 + (365 * jy) + ((jy / 33) * 8) + (((jy % 33) + 3) / 4) + jd
	if jm < 7 {
		days += (jm - 1) * 31
	} else {
		days += ((jm - 7) * 30) + 186
	}
	gy = 400 * (days / 146097)
	days %= 146097
	if days > 36524 {
		days--
		gy += 100 * (days / 36524)
		days %= 36524
		if days >= 365 {
			days++
		}
	}
	gy += 4 * (days / 1461)
	days %= 1461
	if days > 365 {
		gy += (days - 1) / 365
		days = (days - 1) % 365
	}
	gd = days + 1
	sal_a := []int{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}
	var i int
	for i = 0; i < 12; i++ {
		if gd <= sal_a[i] {
			break
		}
	}
	if i == 0 {
		i = 1
	}
	gm = i
	gd -= sal_a[i-1]
	if gm > 2 && ((gy%4 == 0 && gy%100 != 0) || (gy%400 == 0)) {
		gd--
	}
	return
}

func GetShamsiNow() string {
	now := time.Now()
	jy, jm, jd := GregorianToJalali(now.Year(), int(now.Month()), now.Day())
	return fmt.Sprintf("%04d/%02d/%02d - %02d:%02d", jy, jm, jd, now.Hour(), now.Minute())
}

func GetShamsiDate() string {
	now := time.Now()
	jy, jm, jd := GregorianToJalali(now.Year(), int(now.Month()), now.Day())
	return fmt.Sprintf("%04d/%02d/%02d", jy, jm, jd)
}

func ParseShamsiDate(value string) (time.Time, bool) {
	cleaned := strings.ReplaceAll(value, "-", "/")
	parts := strings.Split(cleaned, "/")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	jy, _ := strconv.Atoi(parts[0])
	jm, _ := strconv.Atoi(parts[1])
	jd, _ := strconv.Atoi(parts[2])
	gy, gm, gd := JalaliToGregorian(jy, jm, jd)
	return time.Date(gy, time.Month(gm), gd, 0, 0, 0, 0, time.Local), true
}

func NormalizeText(text string) string {
	if text == "" {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(NORMALIZE_REPLACER.Replace(text)))
}

func FirstWord(text string) string {
	if text == "" {
		return ""
	}
	parts := strings.SplitN(text, " ", 2)
	return strings.Trim(parts[0], ".,!?؟،؛:()")
}

func ToFinglish(text string) string {
	replaced := FINGLISH_REPLACER.Replace(text)
	words := strings.Fields(replaced)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, "")
}

func OneEditMatch(a, b string) bool {
	la, lb := len(a), len(b)
	if la == lb {
		diff := 0
		for i := 0; i < la; i++ {
			if a[i] != b[i] {
				diff++
				if diff > 1 {
					return false
				}
			}
		}
		return true
	}
	if la+1 == lb {
		a, b = b, a
		la, lb = lb, la
	} else if lb+1 != la {
		return false
	}
	ia, ib, diff := 0, 0, 0
	for ia < la && ib < lb {
		if a[ia] != b[ib] {
			diff++
			if diff > 1 {
				return false
			}
			ia++
		} else {
			ia++
			ib++
		}
	}
	return true
}

func NormalizeBuilderName(raw string) (string, string) {
	key := strings.Join(strings.Fields(NORMALIZE_REPLACER.Replace(raw)), " ")
	_name_lock.RLock()
	defer _name_lock.RUnlock()
	if mapped, ok := NAME_MAP[key]; ok {
		return mapped, ""
	}
	l := len(key)
	for _, length := range []int{l, l - 1, l + 1} {
		if candidates, ok := NAME_BY_LEN[length]; ok {
			for _, c := range candidates {
				if OneEditMatch(key, c.Name) {
					return c.Mapped, c.Name
				}
			}
		}
	}
	return ToFinglish(key), ""
}

func CanonicalizeBuilderName(name string) string {
	if name == "" || len(name) < 3 {
		return name
	}
	statsMu.RLock()
	defer statsMu.RUnlock()
	if _, ok := gameStats.BuilderStats[name]; ok {
		return name
	}
	low := strings.ToLower(name)
	_builder_lock.RLock()
	if mapped, ok := _builder_lower_map[low]; ok {
		_builder_lock.RUnlock()
		return mapped
	}
	_builder_lock.RUnlock()

	if len(name) >= 4 {
		lowLen := len(name)
		var match string
		count := 0
		for c := range gameStats.BuilderStats {
			if abs(len(c)-lowLen) <= 1 && OneEditMatch(low, strings.ToLower(c)) {
				match = c
				count++
				if count > 1 {
					break
				}
			}
		}
		if count == 1 {
			return match
		}
		for _, c := range NAME_MAP {
			if abs(len(c)-lowLen) <= 1 && OneEditMatch(low, strings.ToLower(c)) {
				match = c
				count++
				if count > 1 {
					break
				}
			}
		}
		if count == 1 {
			return match
		}
	}
	return name
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func EnsureUserStats(container map[string]*UserStat, guid string) *UserStat {
	u, ok := container[guid]
	if !ok {
		u = &UserStat{Games: 0, Scenarios: make(map[string]int)}
		container[guid] = u
	}
	return u
}

func ResetStatsDaily(s *Stats) {
	s.DailyGames = 0
	s.DailyBuilderStats = make(map[string]int)
	s.DailyScenarioStats = make(map[string]int)
	s.DailyUserStats = make(map[string]*UserStat)
}

func EnsureDailyReset() {
	today := GetShamsiDate()
	statsMu.Lock()
	defer statsMu.Unlock()
	if gameStats.LastReset != today {
		ResetStatsDaily(gameStats)
		gameStats.LastReset = today
		for _, s := range groupGameStats {
			ResetStatsDaily(s)
			s.LastReset = today
		}
		if USE_DATABASE {
			EnqueueDB(func(tx *sql.Tx) {
				tx.Exec("DELETE FROM daily_builder_stats")
				tx.Exec("DELETE FROM daily_scenario_stats")
				tx.Exec("DELETE FROM daily_user_total")
				tx.Exec("DELETE FROM daily_user_scenario")
				tx.Exec("DELETE FROM group_daily_builder_stats")
				tx.Exec("DELETE FROM group_daily_scenario_stats")
				tx.Exec("DELETE FROM group_daily_user_total")
				tx.Exec("DELETE FROM group_daily_user_scenario")
				tx.Exec("UPDATE meta SET value='0' WHERE key='daily_games'")
				tx.Exec("UPDATE group_meta SET value='0' WHERE key='daily_games'")
				tx.Exec("INSERT OR REPLACE INTO meta(key, value) VALUES('last_reset', ?)", today)
			})
		}
	}
}

func IncrementStats(s *Stats, userGuids []string, builder, scenario string) {
	s.TotalGames++
	s.DailyGames++
	_builder_lock.Lock()
	_builder_lower_map[strings.ToLower(builder)] = builder
	_builder_lock.Unlock()
	s.BuilderStats[builder]++
	s.DailyBuilderStats[builder]++
	s.ScenarioStats[scenario]++
	s.DailyScenarioStats[scenario]++
	for _, ug := range userGuids {
		u := EnsureUserStats(s.UserStats, ug)
		u.Games++
		u.Scenarios[scenario]++
		du := EnsureUserStats(s.DailyUserStats, ug)
		du.Games++
		du.Scenarios[scenario]++
	}
}

func EnsureGroupStats(guid string) *Stats {
	s, ok := groupGameStats[guid]
	if !ok {
		s = NewStats()
		s.LastReset = GetShamsiDate()
		groupGameStats[guid] = s
	}
	return s
}

func UpdateStatsMulti(builder, scenario string, userGuids []string, groupGUID string) {
	EnsureDailyReset()
	builder = CanonicalizeBuilderName(builder)
	var cleanGuids []string
	seen := make(map[string]bool)
	for _, g := range userGuids {
		if g != "" && !seen[g] {
			seen[g] = true
			cleanGuids = append(cleanGuids, g)
		}
	}
	if len(cleanGuids) == 0 {
		return
	}
	statsMu.Lock()
	IncrementStats(gameStats, cleanGuids, builder, scenario)
	if groupGUID != "" {
		gs := EnsureGroupStats(groupGUID)
		IncrementStats(gs, cleanGuids, builder, scenario)
		if gs.LastReset == "" {
			gs.LastReset = GetShamsiDate()
		}
	}
	statsMu.Unlock()

	if USE_DATABASE {
		EnqueueDB(func(tx *sql.Tx) {
			ApplyStatsUpsert(tx, cleanGuids, builder, scenario, "", "")
			if groupGUID != "" {
				ApplyStatsUpsert(tx, cleanGuids, builder, scenario, "group", groupGUID)
			}
		})
	}
}

func ApplyStatsUpsert(tx *sql.Tx, guids []string, builder, scenario, prefix, groupGUID string) {
	if prefix == "" {
		tx.Exec("INSERT INTO meta(key, value) VALUES('total_games', 1) ON CONFLICT(key) DO UPDATE SET value = value + 1")
		tx.Exec("INSERT INTO meta(key, value) VALUES('daily_games', 1) ON CONFLICT(key) DO UPDATE SET value = value + 1")
		tx.Exec("INSERT INTO builder_stats(name, count) VALUES(?, 1) ON CONFLICT(name) DO UPDATE SET count = count + 1", builder)
		tx.Exec("INSERT INTO scenario_stats(name, count) VALUES(?, 1) ON CONFLICT(name) DO UPDATE SET count = count + 1", scenario)
		tx.Exec("INSERT INTO daily_builder_stats(name, count) VALUES(?, 1) ON CONFLICT(name) DO UPDATE SET count = count + 1", builder)
		tx.Exec("INSERT INTO daily_scenario_stats(name, count) VALUES(?, 1) ON CONFLICT(name) DO UPDATE SET count = count + 1", scenario)
		for _, ug := range guids {
			tx.Exec("INSERT INTO user_total(user_guid, games) VALUES(?, 1) ON CONFLICT(user_guid) DO UPDATE SET games = games + 1", ug)
			tx.Exec("INSERT INTO user_scenario(user_guid, scenario, count) VALUES(?, ?, 1) ON CONFLICT(user_guid, scenario) DO UPDATE SET count = count + 1", ug, scenario)
			tx.Exec("INSERT INTO daily_user_total(user_guid, games) VALUES(?, 1) ON CONFLICT(user_guid) DO UPDATE SET games = games + 1", ug)
			tx.Exec("INSERT INTO daily_user_scenario(user_guid, scenario, count) VALUES(?, ?, 1) ON CONFLICT(user_guid, scenario) DO UPDATE SET count = count + 1", ug, scenario)
		}
	} else {
		tx.Exec("INSERT INTO group_meta(group_guid, key, value) VALUES(?, 'total_games', 1) ON CONFLICT(group_guid, key) DO UPDATE SET value = CAST(value AS INTEGER) + 1", groupGUID)
		tx.Exec("INSERT INTO group_meta(group_guid, key, value) VALUES(?, 'daily_games', 1) ON CONFLICT(group_guid, key) DO UPDATE SET value = CAST(value AS INTEGER) + 1", groupGUID)
		tx.Exec("INSERT INTO group_builder_stats(group_guid, name, count) VALUES(?, ?, 1) ON CONFLICT(group_guid, name) DO UPDATE SET count = count + 1", groupGUID, builder)
		tx.Exec("INSERT INTO group_scenario_stats(group_guid, name, count) VALUES(?, ?, 1) ON CONFLICT(group_guid, name) DO UPDATE SET count = count + 1", groupGUID, scenario)
		tx.Exec("INSERT INTO group_daily_builder_stats(group_guid, name, count) VALUES(?, ?, 1) ON CONFLICT(group_guid, name) DO UPDATE SET count = count + 1", groupGUID, builder)
		tx.Exec("INSERT INTO group_daily_scenario_stats(group_guid, name, count) VALUES(?, ?, 1) ON CONFLICT(group_guid, name) DO UPDATE SET count = count + 1", groupGUID, scenario)
		for _, ug := range guids {
			tx.Exec("INSERT INTO group_user_total(group_guid, user_guid, games) VALUES(?, ?, 1) ON CONFLICT(group_guid, user_guid) DO UPDATE SET games = games + 1", groupGUID, ug)
			tx.Exec("INSERT INTO group_user_scenario(group_guid, user_guid, scenario, count) VALUES(?, ?, ?, 1) ON CONFLICT(group_guid, user_guid, scenario) DO UPDATE SET count = count + 1", groupGUID, ug, scenario)
			tx.Exec("INSERT INTO group_daily_user_total(group_guid, user_guid, games) VALUES(?, ?, 1) ON CONFLICT(group_guid, user_guid) DO UPDATE SET games = games + 1", groupGUID, ug)
			tx.Exec("INSERT INTO group_daily_user_scenario(group_guid, user_guid, scenario, count) VALUES(?, ?, ?, 1) ON CONFLICT(group_guid, user_guid, scenario) DO UPDATE SET count = count + 1", groupGUID, ug, scenario)
		}
	}
}

func EnqueueDB(fn func(*sql.Tx)) {
	if USE_DATABASE && dbQueue != nil {
		select {
		case dbQueue <- fn:
		default:
		}
	}
}

func InitDB() {
	if !USE_DATABASE {
		_builder_lock.Lock()
		NAME_ID_MAP = make(map[int]string)
		NEXT_BUILDER_ID = 1
		for fa := range NAME_MAP {
			NAME_ID_MAP[NEXT_BUILDER_ID] = fa
			NEXT_BUILDER_ID++
		}
		rebuildBuilderLowerMap()
		_builder_lock.Unlock()
		return
	}
	var err error
	dbConnection, err = sql.Open("sqlite", DATABASE_PATH)
	if err != nil {
		log.Printf("DB open error: %v, falling back to memory", err)
		USE_DATABASE = false
		return
	}
	dbConnection.SetMaxOpenConns(1)
	dbConnection.Exec("PRAGMA journal_mode=WAL")
	dbConnection.Exec("PRAGMA synchronous=NORMAL")
	dbConnection.Exec("PRAGMA temp_store=MEMORY")
	dbConnection.Exec("PRAGMA cache_size=-4000")
	dbConnection.Exec("PRAGMA busy_timeout=5000")

	createTables()
	loadDB()
	restoreGroupDisplay()
	resetDailyDB()

	dbQueue = make(chan func(*sql.Tx), 1000)
	go dbWriter()
}

func createTables() {
	_, err := dbConnection.Exec(`
		CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		CREATE TABLE IF NOT EXISTS builder_stats (name TEXT PRIMARY KEY, count INTEGER NOT NULL);
		CREATE TABLE IF NOT EXISTS scenario_stats (name TEXT PRIMARY KEY, count INTEGER NOT NULL);
		CREATE TABLE IF NOT EXISTS daily_builder_stats (name TEXT PRIMARY KEY, count INTEGER NOT NULL);
		CREATE TABLE IF NOT EXISTS daily_scenario_stats (name TEXT PRIMARY KEY, count INTEGER NOT NULL);
		CREATE TABLE IF NOT EXISTS user_total (user_guid TEXT PRIMARY KEY, games INTEGER NOT NULL);
		CREATE TABLE IF NOT EXISTS user_scenario (user_guid TEXT NOT NULL, scenario TEXT NOT NULL, count INTEGER NOT NULL, PRIMARY KEY (user_guid, scenario));
		CREATE TABLE IF NOT EXISTS daily_user_total (user_guid TEXT PRIMARY KEY, games INTEGER NOT NULL);
		CREATE TABLE IF NOT EXISTS daily_user_scenario (user_guid TEXT NOT NULL, scenario TEXT NOT NULL, count INTEGER NOT NULL, PRIMARY KEY (user_guid, scenario));
		CREATE TABLE IF NOT EXISTS access_control (guid TEXT PRIMARY KEY, type TEXT NOT NULL CHECK(type IN ('admin', 'blocked', 'special')));
		CREATE TABLE IF NOT EXISTS builders (id INTEGER PRIMARY KEY AUTOINCREMENT, fa TEXT UNIQUE NOT NULL, en TEXT NOT NULL);
		CREATE TABLE IF NOT EXISTS super_admins (guid TEXT PRIMARY KEY);
		CREATE TABLE IF NOT EXISTS bot_groups (group_guid TEXT PRIMARY KEY, title TEXT, member_count INTEGER DEFAULT 0, subscription_end TEXT);
		CREATE TABLE IF NOT EXISTS group_builder_stats (group_guid TEXT, name TEXT, count INTEGER NOT NULL, PRIMARY KEY (group_guid, name));
		CREATE TABLE IF NOT EXISTS group_scenario_stats (group_guid TEXT, name TEXT, count INTEGER NOT NULL, PRIMARY KEY (group_guid, name));
		CREATE TABLE IF NOT EXISTS group_daily_builder_stats (group_guid TEXT, name TEXT, count INTEGER NOT NULL, PRIMARY KEY (group_guid, name));
		CREATE TABLE IF NOT EXISTS group_daily_scenario_stats (group_guid TEXT, name TEXT, count INTEGER NOT NULL, PRIMARY KEY (group_guid, name));
		CREATE TABLE IF NOT EXISTS group_user_total (group_guid TEXT, user_guid TEXT, games INTEGER NOT NULL, PRIMARY KEY (group_guid, user_guid));
		CREATE TABLE IF NOT EXISTS group_user_scenario (group_guid TEXT, user_guid TEXT, scenario TEXT, count INTEGER NOT NULL, PRIMARY KEY (group_guid, user_guid, scenario));
		CREATE TABLE IF NOT EXISTS group_daily_user_total (group_guid TEXT, user_guid TEXT, games INTEGER NOT NULL, PRIMARY KEY (group_guid, user_guid));
		CREATE TABLE IF NOT EXISTS group_daily_user_scenario (group_guid TEXT, user_guid TEXT, scenario TEXT, count INTEGER NOT NULL, PRIMARY KEY (group_guid, user_guid, scenario));
		CREATE TABLE IF NOT EXISTS group_meta (group_guid TEXT, key TEXT, value TEXT, PRIMARY KEY (group_guid, key));
	`)
	if err != nil {
		log.Printf("createTables error: %v", err)
	}
}

func resetDailyDB() {
	today := GetShamsiDate()
	EnqueueDB(func(tx *sql.Tx) {
		tx.Exec("DELETE FROM daily_builder_stats")
		tx.Exec("DELETE FROM daily_scenario_stats")
		tx.Exec("DELETE FROM daily_user_total")
		tx.Exec("DELETE FROM daily_user_scenario")
		tx.Exec("DELETE FROM group_daily_builder_stats")
		tx.Exec("DELETE FROM group_daily_scenario_stats")
		tx.Exec("DELETE FROM group_daily_user_total")
		tx.Exec("DELETE FROM group_daily_user_scenario")
		tx.Exec("UPDATE meta SET value='0' WHERE key='daily_games'")
		tx.Exec("UPDATE group_meta SET value='0' WHERE key='daily_games'")
		tx.Exec("INSERT OR REPLACE INTO meta(key, value) VALUES('last_reset', ?)", today)
	})
}

func dbWriter() {
	for fn := range dbQueue {
		tx, err := dbConnection.Begin()
		if err == nil {
			fn(tx)
			tx.Commit()
		}
	}
}

func loadDB() {
	rows, _ := dbConnection.Query("SELECT key, value FROM meta")
	statsMu.Lock()
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		if k == "total_games" {
			gameStats.TotalGames, _ = strconv.Atoi(v)
		} else if k == "daily_games" {
			gameStats.DailyGames, _ = strconv.Atoi(v)
		} else if k == "last_reset" {
			gameStats.LastReset = v
		} else if k == "main_admin_guid" {
			adminGuidsMu.Lock()
			MAIN_ADMIN_GUID = v
			adminGuids[v] = true
			adminGuidsMu.Unlock()
		}
	}
	rows.Close()

	rows, _ = dbConnection.Query("SELECT name, count FROM builder_stats")
	for rows.Next() {
		var n string
		var c int
		rows.Scan(&n, &c)
		gameStats.BuilderStats[n] = c
	}
	rows.Close()

	rows, _ = dbConnection.Query("SELECT name, count FROM scenario_stats")
	for rows.Next() {
		var n string
		var c int
		rows.Scan(&n, &c)
		gameStats.ScenarioStats[n] = c
	}
	rows.Close()

	rows, _ = dbConnection.Query("SELECT user_guid, games FROM user_total")
	for rows.Next() {
		var u string
		var g int
		rows.Scan(&u, &g)
		EnsureUserStats(gameStats.UserStats, u).Games = g
	}
	rows.Close()

	rows, _ = dbConnection.Query("SELECT user_guid, scenario, count FROM user_scenario")
	for rows.Next() {
		var u, s string
		var c int
		rows.Scan(&u, &s, &c)
		EnsureUserStats(gameStats.UserStats, u).Scenarios[s] = c
	}
	rows.Close()
	statsMu.Unlock()

	rows, _ = dbConnection.Query("SELECT guid, type FROM access_control")
	for rows.Next() {
		var g, t string
		rows.Scan(&g, &t)
		if t == "admin" {
			adminGuidsMu.Lock()
			adminGuids[g] = true
			adminGuidsMu.Unlock()
		} else if t == "blocked" {
			blockedUsersMu.Lock()
			blockedUsers[g] = true
			blockedUsersMu.Unlock()
		} else if t == "special" {
			specialUsersMu.Lock()
			specialUsers[g] = true
			specialUsersMu.Unlock()
		}
	}
	rows.Close()

	rows, _ = dbConnection.Query("SELECT guid FROM super_admins")
	for rows.Next() {
		var g string
		rows.Scan(&g)
		superAdminGuidsMu.Lock()
		superAdminGuids[g] = true
		superAdminGuidsMu.Unlock()
		adminGuidsMu.Lock()
		adminGuids[g] = true
		adminGuidsMu.Unlock()
	}
	rows.Close()

	rows, _ = dbConnection.Query("SELECT group_guid, title, member_count, subscription_end FROM bot_groups")
	for rows.Next() {
		var g, t, sub sql.NullString
		var c sql.NullInt64
		rows.Scan(&g, &t, &c, &sub)
		botGroupsMu.Lock()
		botGroups[g.String] = map[string]any{"title": t.String, "member_count": c.Int64}
		botGroupsMu.Unlock()
		if sub.Valid && sub.String != "" {
			if parsed, ok := ParseShamsiDate(sub.String); ok {
				groupSubscriptionsMu.Lock()
				groupSubscriptions[g.String] = parsed
				groupSubscriptionsMu.Unlock()
			}
		}
	}
	rows.Close()

	rows, _ = dbConnection.Query("SELECT id, fa, en FROM builders")
	count := 0
	for rows.Next() {
		var id int
		var fa, en string
		rows.Scan(&id, &fa, &en)
		_name_lock.Lock()
		NAME_MAP[fa] = en
		NAME_ID_MAP[id] = fa
		if id >= NEXT_BUILDER_ID {
			NEXT_BUILDER_ID = id + 1
		}
		_name_lock.Unlock()
		count++
	}
	rows.Close()
	rebuildNameIndex()
	rebuildBuilderLowerMap()
}

func restoreGroupDisplay() {
	var payload string
	err := dbConnection.QueryRow("SELECT value FROM meta WHERE key='last_group_display'").Scan(&payload)
	if err != nil {
		return
	}
	var data [][]any
	json.Unmarshal([]byte(payload), &data)
	lastGroupDisplayMu.Lock()
	for _, item := range data {
		if len(item) >= 3 {
			g, _ := item[0].(string)
			t, _ := item[1].(string)
			c, _ := item[2].(float64)
			lastGroupDisplay = append(lastGroupDisplay, struct {
				GUID        string
				Title       string
				MemberCount int
			}{g, t, int(c)})
		}
	}
	dbConnection.QueryRow("SELECT value FROM meta WHERE key='last_group_display_saved_at'").Scan(&lastGroupDisplaySavedAt)
	lastGroupDisplayMu.Unlock()
}

func PersistGroupDisplay() {
	if !USE_DATABASE {
		return
	}
	lastGroupDisplayMu.RLock()
	var data [][]any
	for _, item := range lastGroupDisplay {
		data = append(data, []any{item.GUID, item.Title, item.MemberCount})
	}
	savedAt := lastGroupDisplaySavedAt
	lastGroupDisplayMu.RUnlock()
	b, _ := json.Marshal(data)
	EnqueueDB(func(tx *sql.Tx) {
		tx.Exec("INSERT OR REPLACE INTO meta(key, value) VALUES('last_group_display', ?)", string(b))
		tx.Exec("INSERT OR REPLACE INTO meta(key, value) VALUES('last_group_display_saved_at', ?)", savedAt)
	})
}

func IsUserAllowed(e *events.Event) bool {
	author := e.AuthorGUID()
	adminGuidsMu.RLock()
	superAdminGuidsMu.RLock()
	isAdmin := adminGuids[author] || superAdminGuids[author]
	adminGuidsMu.RUnlock()
	superAdminGuidsMu.RUnlock()
	if isAdmin {
		return true
	}
	if !IS_BOT_ACTIVE {
		return false
	}
	blockedUsersMu.RLock()
	blocked := blockedUsers[author]
	blockedUsersMu.RUnlock()
	if blocked {
		return false
	}
	specialUsersMu.RLock()
	isSpecial := specialUsers[author]
	hasSpecials := len(specialUsers) > 0
	specialUsersMu.RUnlock()
	if hasSpecials && !isSpecial {
		return false
	}
	return true
}

func IsGroupSubscribed(guid string) bool {
	groupSubscriptionsMu.RLock()
	end, ok := groupSubscriptions[guid]
	groupSubscriptionsMu.RUnlock()
	if !ok {
		return false
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	return !today.After(end)
}

func DaysRemainingSubscription(guid string) int {
	groupSubscriptionsMu.RLock()
	end, ok := groupSubscriptions[guid]
	groupSubscriptionsMu.RUnlock()
	if !ok {
		return -1
	}
	return int(time.Until(end).Hours() / 24)
}

func ExtractMessageID(e *events.Event) string {
	return e.Str("message_id")
}

func SanitizeMdName(name string) string {
	clean := CLEAN_NAME_RE.ReplaceAllString(name, "")
	if clean == "" {
		return "کاربر"
	}
	return strings.TrimSpace(clean)
}

func CleanNamePart(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(CLEAN_NAME_RE.ReplaceAllString(value, ""), "\u200c", " "), "\ufeff", " "))
}

func FormatUserName(info *events.Event) string {
	if info == nil {
		return "ناشناس"
	}
	user := info.EventOf("user")
	if user == nil {
		return "ناشناس"
	}
	fn := CleanNamePart(user.Str("first_name"))
	ln := CleanNamePart(user.Str("last_name"))
	name := strings.TrimSpace(fn + " " + ln)
	if name == "" {
		un := user.Str("username")
		if un != "" {
			name = "@" + un
		}
	}
	if name == "" {
		return "ناشناس"
	}
	return name
}

func SendMessageSafe(ctx context.Context, chatGUID, text string, replyTo string, autoDelete int, parseMode enums.ParseMode) {
	opts := account.SendOptions{
		ReplyToMessageID: replyTo,
		ParseMode:        parseMode,
	}
	if autoDelete > 0 {
		opts.AutoDelete = time.Duration(autoDelete) * time.Second
	} else if autoDelete == 0 {
		opts.AutoDelete = time.Hour * 24 * 365
	} else {
		opts.AutoDelete = time.Duration(AUTO_DELETE_SECONDS) * time.Second
	}

	_, err := acc.SendMessage(ctx, chatGUID, text, opts)
	if err != nil && parseMode != enums.ParseModeNone {
		opts.ParseMode = enums.ParseModeNone
		acc.SendMessage(ctx, chatGUID, text, opts)
	}
}

func RespondAndCleanup(ctx context.Context, e *events.Event, text string, autoDelete int) {
	chatGUID := e.ObjectGUID()
	msgID := e.MessageID()
	SendMessageSafe(ctx, chatGUID, text, msgID, autoDelete, enums.ParseModeNone)
	if strings.HasPrefix(chatGUID, "g") {
		acc.DeleteMessages(ctx, chatGUID, []string{msgID})
	}
}

func DeleteMessagesSafe(ctx context.Context, chatGUID string, msgIDs []string) {
	if len(msgIDs) > 0 {
		acc.DeleteMessages(ctx, chatGUID, msgIDs)
	}
}

func StartGameTimer(chatID string, gameID string) {
	activeGamesMu.RLock()
	game := activeGames[chatID][gameID]
	activeGamesMu.RUnlock()
	if game == nil {
		return
	}

	game.WarnTimer = time.AfterFunc(time.Duration(WARNING_AFTER_HOURS)*time.Hour, func() {
		activeGamesMu.Lock()
		if g, ok := activeGames[chatID][gameID]; ok && !g.TimeoutWarned {
			g.TimeoutWarned = true
			text := fmt.Sprintf("⚠️ هشدار زمان بازی\n\nبازی شماره `%s`\nسناریو: %s\nمدت زمان: %.1f ساعت\n\nلطفاً نسبت به پایان بازی اقدام کنید.", g.ID, g.ScenarioCaption, WARNING_AFTER_HOURS)
			SendMessageSafe(context.Background(), chatID, text, g.StartMsgID, -1, enums.ParseModeNone)
		}
		activeGamesMu.Unlock()
	})

	game.CloseTimer = time.AfterFunc(time.Duration(AUTO_CLOSE_AFTER_HOURS)*time.Hour, func() {
		activeGamesMu.Lock()
		if g, ok := activeGames[chatID][gameID]; ok && !g.Closing {
			g.Closing = true
			activeGamesMu.Unlock()
			RemoveGameByID(chatID, gameID, true)
		} else {
			activeGamesMu.Unlock()
		}
	})
}

func RemoveGameByID(chatID, gameID string, deleteBanners bool) *Game {
	activeGamesMu.Lock()
	games := activeGames[chatID]
	if games == nil {
		activeGamesMu.Unlock()
		return nil
	}
	game := games[gameID]
	if game == nil {
		activeGamesMu.Unlock()
		return nil
	}
	delete(games, gameID)
	if len(games) == 0 {
		delete(activeGames, chatID)
	}
	activeGamesMu.Unlock()

	if game.WarnTimer != nil {
		game.WarnTimer.Stop()
	}
	if game.CloseTimer != nil {
		game.CloseTimer.Stop()
	}

	startMessageIndexMu.Lock()
	delete(startMessageIndex, chatID+":"+game.StartMsgID)
	startMessageIndexMu.Unlock()

	if deleteBanners {
		DeleteMessagesSafe(context.Background(), chatID, []string{game.StartMsgID})
		if game.ForwardedMsgID != "" {
			DeleteMessagesSafe(context.Background(), TARGET_GROUP_GUID, []string{game.ForwardedMsgID})
		}
	}
	return game
}

func HandleUpdate(ctx context.Context, e *events.Event) error {
	chatGUID := e.ObjectGUID()
	if chatGUID == "" {
		return nil
	}
	if strings.HasPrefix(chatGUID, "c") {
		return nil
	}

	if !processedMessageIDs.Consume(chatGUID + ":" + e.MessageID()) {
		return nil
	}

	if strings.HasPrefix(chatGUID, "g") {
		if e.IsFileInline() {
			handleGroupPhoto(ctx, e)
		} else if e.Text() != "" {
			handleGroupText(ctx, e)
		}
	} else if strings.HasPrefix(chatGUID, "u") {
		handlePvText(ctx, e)
	}
	return nil
}

func handleGroupText(ctx context.Context, e *events.Event) {
	chatGUID := e.ObjectGUID()
	author := e.AuthorGUID()
	text := e.Text()
	normalized := NormalizeText(text)
	if normalized == "" {
		return
	}

	RegisterGroup(ctx, chatGUID, "", 0)

	if chatGUID == BROADCAST_GROUP_GUID {
		adminGuidsMu.RLock()
		isMain := adminGuids[author] || superAdminGuids[author]
		adminGuidsMu.RUnlock()
		if !isMain {
			return
		}
		if CANCEL_BROADCAST_REGEX.MatchString(normalized) {
			cancelBroadcast(ctx, e)
			return
		}
		if TAG_ALL_REGEX.MatchString(normalized) {
			tagAll(ctx, e)
			return
		}
		if match := BROADCAST_ALL_REGEX.FindStringSubmatch(normalized); match != nil {
			broadcastAll(ctx, e, match)
			return
		}
		return
	}

	adminGuidsMu.RLock()
	isMainAdmin := author == MAIN_ADMIN_GUID || superAdminGuids[author]
	isAdmin := adminGuids[author] || superAdminGuids[author]
	adminGuidsMu.RUnlock()

	if !isMainAdmin && !IsGroupSubscribed(chatGUID) {
		return
	}
	if !isMainAdmin && !IsUserAllowed(e) {
		return
	}

	replyID := e.ReplyMessageID()
	if isMainAdmin && CANCEL_BROADCAST_REGEX.MatchString(normalized) {
		cancelBroadcast(ctx, e)
		return
	}

	if replyID != "" && REPLY_DELETE_WORDS[normalized] {
		if isAdmin {
			deleteByReply(ctx, e)
			return
		}
	}

	if replyID != "" {
		first := FirstWord(normalized)
		if CONFIRM_WORDS[first] || REJECT_WORDS[first] {
			processConfirmation(ctx, e, replyID, first)
			return
		}
	}

	if replyID != "" && (isAdmin || IsSpecialUser(author)) {
		if START_REGEX.MatchString(normalized) || END_REGEX.MatchString(normalized) {
			if tryReplyPhotoCommand(ctx, e, normalized) {
				return
			}
		}
	}

	dispatchText(ctx, e, normalized, isAdmin, isMainAdmin)
}

func IsSpecialUser(guid string) bool {
	specialUsersMu.RLock()
	defer specialUsersMu.RUnlock()
	return specialUsers[guid]
}

func handleGroupPhoto(ctx context.Context, e *events.Event) {
	chatGUID := e.ObjectGUID()
	if chatGUID == BROADCAST_GROUP_GUID {
		return
	}
	author := e.AuthorGUID()
	text := e.Text()
	normalized := NormalizeText(text)
	if normalized == "" {
		return
	}

	RegisterGroup(ctx, chatGUID, "", 0)

	adminGuidsMu.RLock()
	isMainAdmin := author == MAIN_ADMIN_GUID || superAdminGuids[author]
	adminGuidsMu.RUnlock()

	if !isMainAdmin && !IsGroupSubscribed(chatGUID) {
		return
	}
	if !IsUserAllowed(e) {
		return
	}

	if START_REGEX.MatchString(normalized) {
		processStart(ctx, e, START_REGEX.FindStringSubmatch(normalized))
	} else if END_REGEX.MatchString(normalized) {
		processEnd(ctx, e, END_REGEX.FindStringSubmatch(normalized))
	}
}

func handlePvText(ctx context.Context, e *events.Event) {
	author := e.AuthorGUID()
	if author != MAIN_ADMIN_GUID {
		return
	}
	text := e.Text()
	if text == "" {
		return
	}
	normalized := NormalizeText(text)

	if PV_LIST_GROUPS_REGEX.MatchString(normalized) {
		pvListGroups(ctx, e)
		return
	}
	if m := PV_ADD_GROUP_REGEX.FindStringSubmatch(text); m != nil {
		pvAddGroup(ctx, e, m[1])
		return
	}
	if m := PV_LEAVE_GROUP_REGEX.FindStringSubmatch(normalized); m != nil {
		pvLeaveGroup(ctx, e, m)
		return
	}
	if HELP_REGEX.MatchString(normalized) {
		SendMessageSafe(ctx, e.ObjectGUID(), PV_HELP_TEXT, e.MessageID(), 0, enums.ParseModeNone)
		return
	}
}

func tryReplyPhotoCommand(ctx context.Context, e *events.Event, normalized string) bool {
	startMatch := START_REGEX.FindStringSubmatch(normalized)
	var endMatch []string
	if startMatch == nil {
		endMatch = END_REGEX.FindStringSubmatch(normalized)
	}
	if startMatch == nil && endMatch == nil {
		return false
	}

	replyID := e.ReplyMessageID()
	res, err := acc.GetMessagesByID(ctx, e.ObjectGUID(), []string{replyID})
	if err != nil {
		return false
	}
	msgs := res.Slice("messages")
	if len(msgs) == 0 {
		return false
	}
	msg := events.New(msgs[0].(map[string]any))
	fileInline := msg.EventOf("file_inline")
	if fileInline == nil {
		return false
	}

	photoAuthor := msg.Str("author_object_guid")
	if startMatch != nil {
		processStartFromReply(ctx, e, startMatch, photoAuthor, fileInline, msg)
	} else {
		processEndFromReply(ctx, e, endMatch, photoAuthor, fileInline, msg)
	}
	return true
}

func processStartFromReply(ctx context.Context, e *events.Event, match []string, photoAuthor string, fileInline *events.Event, msg *events.Event) {
	chatID := e.ObjectGUID()
	replyAuthor := e.AuthorGUID()
	scenarioPart := strings.TrimSpace(match[1])
	builderFa := strings.TrimSpace(match[2])
	shouldForward := match[3] != ""

	scenarioWords := strings.Fields(scenarioPart)
	var filtered []string
	hasDonyto := false
	for _, w := range scenarioWords {
		if DUNYTO_VARIATIONS[w] {
			hasDonyto = true
		} else {
			filtered = append(filtered, w)
		}
	}
	var scenarioCaption string
	if hasDonyto {
		sf := strings.Join(filtered, " ")
		if sf != "" {
			scenarioCaption = sf + " ➪ دُنیتو"
		} else {
			scenarioCaption = "دُنیتو"
		}
	} else {
		scenarioCaption = scenarioPart
	}

	builderEn, corrected := NormalizeBuilderName(builderFa)
	signature := fmt.Sprintf("S|%s|%s|%v", scenarioCaption, strings.ToLower(builderEn), shouldForward)

	pendingBannersMu.RLock()
	for _, b := range pendingBanners {
		if b.Signature == signature {
			pendingBannersMu.RUnlock()
			RespondAndCleanup(ctx, e, "⚠️ پیش‌نمایشی با همین اطلاعات در انتظار تأیید است.", -1)
			return
		}
	}
	pendingBannersMu.RUnlock()

	now := GetShamsiNow()
	caption := fmt.Sprintf("🌱بِه نامِ خوبی راستی و پاکی 🌱\n\n🧑‍💻سازنده دک: ✺ DoN 𖤍 %s✺\n\n📜سناریو: %s\n\n⏱︎ساعت شروع : %s\n\n🔖برای عضویت و ارتباط با لیدر گروه:\n\n@Don55555\n\n𒆜ادب احترام میاره❥︎احترام اعتبار𒆜", builderEn, scenarioCaption, now)
	note := ""
	if corrected != "" {
		note = fmt.Sprintf("\n\n⚠️ نام '%s' به '%s' اصلاح شد.", builderFa, corrected)
	}

	allowed := map[string]bool{replyAuthor: true}
	if photoAuthor != "" {
		allowed[photoAuthor] = true
	}

	CancelOldBanner(ctx, chatID, replyAuthor)

	banner := &BannerData{
		ChatID: chatID, FileInline: fileInline, Caption: caption,
		ShouldForward: shouldForward, ScenarioCaption: scenarioCaption,
		BuilderEn: builderEn, Now: now, CreatorGUID: replyAuthor,
		BuilderFa: builderFa, UserMsgID: msg.MessageID(),
		Signature: signature, AllowedUsers: allowed, CommandMsgID: e.MessageID(),
	}

	pendingBannersMu.Lock()
	pendingBanners[chatID+":"+replyAuthor] = banner
	pendingBannersMu.Unlock()

	previewText := fmt.Sprintf("⚠️ پیش‌نمایش بنر (از طریق ریپلای)%s\n\n%s\n\n⏳ مدت اعتبار: %d ثانیه\nآیا اطلاعات صحیح است؟\n✅ برای تایید کلمه بله را ریپلای کنید.\n❌ برای رد کردن کلمه نه را ریپلای کنید.", note, caption, PENDING_EXPIRE_SECONDS)
	res, err := acc.SendMessage(ctx, chatID, previewText, account.SendOptions{ReplyToMessageID: e.MessageID()})
	if err == nil {
		previewID := res.MessageID()
		banner.PreviewMsgID = previewID
		pendingByPreviewMu.Lock()
		pendingByPreview[chatID+":"+previewID] = replyAuthor
		pendingByPreviewMu.Unlock()
		banner.ExpireTimer = time.AfterFunc(time.Duration(PENDING_EXPIRE_SECONDS)*time.Second, func() {
			expirePending(chatID, replyAuthor, previewID)
		})
	}
}

func processEndFromReply(ctx context.Context, e *events.Event, match []string, photoAuthor string, fileInline *events.Event, msg *events.Event) {
	chatID := e.ObjectGUID()
	replyAuthor := e.AuthorGUID()
	winner := strings.TrimSpace(match[1])
	shouldForward := match[2] != ""

	activeGamesMu.RLock()
	game := activeGames[chatID][startMessageIndex[chatID+":"+e.ReplyMessageID()]]
	activeGamesMu.RUnlock()

	if game == nil {
		RespondAndCleanup(ctx, e, "📭 بازی فعالی ثبت نشده است.", -1)
		return
	}

	signature := fmt.Sprintf("E|%s|%s|%v", game.ID, strings.ToLower(winner), shouldForward)
	pendingEndsMu.RLock()
	for _, ed := range pendingEnds {
		if ed.Signature == signature {
			pendingEndsMu.RUnlock()
			RespondAndCleanup(ctx, e, "⚠️ پیش‌نمایش پایانی با همین اطلاعات در انتظار تأیید است.", -1)
			return
		}
	}
	pendingEndsMu.RUnlock()

	now := GetShamsiNow()
	caption := fmt.Sprintf("🏆🏆🏆🏆🏆🏆🏆🏆\n\n🎞سناریو: %s\n\n🧑‍💻سازنده لابی: ✺DσN 𖤍 %s✺\n\n🥇برنده : %s ➪ 🏆\n\n⏱︎ساعت پایان : %s\n\nمافیا های زندگیت رو بشناس 🔥🔥🔥\n\n🔖عضویت و ارتباط با لیدر گروه:\n\n@DoN55555\n\n𒆜ادب احترام میاره❥︎احترام اعتبار𒆜", game.ScenarioCaption, game.Builder, winner, now)

	allowed := map[string]bool{replyAuthor: true}
	if photoAuthor != "" {
		allowed[photoAuthor] = true
	}

	CancelOldEnd(ctx, chatID, replyAuthor)

	endData := &EndData{
		ChatID: chatID, GameID: game.ID, FileInline: fileInline, Caption: caption,
		ShouldForward: shouldForward, Winner: winner, Now: now,
		CreatorGUID: replyAuthor, UserMsgID: msg.MessageID(),
		Signature: signature, AllowedUsers: allowed, CommandMsgID: e.MessageID(),
	}

	pendingEndsMu.Lock()
	pendingEnds[chatID+":"+replyAuthor] = endData
	pendingEndsMu.Unlock()

	previewText := fmt.Sprintf("⚠️ پیش‌نمایش بنر پایان (از طریق ریپلای)\n\n%s\n\n⏳ مدت اعتبار: %d ثانیه\nآیا اطلاعات صحیح است؟\n✅ برای تایید کلمه بله را ریپلای کنید.\n❌ برای رد کردن کلمه نه را ریپلای کنید.", caption, PENDING_EXPIRE_SECONDS)
	res, err := acc.SendMessage(ctx, chatID, previewText, account.SendOptions{ReplyToMessageID: e.MessageID()})
	if err == nil {
		previewID := res.MessageID()
		endData.PreviewMsgID = previewID
		pendingEndByPreviewMu.Lock()
		pendingEndByPreview[chatID+":"+previewID] = replyAuthor
		pendingEndByPreviewMu.Unlock()
		endData.ExpireTimer = time.AfterFunc(time.Duration(PENDING_EXPIRE_SECONDS)*time.Second, func() {
			expirePendingEnd(chatID, replyAuthor, previewID)
		})
	}
}

func processStart(ctx context.Context, e *events.Event, match []string) {
	chatID := e.ObjectGUID()
	userGUID := e.AuthorGUID()
	scenarioPart := strings.TrimSpace(match[1])
	builderFa := strings.TrimSpace(match[2])
	shouldForward := match[3] != ""

	scenarioWords := strings.Fields(scenarioPart)
	var filtered []string
	hasDonyto := false
	for _, w := range scenarioWords {
		if DUNYTO_VARIATIONS[w] {
			hasDonyto = true
		} else {
			filtered = append(filtered, w)
		}
	}
	var scenarioCaption string
	if hasDonyto {
		sf := strings.Join(filtered, " ")
		if sf != "" {
			scenarioCaption = sf + " ➪ دُنیتو"
		} else {
			scenarioCaption = "دُنیتو"
		}
	} else {
		scenarioCaption = scenarioPart
	}

	builderEn, corrected := NormalizeBuilderName(builderFa)
	signature := fmt.Sprintf("S|%s|%s|%v", scenarioCaption, strings.ToLower(builderEn), shouldForward)

	pendingBannersMu.RLock()
	for _, b := range pendingBanners {
		if b.Signature == signature {
			pendingBannersMu.RUnlock()
			RespondAndCleanup(ctx, e, "⚠️ پیش‌نمایشی با همین اطلاعات در انتظار تأیید است.", -1)
			return
		}
	}
	pendingBannersMu.RUnlock()

	fileInline := e.FileInline()
	if fileInline == nil {
		return
	}

	now := GetShamsiNow()
	caption := fmt.Sprintf("🌱بِه نامِ خوبی راستی و پاکی 🌱\n\n🧑‍💻سازنده دک: ✺ DoN 𖤍 %s✺\n\n📜سناریو: %s\n\n⏱︎ساعت شروع : %s\n\n🔖برای عضویت و ارتباط با لیدر گروه:\n\n@Don55555\n\n𒆜ادب احترام میاره❥︎احترام اعتبار𒆜", builderEn, scenarioCaption, now)
	note := ""
	if corrected != "" {
		note = fmt.Sprintf("\n\n⚠️ نام '%s' به '%s' اصلاح شد.", builderFa, corrected)
	}

	CancelOldBanner(ctx, chatID, userGUID)

	banner := &BannerData{
		ChatID: chatID, FileInline: fileInline, Caption: caption,
		ShouldForward: shouldForward, ScenarioCaption: scenarioCaption,
		BuilderEn: builderEn, Now: now, CreatorGUID: userGUID,
		BuilderFa: builderFa, UserMsgID: e.MessageID(),
		Signature: signature, AllowedUsers: map[string]bool{userGUID: true},
	}

	pendingBannersMu.Lock()
	pendingBanners[chatID+":"+userGUID] = banner
	pendingBannersMu.Unlock()

	previewText := fmt.Sprintf("⚠️ پیش‌نمایش بنر%s\n\n%s\n\nآیا اطلاعات صحیح است؟\n✅ برای تایید کلمه بله را ریپلای کنید.\n❌ برای رد کردن کلمه نه را ریپلای کنید.", note, caption)
	res, err := acc.SendMessage(ctx, chatID, previewText, account.SendOptions{ReplyToMessageID: e.MessageID()})
	if err == nil {
		previewID := res.MessageID()
		banner.PreviewMsgID = previewID
		pendingByPreviewMu.Lock()
		pendingByPreview[chatID+":"+previewID] = userGUID
		pendingByPreviewMu.Unlock()
		banner.ExpireTimer = time.AfterFunc(time.Duration(PENDING_EXPIRE_SECONDS)*time.Second, func() {
			expirePending(chatID, userGUID, previewID)
		})
	}
}

func processEnd(ctx context.Context, e *events.Event, match []string) {
	chatID := e.ObjectGUID()
	userGUID := e.AuthorGUID()
	winner := strings.TrimSpace(match[1])
	shouldForward := match[2] != ""

	activeGamesMu.RLock()
	game := activeGames[chatID][startMessageIndex[chatID+":"+e.ReplyMessageID()]]
	if game == nil && e.ReplyMessageID() == "" {
		for _, g := range activeGames[chatID] {
			game = g
			break
		}
	}
	activeGamesMu.RUnlock()

	if game == nil {
		RespondAndCleanup(ctx, e, "🔍 بازی مرتبط با این ریپلای یافت نشد.", -1)
		return
	}

	signature := fmt.Sprintf("E|%s|%s|%v", game.ID, strings.ToLower(winner), shouldForward)
	pendingEndsMu.RLock()
	for _, ed := range pendingEnds {
		if ed.Signature == signature {
			pendingEndsMu.RUnlock()
			RespondAndCleanup(ctx, e, "⚠️ پیش‌نمایش پایانی با همین اطلاعات در انتظار تأیید است.", -1)
			return
		}
	}
	pendingEndsMu.RUnlock()

	fileInline := e.FileInline()
	if fileInline == nil {
		return
	}

	now := GetShamsiNow()
	caption := fmt.Sprintf("🏆🏆🏆🏆🏆🏆🏆🏆\n\n🎞سناریو: %s\n\n🧑‍💻سازنده لابی: ✺DσN 𖤍 %s✺\n\n🥇برنده : %s ➪ 🏆\n\n⏱︎ساعت پایان : %s\n\nمافیا های زندگیت رو بشناس 🔥🔥🔥\n\n🔖عضویت و ارتباط با لیدر گروه:\n\n@DoN55555\n\n𒆜ادب احترام میاره❥︎احترام اعتبار𒆜", game.ScenarioCaption, game.Builder, winner, now)

	CancelOldEnd(ctx, chatID, userGUID)

	endData := &EndData{
		ChatID: chatID, GameID: game.ID, FileInline: fileInline, Caption: caption,
		ShouldForward: shouldForward, Winner: winner, Now: now,
		CreatorGUID: userGUID, UserMsgID: e.MessageID(),
		Signature: signature, AllowedUsers: map[string]bool{userGUID: true},
	}

	pendingEndsMu.Lock()
	pendingEnds[chatID+":"+userGUID] = endData
	pendingEndsMu.Unlock()

	previewText := fmt.Sprintf("⚠️ پیش‌نمایش بنر پایان\n\n%s\n\nآیا اطلاعات صحیح است؟\n✅ برای تایید کلمه بله را ریپلای کنید.\n❌ برای رد کردن کلمه نه را ریپلای کنید.", caption)
	res, err := acc.SendMessage(ctx, chatID, previewText, account.SendOptions{ReplyToMessageID: e.MessageID()})
	if err == nil {
		previewID := res.MessageID()
		endData.PreviewMsgID = previewID
		pendingEndByPreviewMu.Lock()
		pendingEndByPreview[chatID+":"+previewID] = userGUID
		pendingEndByPreviewMu.Unlock()
		endData.ExpireTimer = time.AfterFunc(time.Duration(PENDING_EXPIRE_SECONDS)*time.Second, func() {
			expirePendingEnd(chatID, userGUID, previewID)
		})
	}
}

func processConfirmation(ctx context.Context, e *events.Event, replyID, word string) {
	chatID := e.ObjectGUID()
	key := chatID + ":" + replyID

	processingPreviewsMu.Lock()
	if processingPreviews[key] {
		processingPreviewsMu.Unlock()
		return
	}
	processingPreviews[key] = true
	processingPreviewsMu.Unlock()
	defer func() {
		processingPreviewsMu.Lock()
		delete(processingPreviews, key)
		processingPreviewsMu.Unlock()
	}()

	pendingByPreviewMu.RLock()
	targetUser, isBanner := pendingByPreview[key]
	pendingByPreviewMu.RUnlock()

	if isBanner {
		pendingBannersMu.RLock()
		banner := pendingBanners[chatID+":"+targetUser]
		pendingBannersMu.RUnlock()

		if banner == nil || banner.PreviewMsgID != replyID {
			return
		}
		if !banner.AllowedUsers[e.AuthorGUID()] {
			RespondAndCleanup(ctx, e, "❌ فقط فرستنده عکس یا ریپلای‌زننده اصلی می‌تواند تأیید یا رد کند.", -1)
			return
		}
		if banner.ExpireTimer != nil {
			banner.ExpireTimer.Stop()
		}

		if CONFIRM_WORDS[word] {
			confirmBanner(ctx, e, banner)
		} else {
			rejectBanner(ctx, e, banner)
		}

		pendingBannersMu.Lock()
		delete(pendingBanners, chatID+":"+targetUser)
		pendingBannersMu.Unlock()
		pendingByPreviewMu.Lock()
		delete(pendingByPreview, key)
		pendingByPreviewMu.Unlock()
		return
	}

	pendingEndByPreviewMu.RLock()
	targetUser, isEnd := pendingEndByPreview[key]
	pendingEndByPreviewMu.RUnlock()

	if isEnd {
		pendingEndsMu.RLock()
		endData := pendingEnds[chatID+":"+targetUser]
		pendingEndsMu.RUnlock()

		if endData == nil || endData.PreviewMsgID != replyID {
			return
		}
		if !endData.AllowedUsers[e.AuthorGUID()] {
			RespondAndCleanup(ctx, e, "❌ فقط فرستنده عکس یا ریپلای‌زننده اصلی می‌تواند تأیید یا رد کند.", -1)
			return
		}
		if endData.ExpireTimer != nil {
			endData.ExpireTimer.Stop()
		}

		if CONFIRM_WORDS[word] {
			confirmEnd(ctx, e, endData)
		} else {
			rejectEnd(ctx, e, endData)
		}

		pendingEndsMu.Lock()
		delete(pendingEnds, chatID+":"+targetUser)
		pendingEndsMu.Unlock()
		pendingEndByPreviewMu.Lock()
		delete(pendingEndByPreview, key)
		pendingEndByPreviewMu.Unlock()
	}
}

func confirmBanner(ctx context.Context, e *events.Event, b *BannerData) {
	sent, err := acc.SendPhoto(ctx, b.ChatID, b.FileInline, b.Caption)
	if err != nil {
		RespondAndCleanup(ctx, e, "❌ خطا در ارسال بنر.", -1)
		return
	}
	sentID := sent.MessageID()

	activeGamesMu.Lock()
	if activeGames[b.ChatID] == nil {
		activeGames[b.ChatID] = make(map[string]*Game)
	}
	gameID := FindAvailableGameID(activeGames[b.ChatID])
	game := &Game{
		ID: gameID, StartMsgID: sentID, ScenarioCaption: b.ScenarioCaption,
		Builder: b.BuilderEn, CreatorGUID: b.CreatorGUID, ShouldForward: b.ShouldForward,
		StartTime: b.Now, StartMono: time.Now(),
	}
	activeGames[b.ChatID][gameID] = game
	activeGamesMu.Unlock()

	startMessageIndexMu.Lock()
	startMessageIndex[b.ChatID+":"+sentID] = gameID
	startMessageIndexMu.Unlock()

	StartGameTimer(b.ChatID, gameID)

	var allowed []string
	for u := range b.AllowedUsers {
		allowed = append(allowed, u)
	}
	UpdateStatsMulti(b.BuilderEn, b.ScenarioCaption, allowed, b.ChatID)

	if b.ShouldForward && sentID != "" {
		getForwardQueue() <- map[string]any{
			"from_guid": b.ChatID, "message_id": sentID,
			"chat_id": b.ChatID, "game_id": gameID,
		}
	}

	DeleteMessagesSafe(ctx, b.ChatID, []string{b.PreviewMsgID, e.MessageID()})
}

func rejectBanner(ctx context.Context, e *events.Event, b *BannerData) {
	SendMessageSafe(ctx, b.ChatID, "♻️ بنر رد شد.", e.MessageID(), -1, enums.ParseModeNone)
	DeleteMessagesSafe(ctx, b.ChatID, []string{b.PreviewMsgID, e.MessageID()})
}

func confirmEnd(ctx context.Context, e *events.Event, ed *EndData) {
	activeGamesMu.RLock()
	game := activeGames[ed.ChatID][ed.GameID]
	activeGamesMu.RUnlock()

	if game == nil {
		RespondAndCleanup(ctx, e, "📭 بازی یافت نشد.", -1)
		return
	}

	opts := account.SendOptions{ReplyToMessageID: game.StartMsgID}
	sent, err := acc.SendPhoto(ctx, ed.ChatID, ed.FileInline, ed.Caption, opts)
	if err != nil {
		RespondAndCleanup(ctx, e, "❌ خطا در ارسال بنر پایان.", -1)
		return
	}
	sentID := sent.MessageID()

	RemoveGameByID(ed.ChatID, ed.GameID, false)

	if ed.ShouldForward && sentID != "" {
		getForwardQueue() <- map[string]any{
			"from_guid": ed.ChatID, "message_id": sentID,
		}
	}

	DeleteMessagesSafe(ctx, ed.ChatID, []string{ed.PreviewMsgID, e.MessageID()})
}

func rejectEnd(ctx context.Context, e *events.Event, ed *EndData) {
	SendMessageSafe(ctx, ed.ChatID, "♻️ بنر پایان رد شد.", e.MessageID(), -1, enums.ParseModeNone)
	DeleteMessagesSafe(ctx, ed.ChatID, []string{ed.PreviewMsgID, e.MessageID()})
}

func CancelOldBanner(ctx context.Context, chatID, userGUID string) {
	pendingBannersMu.Lock()
	old := pendingBanners[chatID+":"+userGUID]
	if old != nil {
		delete(pendingBanners, chatID+":"+userGUID)
	}
	pendingBannersMu.Unlock()

	if old != nil {
		if old.ExpireTimer != nil {
			old.ExpireTimer.Stop()
		}
		pendingByPreviewMu.Lock()
		delete(pendingByPreview, chatID+":"+old.PreviewMsgID)
		pendingByPreviewMu.Unlock()
		go DeleteMessagesSafe(context.Background(), chatID, []string{old.PreviewMsgID})
	}
}

func CancelOldEnd(ctx context.Context, chatID, userGUID string) {
	pendingEndsMu.Lock()
	old := pendingEnds[chatID+":"+userGUID]
	if old != nil {
		delete(pendingEnds, chatID+":"+userGUID)
	}
	pendingEndsMu.Unlock()

	if old != nil {
		if old.ExpireTimer != nil {
			old.ExpireTimer.Stop()
		}
		pendingEndByPreviewMu.Lock()
		delete(pendingEndByPreview, chatID+":"+old.PreviewMsgID)
		pendingEndByPreviewMu.Unlock()
		go DeleteMessagesSafe(context.Background(), chatID, []string{old.PreviewMsgID})
	}
}

func expirePending(chatID, userGUID, previewID string) {
	key := chatID + ":" + previewID
	processingPreviewsMu.Lock()
	if processingPreviews[key] {
		processingPreviewsMu.Unlock()
		return
	}
	processingPreviewsMu.Unlock()

	pendingBannersMu.Lock()
	banner := pendingBanners[chatID+":"+userGUID]
	if banner != nil && banner.PreviewMsgID == previewID {
		delete(pendingBanners, chatID+":"+userGUID)
		pendingByPreviewMu.Lock()
		delete(pendingByPreview, key)
		pendingByPreviewMu.Unlock()
		go DeleteMessagesSafe(context.Background(), chatID, []string{previewID})
	}
	pendingBannersMu.Unlock()
}

func expirePendingEnd(chatID, userGUID, previewID string) {
	key := chatID + ":" + previewID
	processingPreviewsMu.Lock()
	if processingPreviews[key] {
		processingPreviewsMu.Unlock()
		return
	}
	processingPreviewsMu.Unlock()

	pendingEndsMu.Lock()
	ed := pendingEnds[chatID+":"+userGUID]
	if ed != nil && ed.PreviewMsgID == previewID {
		delete(pendingEnds, chatID+":"+userGUID)
		pendingEndByPreviewMu.Lock()
		delete(pendingEndByPreview, key)
		pendingEndByPreviewMu.Unlock()
		go DeleteMessagesSafe(context.Background(), chatID, []string{previewID})
	}
	pendingEndsMu.Unlock()
}

func FindAvailableGameID(games map[string]*Game) string {
	for i := 1; i < 100; i++ {
		id := fmt.Sprintf("%02d", i)
		if _, ok := games[id]; !ok {
			return id
		}
	}
	return "XX"
}

func RegisterGroup(ctx context.Context, guid, title string, count int) {
	botGroupsMu.Lock()
	_, exists := botGroups[guid]
	if !exists {
		botGroups[guid] = map[string]any{"title": title, "member_count": count}
		botGroupsMu.Unlock()
		if USE_DATABASE {
			EnqueueDB(func(tx *sql.Tx) {
				tx.Exec("INSERT INTO bot_groups(group_guid, title, member_count) VALUES(?, ?, ?) ON CONFLICT(group_guid) DO UPDATE SET title = excluded.title, member_count = excluded.member_count", guid, title, count)
			})
		}
		return
	}
	botGroupsMu.Unlock()
}

func tagAll(ctx context.Context, e *events.Event) {
	chat := e.ObjectGUID()
	replyID := e.ReplyMessageID()
	acc.DeleteMessages(ctx, chat, []string{e.MessageID()})

	res, err := acc.GetGroupAllMembers(ctx, chat, "", "")
	if err != nil {
		SendMessageSafe(ctx, chat, "❌ خطا در دریافت اعضا.", "", -1, enums.ParseModeNone)
		return
	}
	var members []string
	for _, item := range res.Slice("in_chat") {
		m := events.New(item.(map[string]any))
		members = append(members, m.Str("member_guid"))
	}
	if len(members) == 0 {
		SendMessageSafe(ctx, chat, "📭 هیچ عضوی یافت نشد.", "", -1, enums.ParseModeNone)
		return
	}

	var textParts []string
	currentLen := 0

	for _, guid := range members {
		display := "کاربر"
		info, _ := acc.GetUserInfo(ctx, guid)
		if info != nil {
			display = FormatUserName(info)
		}
		display = SanitizeMdName(display)
		mention := fmt.Sprintf("[%s](%s) , ", display, guid)

		if currentLen+len([]rune(mention)) > TAG_CHUNK_LIMIT {
			SendTagChunk(ctx, chat, replyID, textParts)
			textParts = nil
			currentLen = 0
		}

		textParts = append(textParts, mention)
		currentLen += len([]rune(mention))
	}
	if len(textParts) > 0 {
		SendTagChunk(ctx, chat, replyID, textParts)
	}
}

func SendTagChunk(ctx context.Context, chat, replyID string, textParts []string) {
	text := strings.Join(textParts, "")
	opts := account.SendOptions{
		ReplyToMessageID: replyID,
		ParseMode:        enums.ParseModeMarkdown,
	}
	acc.SendMessage(ctx, chat, text, opts)
}

func broadcastAll(ctx context.Context, e *events.Event, match []string) {
	activeBroadcastMu.Lock()
	if activeBroadcastCancel != nil {
		activeBroadcastMu.Unlock()
		RespondAndCleanup(ctx, e, "⚠️ یک فور همگانی در حال اجراست. ابتدا «فور لغو» بزنید.", -1)
		return
	}

	body := strings.TrimSpace(match[1])
	replyID := e.ReplyMessageID()
	chat := e.ObjectGUID()

	if body == "" && replyID == "" {
		activeBroadcastMu.Unlock()
		RespondAndCleanup(ctx, e, "⚠️ متن پیام خالی است.\nدستور: فور همگانی متن پیام\nیا: ریپلای روی یک پیام + فور همگانی", -1)
		return
	}

	acc.DeleteMessages(ctx, chat, []string{e.MessageID()})

	res, err := acc.GetGroupAllMembers(ctx, chat, "", "")
	if err != nil || res == nil {
		activeBroadcastMu.Unlock()
		SendMessageSafe(ctx, chat, "❌ خطا در دریافت لیست اعضا.", "", -1, enums.ParseModeNone)
		return
	}

	var targets []string
	for _, item := range res.Slice("in_chat") {
		m := events.New(item.(map[string]any))
		guid := m.Str("member_guid")
		if guid != acc.GUID() {
			targets = append(targets, guid)
		}
	}

	if len(targets) == 0 {
		activeBroadcastMu.Unlock()
		SendMessageSafe(ctx, chat, "📭 هیچ گیرنده‌ای یافت نشد.", "", -1, enums.ParseModeNone)
		return
	}

	bCtx, cancel := context.WithCancel(context.Background())
	activeBroadcastCancel = cancel
	activeBroadcastMu.Unlock()

	kindLabel := "متن"
	if body == "" {
		kindLabel = "پیام فورواردی"
	}

	SendMessageSafe(ctx, chat, fmt.Sprintf("📢 شروع فور همگانی (%s) به %d کاربر.\n⏱ هر %v ثانیه یک ارسال.\nبرای لغو: فور لغو", kindLabel, len(targets), BROADCAST_INTERVAL), "", -1, enums.ParseModeNone)

	go func() {
		sent, failed := 0, 0
		defer func() {
			activeBroadcastMu.Lock()
			activeBroadcastCancel = nil
			activeBroadcastMu.Unlock()
			report := fmt.Sprintf("📢 فور همگانی تکمیل شد.\n👥 گیرندگان: %d\n✅ موفق: %d\n❌ ناموفق: %d", len(targets), sent, failed)
			SendMessageSafe(context.Background(), chat, report, "", -1, enums.ParseModeNone)
		}()

		for _, target := range targets {
			select {
			case <-bCtx.Done():
				return
			default:
			}

			if body != "" {
				_, err := acc.SendMessage(context.Background(), target, body)
				if err == nil {
					sent++
				} else {
					failed++
				}
			} else {
				_, err := acc.ForwardMessages(context.Background(), chat, []string{replyID}, target, false, false)
				if err == nil {
					sent++
				} else {
					failed++
				}
			}
			time.Sleep(BROADCAST_INTERVAL)
		}
	}()
}

func cancelBroadcast(ctx context.Context, e *events.Event) {
	activeBroadcastMu.Lock()
	defer activeBroadcastMu.Unlock()
	if activeBroadcastCancel == nil {
		RespondAndCleanup(ctx, e, "⚠️ هیچ فور همگانی در حال اجرا نیست.", -1)
		return
	}
	activeBroadcastCancel()
	RespondAndCleanup(ctx, e, "🛑 درخواست لغو فور همگانی ثبت شد.", -1)
}

func deleteByReply(ctx context.Context, e *events.Event) {
	acc.DeleteMessages(ctx, e.ObjectGUID(), []string{e.ReplyMessageID(), e.MessageID()})
}

func pinByReply(ctx context.Context, e *events.Event) {
	replyID := e.ReplyMessageID()
	if replyID == "" {
		RespondAndCleanup(ctx, e, "⚠️ روی پیام مورد نظر ریپلای کنید و سپس «پین» را بزنید.", -1)
		return
	}
	acc.SetPinMessage(ctx, e.ObjectGUID(), replyID)
	RespondAndCleanup(ctx, e, "📌 پیام پین شد.", -1)
}

func unpinByReply(ctx context.Context, e *events.Event) {
	replyID := e.ReplyMessageID()
	if replyID == "" {
		RespondAndCleanup(ctx, e, "⚠️ روی پیام پین‌شده ریپلای کنید و سپس «انپین» را بزنید.", -1)
		return
	}
	acc.SetUnpinMessage(ctx, e.ObjectGUID(), replyID)
	RespondAndCleanup(ctx, e, "📌 پین پیام برداشته شد.", -1)
}

func adminStop(ctx context.Context, e *events.Event) {
	IS_BOT_ACTIVE = false
	RespondAndCleanup(ctx, e, "🛑 ربات متوقف شد.", -1)
}

func adminStart(ctx context.Context, e *events.Event) {
	IS_BOT_ACTIVE = true
	RespondAndCleanup(ctx, e, "✅ ربات راه‌اندازی شد.", -1)
}

func setAutoStats(ctx context.Context, e *events.Event, match []string) {
	autoStatsMu.Lock()
	defer autoStatsMu.Unlock()
	autoStatsEnabled = true
	autoStatsTime = DIGIT_REPLACER.Replace(match[1])
	lastAutoStatsDate = ""
	if autoStatsHandle != nil {
		autoStatsHandle.Stop()
	}
	ScheduleAutoStats()
	RespondAndCleanup(ctx, e, fmt.Sprintf("⏰ آمار خودکار برای ساعت %s فعال شد.", autoStatsTime), -1)
}

func disableAutoStats(ctx context.Context, e *events.Event) {
	autoStatsMu.Lock()
	defer autoStatsMu.Unlock()
	autoStatsEnabled = false
	if autoStatsHandle != nil {
		autoStatsHandle.Stop()
	}
	RespondAndCleanup(ctx, e, "🛑 آمار خودکار غیرفعال شد.", -1)
}

func setAutoDelete(ctx context.Context, e *events.Event, match []string) {
	sec, _ := strconv.Atoi(DIGIT_REPLACER.Replace(match[1]))
	if sec < 1 {
		RespondAndCleanup(ctx, e, "⚠️ زمان باید حداقل ۱ ثانیه باشد.", -1)
		return
	}
	if sec > 86400 {
		RespondAndCleanup(ctx, e, "⚠️ زمان نباید بیشتر از ۲۴ ساعت باشد.", -1)
		return
	}
	AUTO_DELETE_SECONDS = sec
	RespondAndCleanup(ctx, e, fmt.Sprintf("⏱️ حذف خودکار روی %d ثانیه تنظیم شد.", sec), -1)
}

func disableAutoDelete(ctx context.Context, e *events.Event) {
	AUTO_DELETE_SECONDS = 0
	RespondAndCleanup(ctx, e, "🛑 حذف خودکار پیام‌ها غیرفعال شد.", -1)
}

func blockUser(ctx context.Context, e *events.Event, match []string) {
	target := match[1]
	if target == "" && e.ReplyMessageID() != "" {
		res, _ := acc.GetMessagesByID(ctx, e.ObjectGUID(), []string{e.ReplyMessageID()})
		msgs := res.Slice("messages")
		if len(msgs) > 0 {
			target = events.New(msgs[0].(map[string]any)).AuthorGUID()
		}
	}
	if target == "" {
		RespondAndCleanup(ctx, e, "❌ کاربری مشخص نشد.", -1)
		return
	}
	adminGuidsMu.RLock()
	superAdminGuidsMu.RLock()
	isProtected := target == MAIN_ADMIN_GUID || superAdminGuids[target]
	superAdminGuidsMu.RUnlock()
	adminGuidsMu.RUnlock()

	if isProtected {
		RespondAndCleanup(ctx, e, "⛔ نمی‌توانید ادمین اصلی یا جانشین را بلاک کنید.", -1)
		return
	}

	blockedUsersMu.Lock()
	if blockedUsers[target] {
		blockedUsersMu.Unlock()
		RespondAndCleanup(ctx, e, fmt.Sprintf("⚠️ کاربر `%s` قبلاً بلاک شده است.", target), -1)
		return
	}
	blockedUsers[target] = true
	blockedUsersMu.Unlock()

	if USE_DATABASE {
		EnqueueDB(func(tx *sql.Tx) {
			tx.Exec("INSERT OR REPLACE INTO access_control(guid, type) VALUES(?, 'blocked')", target)
		})
	}
	RespondAndCleanup(ctx, e, fmt.Sprintf("🚫 کاربر `%s` بلاک شد.", target), -1)
}

func unblockUser(ctx context.Context, e *events.Event, match []string) {
	target := match[1]
	if target == "" && e.ReplyMessageID() != "" {
		res, _ := acc.GetMessagesByID(ctx, e.ObjectGUID(), []string{e.ReplyMessageID()})
		msgs := res.Slice("messages")
		if len(msgs) > 0 {
			target = events.New(msgs[0].(map[string]any)).AuthorGUID()
		}
	}
	if target == "" {
		RespondAndCleanup(ctx, e, "❌ کاربری مشخص نشد.", -1)
		return
	}

	blockedUsersMu.Lock()
	if !blockedUsers[target] {
		blockedUsersMu.Unlock()
		RespondAndCleanup(ctx, e, fmt.Sprintf("⚠️ کاربر `%s` در لیست بلاک نیست.", target), -1)
		return
	}
	delete(blockedUsers, target)
	blockedUsersMu.Unlock()

	if USE_DATABASE {
		EnqueueDB(func(tx *sql.Tx) {
			tx.Exec("DELETE FROM access_control WHERE guid=? AND type='blocked'", target)
		})
	}
	SendMessageSafe(ctx, target, "✅ شما از لیست بلاک ربات خارج شدید.", "", -1, enums.ParseModeNone)
	RespondAndCleanup(ctx, e, fmt.Sprintf("🔓 کاربر `%s` از بلاک خارج شد.", target), -1)
}

func specialUser(ctx context.Context, e *events.Event, match []string) {
	target := match[1]
	if target == "" && e.ReplyMessageID() != "" {
		res, _ := acc.GetMessagesByID(ctx, e.ObjectGUID(), []string{e.ReplyMessageID()})
		msgs := res.Slice("messages")
		if len(msgs) > 0 {
			target = events.New(msgs[0].(map[string]any)).AuthorGUID()
		}
	}
	if target == "" {
		RespondAndCleanup(ctx, e, "❌ کاربری مشخص نشد.", -1)
		return
	}

	specialUsersMu.Lock()
	if specialUsers[target] {
		specialUsersMu.Unlock()
		RespondAndCleanup(ctx, e, fmt.Sprintf("⭐ کاربر `%s` قبلاً ویژه شده است.", target), -1)
		return
	}
	specialUsers[target] = true
	specialUsersMu.Unlock()

	blockedUsersMu.Lock()
	delete(blockedUsers, target)
	blockedUsersMu.Unlock()

	if USE_DATABASE {
		EnqueueDB(func(tx *sql.Tx) {
			tx.Exec("INSERT OR REPLACE INTO access_control(guid, type) VALUES(?, 'special')", target)
			tx.Exec("DELETE FROM access_control WHERE guid=? AND type='blocked'", target)
		})
	}
	RespondAndCleanup(ctx, e, fmt.Sprintf("⭐ کاربر `%s` ویژه شد.", target), -1)
}

func closeGroup(ctx context.Context, e *events.Event) {
	_, err := acc.SetGroupDefaultAccess(ctx, e.ObjectGUID(), []string{"ViewMembers", "ViewAdmins", "AddMember"})
	if err != nil {
		RespondAndCleanup(ctx, e, "❌ خطا در بستن گروه.", -1)
		return
	}
	RespondAndCleanup(ctx, e, "🔒 گروه بسته شد.", -1)
}

func openGroup(ctx context.Context, e *events.Event) {
	_, err := acc.SetGroupDefaultAccess(ctx, e.ObjectGUID(), []string{"ViewMembers", "ViewAdmins", "SendMessages", "AddMember"})
	if err != nil {
		RespondAndCleanup(ctx, e, "❌ خطا در باز کردن گروه.", -1)
		return
	}
	RespondAndCleanup(ctx, e, "🔓 گروه باز شد.", -1)
}

func addAdmin(ctx context.Context, e *events.Event, match []string) {
	target := match[1]
	if target == "" && e.ReplyMessageID() != "" {
		res, _ := acc.GetMessagesByID(ctx, e.ObjectGUID(), []string{e.ReplyMessageID()})
		msgs := res.Slice("messages")
		if len(msgs) > 0 {
			target = events.New(msgs[0].(map[string]any)).AuthorGUID()
		}
	}
	if target == "" {
		RespondAndCleanup(ctx, e, "❌ کاربری مشخص نشد.", -1)
		return
	}

	adminGuidsMu.Lock()
	defer adminGuidsMu.Unlock()
	if adminGuids[target] {
		RespondAndCleanup(ctx, e, fmt.Sprintf("⚠️ کاربر `%s` قبلاً ادمین است.", target), -1)
		return
	}
	adminGuids[target] = true
	if USE_DATABASE {
		EnqueueDB(func(tx *sql.Tx) {
			tx.Exec("INSERT OR REPLACE INTO access_control(guid, type) VALUES(?, 'admin')", target)
		})
	}
	RespondAndCleanup(ctx, e, fmt.Sprintf("👑 کاربر `%s` به لیست ادمین‌ها اضافه شد.", target), -1)
}

func removeAdmin(ctx context.Context, e *events.Event, match []string) {
	target := match[1]
	if target == "" && e.ReplyMessageID() != "" {
		res, _ := acc.GetMessagesByID(ctx, e.ObjectGUID(), []string{e.ReplyMessageID()})
		msgs := res.Slice("messages")
		if len(msgs) > 0 {
			target = events.New(msgs[0].(map[string]any)).AuthorGUID()
		}
	}
	if target == "" {
		RespondAndCleanup(ctx, e, "❌ کاربری مشخص نشد.", -1)
		return
	}

	adminGuidsMu.Lock()
	defer adminGuidsMu.Unlock()
	if target == MAIN_ADMIN_GUID {
		RespondAndCleanup(ctx, e, "⛔ نمی‌توانید ادمین اصلی را عزل کنید.", -1)
		return
	}
	if !adminGuids[target] {
		RespondAndCleanup(ctx, e, fmt.Sprintf("⚠️ کاربر `%s` ادمین نیست.", target), -1)
		return
	}
	delete(adminGuids, target)
	if USE_DATABASE {
		EnqueueDB(func(tx *sql.Tx) {
			tx.Exec("DELETE FROM access_control WHERE guid=? AND type='admin'", target)
		})
	}
	RespondAndCleanup(ctx, e, fmt.Sprintf("❌ کاربر `%s` از لیست ادمین‌ها حذف شد.", target), -1)
}

func addSuperAdmin(ctx context.Context, e *events.Event, match []string) {
	target := match[1]
	if target == "" && e.ReplyMessageID() != "" {
		res, _ := acc.GetMessagesByID(ctx, e.ObjectGUID(), []string{e.ReplyMessageID()})
		msgs := res.Slice("messages")
		if len(msgs) > 0 {
			target = events.New(msgs[0].(map[string]any)).AuthorGUID()
		}
	}
	if target == "" {
		RespondAndCleanup(ctx, e, "❌ کاربری مشخص نشد.", -1)
		return
	}

	superAdminGuidsMu.Lock()
	defer superAdminGuidsMu.Unlock()
	if target == MAIN_ADMIN_GUID || superAdminGuids[target] {
		RespondAndCleanup(ctx, e, fmt.Sprintf("⚠️ کاربر `%s` قبلاً جانشین است.", target), -1)
		return
	}
	superAdminGuids[target] = true
	adminGuidsMu.Lock()
	adminGuids[target] = true
	adminGuidsMu.Unlock()

	if USE_DATABASE {
		EnqueueDB(func(tx *sql.Tx) {
			tx.Exec("INSERT OR IGNORE INTO super_admins(guid) VALUES(?)", target)
		})
	}
	RespondAndCleanup(ctx, e, fmt.Sprintf("👑 کاربر `%s` به عنوان جانشین منصوب شد.", target), -1)
}

func removeSuperAdmin(ctx context.Context, e *events.Event, match []string) {
	target := match[1]
	if target == "" && e.ReplyMessageID() != "" {
		res, _ := acc.GetMessagesByID(ctx, e.ObjectGUID(), []string{e.ReplyMessageID()})
		msgs := res.Slice("messages")
		if len(msgs) > 0 {
			target = events.New(msgs[0].(map[string]any)).AuthorGUID()
		}
	}
	if target == "" {
		RespondAndCleanup(ctx, e, "❌ کاربری مشخص نشد.", -1)
		return
	}

	superAdminGuidsMu.Lock()
	defer superAdminGuidsMu.Unlock()
	if target == MAIN_ADMIN_GUID {
		RespondAndCleanup(ctx, e, "⛔ نمی‌توانید ادمین اصلی را عزل کنید.", -1)
		return
	}
	if !superAdminGuids[target] {
		RespondAndCleanup(ctx, e, fmt.Sprintf("⚠️ کاربر `%s` جانشین نیست.", target), -1)
		return
	}
	delete(superAdminGuids, target)
	if USE_DATABASE {
		EnqueueDB(func(tx *sql.Tx) {
			tx.Exec("DELETE FROM super_admins WHERE guid=?", target)
		})
	}
	RespondAndCleanup(ctx, e, fmt.Sprintf("❌ کاربر `%s` از لیست جانشین‌ها حذف شد.", target), -1)
}

func listSuperAdmins(ctx context.Context, e *events.Event) {
	lines := []string{fmt.Sprintf("👑 ادمین اصلی: `%s`", MAIN_ADMIN_GUID), ""}
	superAdminGuidsMu.RLock()
	if len(superAdminGuids) > 0 {
		lines = append(lines, "👤 لیست جانشین‌ها:")
		idx := 1
		for guid := range superAdminGuids {
			lines = append(lines, fmt.Sprintf("%d. `%s`", idx, guid))
			idx++
		}
	} else {
		lines = append(lines, "📭 جانشینی تعیین نشده است.")
	}
	superAdminGuidsMu.RUnlock()
	RespondAndCleanup(ctx, e, strings.Join(lines, "\n"), -1)
}

func deleteMessagesCommand(ctx context.Context, e *events.Event, match []string) {
	count, _ := strconv.Atoi(DIGIT_REPLACER.Replace(match[1]))
	if count < 1 {
		return
	}
	if count > 1000 {
		RespondAndCleanup(ctx, e, "⚠️ حداکثر ۱۰۰۰ پیام را می‌توانید حذف کنید.", -1)
		return
	}
	res, err := acc.GetMessages(ctx, e.ObjectGUID(), "", count, "FromMax")
	if err != nil {
		RespondAndCleanup(ctx, e, "❌ خطا در دریافت پیام‌ها.", -1)
		return
	}
	var ids []string
	for _, item := range res.Slice("messages") {
		ids = append(ids, events.New(item.(map[string]any)).MessageID())
	}
	if len(ids) == 0 {
		RespondAndCleanup(ctx, e, "📭 پیامی برای حذف یافت نشد.", -1)
		return
	}
	acc.DeleteMessages(ctx, e.ObjectGUID(), ids)
	RespondAndCleanup(ctx, e, fmt.Sprintf("🧹 %d پیام با موفقیت حذف شد.", len(ids)), -1)
}

func banUser(ctx context.Context, e *events.Event, match []string) {
	target := match[1]
	if target == "" && e.ReplyMessageID() != "" {
		res, _ := acc.GetMessagesByID(ctx, e.ObjectGUID(), []string{e.ReplyMessageID()})
		msgs := res.Slice("messages")
		if len(msgs) > 0 {
			target = events.New(msgs[0].(map[string]any)).AuthorGUID()
		}
	}
	if target == "" {
		RespondAndCleanup(ctx, e, "❌ کاربری مشخص نشد.", -1)
		return
	}
	_, err := acc.BanGroupMember(ctx, e.ObjectGUID(), target)
	if err != nil {
		RespondAndCleanup(ctx, e, "❌ خطا در بن کردن کاربر.", -1)
		return
	}
	RespondAndCleanup(ctx, e, fmt.Sprintf("🚫 کاربر `%s` بن شد.", target), -1)
}

func unbanUser(ctx context.Context, e *events.Event, match []string) {
	target := match[1]
	if target == "" && e.ReplyMessageID() != "" {
		res, _ := acc.GetMessagesByID(ctx, e.ObjectGUID(), []string{e.ReplyMessageID()})
		msgs := res.Slice("messages")
		if len(msgs) > 0 {
			target = events.New(msgs[0].(map[string]any)).AuthorGUID()
		}
	}
	if target == "" {
		RespondAndCleanup(ctx, e, "❌ کاربری مشخص نشد.", -1)
		return
	}
	_, err := acc.UnbanGroupMember(ctx, e.ObjectGUID(), target)
	if err != nil {
		RespondAndCleanup(ctx, e, "❌ خطا در آنبن کردن کاربر.", -1)
		return
	}
	linkRes, _ := acc.GetGroupLink(ctx, e.ObjectGUID())
	if linkRes != nil {
		link := linkRes.Str("join_link")
		if link != "" {
			SendMessageSafe(ctx, target, fmt.Sprintf("✅ شما از گروه آنبن شدید.\n🔗 لینک گروه:\n%s", link), "", -1, enums.ParseModeNone)
		}
	}
	RespondAndCleanup(ctx, e, fmt.Sprintf("🔓 کاربر `%s` از بن خارج شد.", target), -1)
}

func addBuilder(ctx context.Context, e *events.Event, match []string) {
	fa := strings.TrimSpace(match[1])
	en := strings.TrimSpace(match[2])
	if fa == "" || en == "" {
		return
	}
	if strings.ContainsAny(fa+en, "\n\r`[]()=") {
		RespondAndCleanup(ctx, e, "⚠️ نام نباید شامل کاراکترهای خاص باشد.", -1)
		return
	}

	_name_lock.Lock()
	if _, exists := NAME_MAP[fa]; exists {
		_name_lock.Unlock()
		RespondAndCleanup(ctx, e, fmt.Sprintf("⚠️ سازنده «%s» از قبل وجود دارد.", fa), -1)
		return
	}
	bid := NEXT_BUILDER_ID
	NEXT_BUILDER_ID++
	NAME_MAP[fa] = en
	NAME_ID_MAP[bid] = fa
	rebuildNameIndex()
	_name_lock.Unlock()

	_builder_lock.Lock()
	_builder_lower_map[strings.ToLower(en)] = en
	_builder_lock.Unlock()

	if USE_DATABASE {
		EnqueueDB(func(tx *sql.Tx) {
			tx.Exec("INSERT OR REPLACE INTO builders(id, fa, en) VALUES(?, ?, ?)", bid, fa, en)
		})
	}
	RespondAndCleanup(ctx, e, fmt.Sprintf("✅ سازنده «%s» با شناسه #%d → «%s» اضافه شد.", fa, bid, en), -1)
}

func removeBuilder(ctx context.Context, e *events.Event, match []string) {
	bid, _ := strconv.Atoi(DIGIT_REPLACER.Replace(match[1]))
	_name_lock.Lock()
	fa, exists := NAME_ID_MAP[bid]
	if !exists {
		_name_lock.Unlock()
		RespondAndCleanup(ctx, e, fmt.Sprintf("⚠️ سازنده با شناسه #%d یافت نشد.", bid), -1)
		return
	}
	en := NAME_MAP[fa]
	delete(NAME_MAP, fa)
	delete(NAME_ID_MAP, bid)
	rebuildNameIndex()
	_name_lock.Unlock()

	_builder_lock.Lock()
	delete(_builder_lower_map, strings.ToLower(en))
	_builder_lock.Unlock()

	if USE_DATABASE {
		EnqueueDB(func(tx *sql.Tx) {
			tx.Exec("DELETE FROM builders WHERE id=?", bid)
		})
	}
	RespondAndCleanup(ctx, e, fmt.Sprintf("❌ سازنده «%s» (#%d) حذف شد.", fa, bid), -1)
}

func listBuilders(ctx context.Context, e *events.Event) {
	_name_lock.RLock()
	if len(NAME_ID_MAP) == 0 {
		_name_lock.RUnlock()
		RespondAndCleanup(ctx, e, "📭 لیست سازنده‌ها خالی است.", -1)
		return
	}
	var lines []string
	lines = append(lines, "📋 لیست سازنده‌ها:")
	var ids []int
	for id := range NAME_ID_MAP {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, bid := range ids {
		fa := NAME_ID_MAP[bid]
		en := NAME_MAP[fa]
		lines = append(lines, fmt.Sprintf("`#%d` %s → %s", bid, fa, en))
	}
	_name_lock.RUnlock()

	text := strings.Join(lines, "\n")
	if len(text) > 3500 {
		text = text[:3500] + "\n… (ادامه کوتاه شد)"
	}
	RespondAndCleanup(ctx, e, text, -1)
}

func deleteBuilderStats(ctx context.Context, e *events.Event, match []string) {
	idx, _ := strconv.Atoi(DIGIT_REPLACER.Replace(match[1]))
	statsMu.Lock()
	defer statsMu.Unlock()

	var ss []struct {
		Name  string
		Count int
	}
	for k, v := range gameStats.BuilderStats {
		ss = append(ss, struct {
			Name  string
			Count int
		}{k, v})
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i].Count > ss[j].Count })

	if len(ss) == 0 {
		RespondAndCleanup(ctx, e, "📭 آماری برای حذف وجود ندارد.", -1)
		return
	}
	if idx < 1 || idx > len(ss) {
		RespondAndCleanup(ctx, e, fmt.Sprintf("⚠️ شماره باید بین ۱ تا %d باشد.", len(ss)), -1)
		return
	}

	name := ss[idx-1].Name
	delete(gameStats.BuilderStats, name)
	delete(gameStats.DailyBuilderStats, name)
	for _, gs := range groupGameStats {
		delete(gs.BuilderStats, name)
		delete(gs.DailyBuilderStats, name)
	}

	if USE_DATABASE {
		EnqueueDB(func(tx *sql.Tx) {
			tx.Exec("DELETE FROM builder_stats WHERE name=?", name)
			tx.Exec("DELETE FROM daily_builder_stats WHERE name=?", name)
			tx.Exec("DELETE FROM group_builder_stats WHERE name=?", name)
			tx.Exec("DELETE FROM group_daily_builder_stats WHERE name=?", name)
		})
	}
	RespondAndCleanup(ctx, e, fmt.Sprintf("❌ سازنده «%s» با %d بازی از آمار حذف شد.", name, ss[idx-1].Count), -1)
}

func mergeBuilder(ctx context.Context, e *events.Event, match []string) {
	srcArg := strings.TrimSpace(match[1])
	dstArg := strings.TrimSpace(match[2])

	statsMu.Lock()
	defer statsMu.Unlock()

	var ss []struct {
		Name  string
		Count int
	}
	for k, v := range gameStats.BuilderStats {
		ss = append(ss, struct {
			Name  string
			Count int
		}{k, v})
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i].Count > ss[j].Count })

	resolve := func(arg string) string {
		if idx, err := strconv.Atoi(arg); err == nil {
			if idx >= 1 && idx <= len(ss) {
				return ss[idx-1].Name
			}
			return ""
		}
		return arg
	}

	src := resolve(srcArg)
	dst := resolve(dstArg)

	if src == "" || dst == "" || src == dst {
		RespondAndCleanup(ctx, e, "⚠️ نام‌ها باید متفاوت و معتبر باشند.", -1)
		return
	}

	srcCount := gameStats.BuilderStats[src]
	if srcCount == 0 {
		RespondAndCleanup(ctx, e, fmt.Sprintf("⚠️ سازنده «%s» در آمار نیست.", src), -1)
		return
	}
	dailySrc := gameStats.DailyBuilderStats[src]
	delete(gameStats.BuilderStats, src)
	delete(gameStats.DailyBuilderStats, src)

	gameStats.BuilderStats[dst] += srcCount
	if dailySrc > 0 {
		gameStats.DailyBuilderStats[dst] += dailySrc
	}

	for _, gs := range groupGameStats {
		c := gs.BuilderStats[src]
		if c > 0 {
			gs.BuilderStats[dst] += c
		}
		dc := gs.DailyBuilderStats[src]
		if dc > 0 {
			gs.DailyBuilderStats[dst] += dc
		}
		delete(gs.BuilderStats, src)
		delete(gs.DailyBuilderStats, src)
	}

	if USE_DATABASE {
		EnqueueDB(func(tx *sql.Tx) {
			tx.Exec("DELETE FROM builder_stats WHERE name=?", src)
			tx.Exec("DELETE FROM daily_builder_stats WHERE name=?", src)
			tx.Exec("INSERT INTO builder_stats(name, count) VALUES(?, ?) ON CONFLICT(name) DO UPDATE SET count = count + ?", dst, srcCount, srcCount)
			if dailySrc > 0 {
				tx.Exec("INSERT INTO daily_builder_stats(name, count) VALUES(?, ?) ON CONFLICT(name) DO UPDATE SET count = count + ?", dst, dailySrc, dailySrc)
			}
			tx.Exec("DELETE FROM group_builder_stats WHERE name=?", src)
			tx.Exec("DELETE FROM group_daily_builder_stats WHERE name=?", src)
			tx.Exec("UPDATE OR IGNORE group_builder_stats SET name = ? WHERE name = ?", dst, src)
			tx.Exec("UPDATE OR IGNORE group_daily_builder_stats SET name = ? WHERE name = ?", dst, src)
		})
	}
	RespondAndCleanup(ctx, e, fmt.Sprintf("🔀 «%s» (%d بازی) در «%s» ادغام شد.\nمجموع جدید «%s»: %d بازی", src, srcCount, dst, dst, gameStats.BuilderStats[dst]), -1)
}

func deleteScenarioStats(ctx context.Context, e *events.Event, match []string) {
	idx, _ := strconv.Atoi(DIGIT_REPLACER.Replace(match[1]))
	statsMu.Lock()
	defer statsMu.Unlock()

	var ss []struct {
		Name  string
		Count int
	}
	for k, v := range gameStats.ScenarioStats {
		ss = append(ss, struct {
			Name  string
			Count int
		}{k, v})
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i].Count > ss[j].Count })

	if len(ss) == 0 {
		RespondAndCleanup(ctx, e, "📭 آماری برای حذف وجود ندارد.", -1)
		return
	}
	if idx < 1 || idx > len(ss) {
		RespondAndCleanup(ctx, e, fmt.Sprintf("⚠️ شماره باید بین ۱ تا %d باشد.", len(ss)), -1)
		return
	}

	name := ss[idx-1].Name
	delete(gameStats.ScenarioStats, name)
	delete(gameStats.DailyScenarioStats, name)
	for _, gs := range groupGameStats {
		delete(gs.ScenarioStats, name)
		delete(gs.DailyScenarioStats, name)
	}

	if USE_DATABASE {
		EnqueueDB(func(tx *sql.Tx) {
			tx.Exec("DELETE FROM scenario_stats WHERE name=?", name)
			tx.Exec("DELETE FROM daily_scenario_stats WHERE name=?", name)
			tx.Exec("DELETE FROM group_scenario_stats WHERE name=?", name)
			tx.Exec("DELETE FROM group_daily_scenario_stats WHERE name=?", name)
		})
	}
	RespondAndCleanup(ctx, e, fmt.Sprintf("❌ سناریو «%s» با %d بازی از آمار حذف شد.", name, ss[idx-1].Count), -1)
}

func mergeScenario(ctx context.Context, e *events.Event, match []string) {
	srcArg := strings.TrimSpace(match[1])
	dstArg := strings.TrimSpace(match[2])

	statsMu.Lock()
	defer statsMu.Unlock()

	var ss []struct {
		Name  string
		Count int
	}
	for k, v := range gameStats.ScenarioStats {
		ss = append(ss, struct {
			Name  string
			Count int
		}{k, v})
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i].Count > ss[j].Count })

	resolve := func(arg string) string {
		if idx, err := strconv.Atoi(arg); err == nil {
			if idx >= 1 && idx <= len(ss) {
				return ss[idx-1].Name
			}
			return ""
		}
		return arg
	}

	src := resolve(srcArg)
	dst := resolve(dstArg)

	if src == "" || dst == "" || src == dst {
		RespondAndCleanup(ctx, e, "⚠️ نام‌ها باید متفاوت و معتبر باشند.", -1)
		return
	}

	srcCount := gameStats.ScenarioStats[src]
	if srcCount == 0 {
		RespondAndCleanup(ctx, e, fmt.Sprintf("⚠️ سناریو «%s» در آمار نیست.", src), -1)
		return
	}
	dailySrc := gameStats.DailyScenarioStats[src]
	delete(gameStats.ScenarioStats, src)
	delete(gameStats.DailyScenarioStats, src)

	gameStats.ScenarioStats[dst] += srcCount
	if dailySrc > 0 {
		gameStats.DailyScenarioStats[dst] += dailySrc
	}

	for _, gs := range groupGameStats {
		c := gs.ScenarioStats[src]
		if c > 0 {
			gs.ScenarioStats[dst] += c
		}
		dc := gs.DailyScenarioStats[src]
		if dc > 0 {
			gs.DailyScenarioStats[dst] += dc
		}
		delete(gs.ScenarioStats, src)
		delete(gs.DailyScenarioStats, src)
	}

	if USE_DATABASE {
		EnqueueDB(func(tx *sql.Tx) {
			tx.Exec("DELETE FROM scenario_stats WHERE name=?", src)
			tx.Exec("DELETE FROM daily_scenario_stats WHERE name=?", src)
			tx.Exec("INSERT INTO scenario_stats(name, count) VALUES(?, ?) ON CONFLICT(name) DO UPDATE SET count = count + ?", dst, srcCount, srcCount)
			if dailySrc > 0 {
				tx.Exec("INSERT INTO daily_scenario_stats(name, count) VALUES(?, ?) ON CONFLICT(name) DO UPDATE SET count = count + ?", dst, dailySrc, dailySrc)
			}
			tx.Exec("DELETE FROM group_scenario_stats WHERE name=?", src)
			tx.Exec("DELETE FROM group_daily_scenario_stats WHERE name=?", src)
			tx.Exec("UPDATE OR IGNORE group_scenario_stats SET name = ? WHERE name = ?", dst, src)
			tx.Exec("UPDATE OR IGNORE group_daily_scenario_stats SET name = ? WHERE name = ?", dst, src)
		})
	}
	RespondAndCleanup(ctx, e, fmt.Sprintf("🔀 سناریو «%s» (%d بازی) در «%s» ادغام شد.\nمجموع جدید «%s»: %d بازی", src, srcCount, dst, dst, gameStats.ScenarioStats[dst]), -1)
}

func deleteSpecificGame(ctx context.Context, e *events.Event, match []string) {
	gameID := fmt.Sprintf("%02d", func() int { i, _ := strconv.Atoi(DIGIT_REPLACER.Replace(match[1])); return i }())
	chat := e.ObjectGUID()

	game := RemoveGameByID(chat, gameID, true)
	if game == nil {
		RespondAndCleanup(ctx, e, fmt.Sprintf("🔍 بازی با شناسه `%s` یافت نشد.", gameID), -1)
		return
	}
	RespondAndCleanup(ctx, e, fmt.Sprintf("✅ بازی `%s` با موفقیت حذف شد.", gameID), -1)
}

func deleteAllGames(ctx context.Context, e *events.Event) {
	chat := e.ObjectGUID()
	activeGamesMu.Lock()
	games := activeGames[chat]
	if games == nil {
		activeGamesMu.Unlock()
		RespondAndCleanup(ctx, e, "📭 هیچ بازی فعالی نیست.", -1)
		return
	}
	var startIDs []string
	var forwardIDs []string
	for _, game := range games {
		if game.WarnTimer != nil {
			game.WarnTimer.Stop()
		}
		if game.CloseTimer != nil {
			game.CloseTimer.Stop()
		}
		startIDs = append(startIDs, game.StartMsgID)
		if game.ForwardedMsgID != "" {
			forwardIDs = append(forwardIDs, game.ForwardedMsgID)
		}
	}
	delete(activeGames, chat)
	activeGamesMu.Unlock()

	startMessageIndexMu.Lock()
	for _, id := range startIDs {
		delete(startMessageIndex, chat+":"+id)
	}
	startMessageIndexMu.Unlock()

	DeleteMessagesSafe(ctx, chat, startIDs)
	DeleteMessagesSafe(ctx, TARGET_GROUP_GUID, forwardIDs)
	RespondAndCleanup(ctx, e, "🧹 تمامی بازی‌ها حذف شدند.", -1)
}

func handleStatsCommand(ctx context.Context, e *events.Event) {
	chat := e.ObjectGUID()
	replyID := e.ReplyMessageID()

	if replyID != "" {
		var targetGUID string
		startMessageIndexMu.RLock()
		indexed, ok := startMessageIndex[chat+":"+replyID]
		startMessageIndexMu.RUnlock()

		if ok {
			activeGamesMu.RLock()
			game := activeGames[chat][indexed]
			if game != nil {
				targetGUID = game.CreatorGUID
			}
			activeGamesMu.RUnlock()
		}

		if targetGUID == "" {
			res, _ := acc.GetMessagesByID(ctx, chat, []string{replyID})
			msgs := res.Slice("messages")
			if len(msgs) > 0 {
				targetGUID = events.New(msgs[0].(map[string]any)).AuthorGUID()
			}
		}

		if targetGUID != "" {
			RespondAndCleanup(ctx, e, GenerateUserStatsText(ctx, targetGUID, chat), -1)
			return
		}
	}

	text := NormalizeText(e.Text())
	parts := strings.SplitN(text, " ", 2)
	second := ""
	if len(parts) > 1 {
		second = FirstWord(parts[1])
	}

	if TOTAL_STATS_WORDS[second] {
		RespondAndCleanup(ctx, e, GenerateTotalStatsText(chat), -1)
	} else {
		RespondAndCleanup(ctx, e, GenerateDailyStatsText(chat), -1)
	}
}

func GenerateTotalStatsText(groupGUID string) string {
	EnsureDailyReset()
	statsMu.RLock()
	defer statsMu.RUnlock()
	stats := gameStats
	title := "ربات دُنیتو"
	if groupGUID != "" {
		if gs, ok := groupGameStats[groupGUID]; ok {
			stats = gs
		} else {
			stats = NewStats()
		}
		title = "گروه"
	}

	date := stats.LastReset
	if date == "" {
		date = GetShamsiDate()
	}

	return fmt.Sprintf("📊 آمار کلی %s\n\n📅 تاریخ: %s\n🎮 کل بازی‌های برگزار شده: %d\n\n🥇 سازنده‌ها (کل):\n%s\n\n🎭 سناریوها (کل):\n%s",
		title, date, stats.TotalGames, formatScenarioLine(stats.BuilderStats), formatScenarioLine(stats.ScenarioStats))
}

func GenerateUserStatsText(ctx context.Context, userGUID string, groupGUID string) string {
	EnsureDailyReset()
	statsMu.RLock()
	stats := gameStats
	scope := "ربات"
	if groupGUID != "" {
		if gs, ok := groupGameStats[groupGUID]; ok {
			stats = gs
		}
		scope = "گروه"
	}

	u := stats.UserStats[userGUID]
	du := stats.DailyUserStats[userGUID]
	date := stats.LastReset
	if date == "" {
		date = GetShamsiDate()
	}
	statsMu.RUnlock()

	display := "ناشناس"
	info, _ := acc.GetUserInfo(ctx, userGUID)
	if info != nil {
		display = FormatUserName(info)
	}
	display = SanitizeMdName(display)

	totalGames := 0
	var totalTop map[string]int
	if u != nil {
		totalGames = u.Games
		totalTop = u.Scenarios
	}

	dailyGames := 0
	var dailyTop map[string]int
	if du != nil {
		dailyGames = du.Games
		dailyTop = du.Scenarios
	}

	return fmt.Sprintf("👤 آمار اختصاصی کاربر (%s)\n\n🆔 کاربر: [%s](%s)\n🔑 شناسه: `%s`\n📅 تاریخ آمار روزانه: %s\n\n🎮 کل بازی‌ها: %d\n📈 بازی‌های روز: %d\n\n🏆 سناریوهای برتر (کل):\n%s\n\n📅 سناریوهای برتر (روز):\n%s",
		scope, display, userGUID, userGUID, date, totalGames, dailyGames, formatScenarioLine(totalTop), formatScenarioLine(dailyTop))
}

func formatScenarioLine(items map[string]int) string {
	if len(items) == 0 {
		return "📭 داده‌ای موجود نیست"
	}
	type kv struct {
		Key string
		Val int
	}
	var ss []kv
	for k, v := range items {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i].Val > ss[j].Val })
	var lines []string
	for i, kv := range ss {
		if i >= 10 {
			break
		}
		lines = append(lines, fmt.Sprintf("%d. %s: %d بازی", i+1, kv.Key, kv.Val))
	}
	return strings.Join(lines, "\n")
}

func ScheduleAutoStats() {
	if !autoStatsEnabled || autoStatsTime == "" {
		return
	}
	now := time.Now()
	parts := strings.Split(autoStatsTime, ":")
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])
	target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if target.Before(now) || target.Equal(now) {
		target = target.Add(24 * time.Hour)
	}
	delay := time.Until(target)
	autoStatsHandle = time.AfterFunc(delay, autoStatsFire)
}

func autoStatsFire() {
	autoStatsMu.Lock()
	if !autoStatsEnabled {
		autoStatsMu.Unlock()
		return
	}
	now := time.Now()
	currentTime := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())
	if currentTime != autoStatsTime {
		ScheduleAutoStats()
		autoStatsMu.Unlock()
		return
	}
	today := GetShamsiDate()
	if today == lastAutoStatsDate || !IS_BOT_ACTIVE {
		ScheduleAutoStats()
		autoStatsMu.Unlock()
		return
	}
	lastAutoStatsDate = today
	autoStatsMu.Unlock()

	go func() {
		botGroupsMu.RLock()
		var groups []string
		for guid := range botGroups {
			if guid != BROADCAST_GROUP_GUID && IsGroupSubscribed(guid) {
				groups = append(groups, guid)
			}
		}
		botGroupsMu.RUnlock()

		for _, guid := range groups {
			text := GenerateDailyStatsText(guid)
			SendMessageSafe(context.Background(), guid, text, "", 0, enums.ParseModeNone)
			time.Sleep(2 * time.Second)
		}
	}()
	ScheduleAutoStats()
}

func getForwardQueue() chan map[string]any {
	if forwardQueue == nil {
		forwardQueue = make(chan map[string]any, 1000)
		for i := 0; i < FORWARD_WORKERS; i++ {
			go forwardWorker()
		}
	}
	return forwardQueue
}

func forwardWorker() {
	for task := range forwardQueue {
		fromGUID := task["from_guid"].(string)
		msgID := task["message_id"].(string)
		var target string
		if t, ok := task["to_guid"].(string); ok && t != "" {
			target = t
		} else {
			target = TARGET_GROUP_GUID
		}

		var res *events.Event
		var err error
		for retries := 0; retries < 3; retries++ {
			res, err = acc.ForwardMessages(context.Background(), fromGUID, []string{msgID}, target, false, false)
			if err == nil {
				break
			}
			time.Sleep(FORWARD_RETRY_DELAY)
		}

		if err == nil && res != nil {
			if chatID, ok := task["chat_id"].(string); ok {
				if gameID, ok := task["game_id"].(string); ok {
					activeGamesMu.Lock()
					if g := activeGames[chatID][gameID]; g != nil {
						g.ForwardedMsgID = res.MessageID()
					}
					activeGamesMu.Unlock()
				}
			}
		}
	}
}

func pvListGroups(ctx context.Context, e *events.Event) {
	botGroupsMu.RLock()
	var list []struct {
		Guid  string
		Title string
		Count int
	}
	for g, info := range botGroups {
		t, _ := info["title"].(string)
		c, _ := info["member_count"].(int)
		list = append(list, struct {
			Guid  string
			Title string
			Count int
		}{g, t, c})
	}
	botGroupsMu.RUnlock()

	sort.Slice(list, func(i, j int) bool { return list[i].Title < list[j].Title })

	lastGroupDisplayMu.Lock()
	lastGroupDisplay = nil
	for _, item := range list {
		lastGroupDisplay = append(lastGroupDisplay, struct {
			GUID        string
			Title       string
			MemberCount int
		}{item.Guid, item.Title, item.Count})
	}
	lastGroupDisplaySavedAt = GetShamsiNow()
	lastGroupDisplayMu.Unlock()
	PersistGroupDisplay()

	var lines []string
	lines = append(lines, fmt.Sprintf("📋 لیست گروه‌های ربات (%d گروه)\n", len(list)))
	lines = append(lines, fmt.Sprintf("🕐 آخرین بروزرسانی: %s\n", lastGroupDisplaySavedAt))
	for i, item := range list {
		subText := "❌ بدون اشتراک"
		if item.Guid == BROADCAST_GROUP_GUID {
			subText = "📢 گروه پخش"
		} else if days := DaysRemainingSubscription(item.Guid); days > 0 {
			subText = fmt.Sprintf("✅ %d روز مانده", days)
		}
		lines = append(lines, fmt.Sprintf("%d. %s\n   👥 %d عضو\n   💎 %s\n   🆔 `%s`\n", i+1, item.Title, item.Count, subText, item.Guid))
	}

	statsMu.RLock()
	lines = append(lines, "\n📊 آمار سراسری ربات:")
	lines = append(lines, fmt.Sprintf("🎮 کل بازی‌ها: %d", gameStats.TotalGames))
	lines = append(lines, fmt.Sprintf("📈 بازی‌های امروز: %d", gameStats.DailyGames))
	statsMu.RUnlock()

	text := strings.Join(lines, "\n")
	if len(text) > 3500 {
		text = text[:3500] + "\n… (ادامه کوتاه شد)"
	}
	RespondAndCleanup(ctx, e, text, -1)
}

func pvAddGroup(ctx context.Context, e *events.Event, link string) {
	res, err := acc.Join(ctx, link)
	if err != nil {
		SendMessageSafe(ctx, e.ObjectGUID(), fmt.Sprintf("❌ خطا: %v", err), "", -1, enums.ParseModeNone)
		return
	}
	guid := res.ObjectGUID()
	RegisterGroup(ctx, guid, "", 0)
	SendMessageSafe(ctx, e.ObjectGUID(), fmt.Sprintf("✅ به گروه جوین شدیم.\n🆔 `%s`\n\nبرای فعال‌سازی، دستور «افزایش اشتراک YYYY/MM/DD» را در خود گروه بفرستید.", guid), "", -1, enums.ParseModeNone)
}

func pvLeaveGroup(ctx context.Context, e *events.Event, match []string) {
	idx, _ := strconv.Atoi(DIGIT_REPLACER.Replace(match[1]))
	lastGroupDisplayMu.RLock()
	if idx > 0 && idx <= len(lastGroupDisplay) {
		item := lastGroupDisplay[idx-1]
		err := acc.LeaveChat(ctx, item.GUID)
		if err == nil {
			botGroupsMu.Lock()
			delete(botGroups, item.GUID)
			botGroupsMu.Unlock()
			RespondAndCleanup(ctx, e, fmt.Sprintf("✅ از گروه «%s» لفت دادیم.", item.Title), -1)
		} else {
			RespondAndCleanup(ctx, e, "❌ خطا در لفت دادن گروه.", -1)
		}
	} else {
		RespondAndCleanup(ctx, e, "⚠️ شماره نامعتبر است.", -1)
	}
	lastGroupDisplayMu.RUnlock()
}

func BackupDatabase() {
	backupMu.Lock()
	defer backupMu.Unlock()

	if !USE_DATABASE || dbConnection == nil {
		return
	}

	backupPath := DATABASE_PATH + ".backup"
	os.Remove(backupPath)

	_, err := dbConnection.Exec("VACUUM INTO ?", backupPath)
	if err != nil {
		log.Printf("Backup error: %v", err)
		return
	}

	_, err = acc.UploadFromPath(context.Background(), backupPath)
	if err == nil {
		os.Remove(backupPath)
	}
}

func backupLoop() {
	for {
		now := time.Now()
		backupMu.Lock()
		parts := strings.Split(backupTime, ":")
		hour, _ := strconv.Atoi(parts[0])
		minute, _ := strconv.Atoi(parts[1])
		backupMu.Unlock()

		target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
		if target.Before(now) || target.Equal(now) {
			target = target.Add(24 * time.Hour)
		}
		time.Sleep(time.Until(target))
		BackupDatabase()
	}
}

func setBackupTime(ctx context.Context, e *events.Event, match []string) {
	backupMu.Lock()
	backupTime = DIGIT_REPLACER.Replace(match[1])
	backupMu.Unlock()
	RespondAndCleanup(ctx, e, fmt.Sprintf("📦 زمان بکاپ خودکار روی ساعت %s تنظیم شد.", backupTime), -1)
}

func githubSyncLoop() {
	for {
		now := time.Now()
		target := time.Date(now.Year(), now.Month(), now.Day(), GITHUB_SYNC_HOUR, GITHUB_SYNC_MINUTE, 0, 0, now.Location())
		if target.Before(now) || target.Equal(now) {
			target = target.Add(24 * time.Hour)
		}
		time.Sleep(time.Until(target))

		var newGUID string
		for i := 0; i < GITHUB_FETCH_RETRIES; i++ {
			resp, err := http.Get(GITHUB_GUID_URL)
			if err == nil {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				re := regexp.MustCompile(`u[0-9A-Za-z]{10,}`)
				m := re.FindString(string(body))
				if m != "" {
					newGUID = m
					break
				}
			}
			time.Sleep(time.Duration(i*i) * time.Second)
		}

		if newGUID != "" && newGUID != MAIN_ADMIN_GUID {
			adminGuidsMu.Lock()
			MAIN_ADMIN_GUID = newGUID
			adminGuids[newGUID] = true
			adminGuidsMu.Unlock()

			if USE_DATABASE {
				EnqueueDB(func(tx *sql.Tx) {
					tx.Exec("INSERT OR REPLACE INTO meta(key, value) VALUES('main_admin_guid', ?)", newGUID)
				})
			}
		}
	}
}

func dispatchText(ctx context.Context, e *events.Event, normalized string, isAdmin, isMainAdmin bool) {
	if isMainAdmin {
		if STOP_REGEX.MatchString(normalized) {
			adminStop(ctx, e)
			return
		}
		if START_BOT_REGEX.MatchString(normalized) {
			adminStart(ctx, e)
			return
		}
		if m := ADD_SUB_REGEX.FindStringSubmatch(normalized); m != nil {
			dateStr := DIGIT_REPLACER.Replace(m[1])
			if end, ok := ParseShamsiDate(dateStr); ok {
				groupSubscriptionsMu.Lock()
				groupSubscriptions[e.ObjectGUID()] = end
				groupSubscriptionsMu.Unlock()
				if USE_DATABASE {
					EnqueueDB(func(tx *sql.Tx) {
						tx.Exec("INSERT INTO bot_groups(group_guid, subscription_end) VALUES(?, ?) ON CONFLICT(group_guid) DO UPDATE SET subscription_end = excluded.subscription_end", e.ObjectGUID(), end.Format("2006/01/02"))
					})
				}
				days := DaysRemainingSubscription(e.ObjectGUID())
				RespondAndCleanup(ctx, e, fmt.Sprintf("✅ اشتراک گروه تا %s فعال شد.\n📅 مدت باقی‌مانده: %d روز", dateStr, days), -1)
			}
			return
		}
		if m := ADD_ADMIN_REGEX.FindStringSubmatch(normalized); m != nil {
			addAdmin(ctx, e, m)
			return
		}
		if m := REMOVE_ADMIN_REGEX.FindStringSubmatch(normalized); m != nil {
			removeAdmin(ctx, e, m)
			return
		}
		if m := ADD_SUPER_ADMIN_REGEX.FindStringSubmatch(normalized); m != nil {
			addSuperAdmin(ctx, e, m)
			return
		}
		if m := REMOVE_SUPER_ADMIN_REGEX.FindStringSubmatch(normalized); m != nil {
			removeSuperAdmin(ctx, e, m)
			return
		}
		if LIST_SUPER_ADMINS_REGEX.MatchString(normalized) {
			listSuperAdmins(ctx, e)
			return
		}
		if m := BROADCAST_ALL_REGEX.FindStringSubmatch(normalized); m != nil {
			broadcastAll(ctx, e, m)
			return
		}
	}

	if isAdmin {
		if m := AUTO_STATS_SET_REGEX.FindStringSubmatch(normalized); m != nil {
			setAutoStats(ctx, e, m)
			return
		}
		if AUTO_STATS_DISABLE_REGEX.MatchString(normalized) {
			disableAutoStats(ctx, e)
			return
		}
		if m := SET_AUTO_DELETE_REGEX.FindStringSubmatch(normalized); m != nil {
			setAutoDelete(ctx, e, m)
			return
		}
		if DISABLE_AUTO_DELETE_REGEX.MatchString(normalized) {
			disableAutoDelete(ctx, e)
			return
		}
		if CLOSE_GROUP_REGEX.MatchString(normalized) {
			closeGroup(ctx, e)
			return
		}
		if OPEN_GROUP_REGEX.MatchString(normalized) {
			openGroup(ctx, e)
			return
		}
		if m := DELETE_MESSAGES_REGEX.FindStringSubmatch(normalized); m != nil {
			deleteMessagesCommand(ctx, e, m)
			return
		}
		if m := BLOCK_REGEX.FindStringSubmatch(normalized); m != nil {
			blockUser(ctx, e, m)
			return
		}
		if m := UNBLOCK_REGEX.FindStringSubmatch(normalized); m != nil {
			unblockUser(ctx, e, m)
			return
		}
		if m := SPECIAL_REGEX.FindStringSubmatch(normalized); m != nil {
			specialUser(ctx, e, m)
			return
		}
		if m := BAN_REGEX.FindStringSubmatch(normalized); m != nil {
			banUser(ctx, e, m)
			return
		}
		if m := UNBAN_REGEX.FindStringSubmatch(normalized); m != nil {
			unbanUser(ctx, e, m)
			return
		}
		if m := ADD_BUILDER_REGEX.FindStringSubmatch(normalized); m != nil {
			addBuilder(ctx, e, m)
			return
		}
		if m := REMOVE_BUILDER_REGEX.FindStringSubmatch(normalized); m != nil {
			removeBuilder(ctx, e, m)
			return
		}
		if m := DELETE_BUILDER_STATS_REGEX.FindStringSubmatch(normalized); m != nil {
			deleteBuilderStats(ctx, e, m)
			return
		}
		if m := MERGE_BUILDER_REGEX.FindStringSubmatch(normalized); m != nil {
			mergeBuilder(ctx, e, m)
			return
		}
		if m := DELETE_SCENARIO_STATS_REGEX.FindStringSubmatch(normalized); m != nil {
			deleteScenarioStats(ctx, e, m)
			return
		}
		if m := MERGE_SCENARIO_REGEX.FindStringSubmatch(normalized); m != nil {
			mergeScenario(ctx, e, m)
			return
		}
		if TAG_ALL_REGEX.MatchString(normalized) {
			tagAll(ctx, e)
			return
		}
		if PIN_REGEX.MatchString(normalized) {
			pinByReply(ctx, e)
			return
		}
		if UNPIN_REGEX.MatchString(normalized) {
			unpinByReply(ctx, e)
			return
		}
		if m := SET_BACKUP_TIME_REGEX.FindStringSubmatch(normalized); m != nil {
			setBackupTime(ctx, e, m)
			return
		}
	}

	if LIST_GAMES_REGEX.MatchString(normalized) {
		listGames(ctx, e)
		return
	}
	if DELETE_ALL_REGEX.MatchString(normalized) {
		deleteAllGames(ctx, e)
		return
	}
	if m := DELETE_GAME_REGEX.FindStringSubmatch(normalized); m != nil {
		deleteSpecificGame(ctx, e, m)
		return
	}
	if LIST_BUILDERS_REGEX.MatchString(normalized) {
		listBuilders(ctx, e)
		return
	}
	if STATS_TOTAL_REGEX.MatchString(normalized) || STATS_BASIC_REGEX.MatchString(normalized) {
		handleStatsCommand(ctx, e)
		return
	}
	if HELP_REGEX.MatchString(normalized) {
		SendMessageSafe(ctx, e.ObjectGUID(), HELP_TEXT, e.MessageID(), -1, enums.ParseModeNone)
		acc.DeleteMessages(ctx, e.ObjectGUID(), []string{e.MessageID()})
		return
	}
}

func IsAdmin(guid string) bool {
	adminGuidsMu.RLock()
	defer adminGuidsMu.RUnlock()
	return adminGuids[guid]
}

func IsSuperAdmin(guid string) bool {
	superAdminGuidsMu.RLock()
	defer superAdminGuidsMu.RUnlock()
	return superAdminGuids[guid]
}

func listGames(ctx context.Context, e *events.Event) {
	chat := e.ObjectGUID()
	activeGamesMu.RLock()
	games := activeGames[chat]
	if len(games) == 0 {
		activeGamesMu.RUnlock()
		RespondAndCleanup(ctx, e, "📭 لیست بازی‌ها خالی است.", -1)
		return
	}
	var lines []string
	lines = append(lines, "📋 لیست بازی‌های فعال:")
	for _, g := range games {
		hours := time.Since(g.StartMono).Hours()
		warn := ""
		if hours >= WARNING_AFTER_HOURS {
			warn = "⚠️ "
		}
		lines = append(lines, fmt.Sprintf("🆔 شناسه بازی: `%s` %s\n🎭 سناریو: %s\n🛠️ سازنده: %s\n⏰ زمان استارت: %s\n⏳ مدت: %.1f ساعت\n➖➖➖➖➖➖➖➖",
			g.ID, warn, g.ScenarioCaption, g.Builder, g.StartTime, hours))
	}
	activeGamesMu.RUnlock()
	RespondAndCleanup(ctx, e, strings.Join(lines, "\n"), -1)
}

type AllMessagesMatcher struct{}

func (m *AllMessagesMatcher) Kind() string { return "MessageUpdates" }
func (m *AllMessagesMatcher) Match(ctx context.Context, raw map[string]any) bool { return true }

func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	InitDB()
	defer func() {
		if dbConnection != nil {
			dbConnection.Close()
		}
	}()

	var err error
	acc, err = account.New(
		account.WithFileSession("donito_session.json"),
	)
	if err != nil {
		log.Fatalf("Account error: %v", err)
	}
	defer acc.Close()

	acc.AddListener(func(ctx context.Context, e *events.Event) error {
		HandleUpdate(ctx, e)
		return nil
	}, &AllMessagesMatcher{})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if USE_DATABASE {
		go backupLoop()
	}
	go githubSyncLoop()
	go getForwardQueue()

	log.Println("Bot started.")
	if err := acc.Run(ctx, ""); err != nil && err != context.Canceled {
		log.Fatalf("Run error: %v", err)
	}
}
