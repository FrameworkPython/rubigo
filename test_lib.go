package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/account"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
)

type AllMessagesMatcher struct{}

func (m *AllMessagesMatcher) Kind() string { return "MessageUpdates" }
func (m *AllMessagesMatcher) Match(ctx context.Context, raw map[string]any) bool { return true }

var sentMessages sync.Map

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	acc, err := account.New(
		account.WithFileSession("rubika_session.json"),
	)
	if err != nil {
		log.Fatalf("Account error: %v", err)
	}
	defer acc.Close()

	acc.AddListener(func(ctx context.Context, e *events.Event) error {
		if e.AuthorGUID() != acc.GUID() {
			return nil
		}

		if _, isSentByBot := sentMessages.Load(e.MessageID()); isSentByBot {
			return nil
		}

		chatGUID := e.ObjectGUID()
		msgID := e.MessageID()
		opts := account.SendOptions{
			ReplyToMessageID: msgID,
		}

		var res *events.Event
		var sendErr error

		if e.IsFileInline() {
			file := e.FileInline()
			fileType := file.Str("type")
			caption := e.Text()

			switch fileType {
			case "Image":
				res, sendErr = acc.SendPhoto(ctx, chatGUID, file, caption, opts)
			case "Video", "VideoMessage":
				res, sendErr = acc.SendVideo(ctx, chatGUID, file, caption, opts)
			case "Voice":
				res, sendErr = acc.SendVoice(ctx, chatGUID, file, caption, opts)
			case "Music":
				res, sendErr = acc.SendMusic(ctx, chatGUID, file, caption, opts)
			case "Gif":
				res, sendErr = acc.SendGif(ctx, chatGUID, file, caption, opts)
			default:
				res, sendErr = acc.SendDocument(ctx, chatGUID, file, caption, opts)
			}
		} else if e.Sticker() != nil {
			opts.Sticker = e.Sticker().Raw
			res, sendErr = acc.SendMessage(ctx, chatGUID, "", opts)
		} else if e.Text() != "" {
			res, sendErr = acc.SendMessage(ctx, chatGUID, e.Text(), opts)
		} else {
			return nil
		}

		if sendErr != nil {
			fmt.Printf("Error sending message: %v\n", sendErr)
		} else if res != nil {
			sentMessages.Store(res.MessageID(), true)
			fmt.Println("Replied successfully.")
		}

		return nil
	}, &AllMessagesMatcher{})

	fmt.Println("Echo bot started.")
	err = acc.Run(ctx, "")
	if err != nil && err != context.Canceled {
		log.Fatalf("Run error: %v", err)
	}
}
