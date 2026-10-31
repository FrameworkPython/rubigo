package main

import (
	"fmt"
	"log"

	"github.com/farshadnobody/Rubingo/rubingo"
)

func main() {
	client, err := rubingo.NewClient("my_session",
		rubingo.WithProxy("http://user:pass@gate.decodo.com:10001"),
		rubingo.WithProxyEnabled(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}

	if err := client.Start(""); err != nil {
		log.Fatal(err)
	}

	// ═══════════════════════════════════════════════════════
	// بررسی وضعیت استثنا
	// ═══════════════════════════════════════════════════════
	fmt.Println("آپلود استثنا شده؟", client.IsExcludedFromProxy(rubingo.ProxyOpUpload))
	// false

	// ═══════════════════════════════════════════════════════
	// آپلود فایل با پروکسی (کند!)
	// ═══════════════════════════════════════════════════════
	client.SendMessage(rubingo.SendMessageOptions{
		ObjectGUID: "me",
		Text:       "عکس با پروکسی",
		FileInline: "/path/to/image.jpg",
		Type:       "Image",
	})

	// ═══════════════════════════════════════════════════════
	// استثنا کردن آپلود و دانلود (سرعت بیشتر)
	// ═══════════════════════════════════════════════════════
	client.ExcludeFromProxy(rubingo.ProxyOpUpload)
	client.ExcludeFromProxy(rubingo.ProxyOpDownload)

	fmt.Println("آپلود استثنا شده؟", client.IsExcludedFromProxy(rubingo.ProxyOpUpload))
	// true

	// ═══════════════════════════════════════════════════════
	// آپلود فایل بدون پروکسی (سریع!)
	// ═══════════════════════════════════════════════════════
	client.SendMessage(rubingo.SendMessageOptions{
		ObjectGUID: "me",
		Text:       "عکس بدون پروکسی (سریع‌تر)",
		FileInline: "/path/to/image.jpg",
		Type:       "Image",
	})

	// ═══════════════════════════════════════════════════════
	// برگرداندن به حالت عادی
	// ═══════════════════════════════════════════════════════
	client.IncludeInProxy(rubingo.ProxyOpUpload)

	fmt.Println("آپلود استثنا شده؟", client.IsExcludedFromProxy(rubingo.ProxyOpUpload))
	// false

	client.Disconnect()
}
