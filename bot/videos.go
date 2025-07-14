package bot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/go-telegram/bot"
	tgbotapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

const (
	maxVideoSize     = 10_000_000
	maxVideoDuration = 60
	telegramFileLink = "https://api.telegram.org/file/bot%s/%s"
)

// processMessage routes message to the handler functions
func (b *Bot) processVideo(ctx context.Context, update *models.Update) {
	b.logger.Infow("Video Received", "from", update.Message.From.Username)
	video := update.Message.Video

	//If video is too big gg
	if video.Duration > maxVideoDuration || video.FileSize > maxVideoSize {
		b.processVideoTooLarge(ctx, update)
		return
	}

	//Send the message that the request is being processed
	waitMsg, err := b.b.SendMessage(ctx, &tgbotapi.SendMessageParams{ChatID: update.Message.Chat.ID, Text: b.msgs.waitMsg[language(update.Message.From)]})
	if err != nil {
		b.logger.Errorw("Error sending message", "error", err)
		return
	}

	//Get file info to download the file
	fileInfo, err := b.b.GetFile(ctx, &bot.GetFileParams{
		FileID: video.FileID,
	})

	fileURL := fmt.Sprintf(telegramFileLink, b.b.Token(), fileInfo.FilePath)

	resp, err := http.Get(fileURL)
	if err != nil {
		b.logger.Errorw("Error downloading video from telegram server", "error", err)
		if _, err := b.b.DeleteMessage(ctx, &tgbotapi.DeleteMessageParams{ChatID: update.Message.Chat.ID, MessageID: waitMsg.ID}); err != nil {
			b.logger.Errorw("Error sending message", "error", err)
			return
		}
		b.sendErrorMessage(ctx, update)
		return
	}
	defer resp.Body.Close()

	//Read video from response
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		b.logger.Errorw("Error parsing response body", "error", err)
		if _, err := b.b.DeleteMessage(ctx, &tgbotapi.DeleteMessageParams{ChatID: update.Message.Chat.ID, MessageID: waitMsg.ID}); err != nil {
			b.logger.Errorw("Error sending message", "error", err)
			return
		}
		b.sendErrorMessage(ctx, update)
		return
	}

	//Crop video to square
	videoNote, err := b.cropVideoNote(data)
	if err != nil {
		b.logger.Errorw("Error cropping video", "error", err)
		if _, err := b.b.DeleteMessage(ctx, &tgbotapi.DeleteMessageParams{ChatID: update.Message.Chat.ID, MessageID: waitMsg.ID}); err != nil {
			b.logger.Errorw("Error sending message", "error", err)
			return
		}
		b.sendErrorMessage(ctx, update)
		return
	}

	//Delete waiting message
	if _, err := b.b.DeleteMessage(ctx, &tgbotapi.DeleteMessageParams{ChatID: update.Message.Chat.ID, MessageID: waitMsg.ID}); err != nil {
		b.logger.Errorw("Error sending message", "error", err)
		return
	}

	//Send VideoNote
	_, err = b.b.SendVideoNote(ctx, &tgbotapi.SendVideoNoteParams{ChatID: update.Message.Chat.ID, VideoNote: &models.InputFileUpload{
		Filename: "note.mp4",
		Data:     bytes.NewReader(videoNote)}})
	if err != nil {
		b.sendErrorMessage(ctx, update)
		b.logger.Errorw("error sending message", "error", err)
	}
}

// processVideoTooLarge notifies user that their video is too big
func (b *Bot) processVideoTooLarge(ctx context.Context, update *models.Update) {
	if _, err := b.b.SendMessage(ctx, &tgbotapi.SendMessageParams{ChatID: update.Message.Chat.ID, Text: b.msgs.videoTooLargeMsg[language(update.Message.From)]}); err != nil {
		b.logger.Errorw("error sending message", "error", err)
	}
}

// cropVideoNote takes a video and returns a square‑cropped mp4. It uses ffmpeg
func (b *Bot) cropVideoNote(data []byte) ([]byte, error) {
	//Create input and output files
	fileIn := uuid.New().String() + ".mp4"
	fileOut := uuid.New().String() + ".mp4"
	if err := os.WriteFile(fileIn, data, 0644); err != nil {
		return nil, err
	}

	err := ffmpeg.
		Input(fileIn).
		Output(fileOut,
			ffmpeg.KwArgs{
				"c:v":             "libx264",                                                                                              // H.264 encoder
				"profile:v":       "high",                                                                                                 // High profile
				"pix_fmt":         "yuv420p",                                                                                              // yuv420p
				"color_primaries": "bt709",                                                                                                // BT.709 primaries
				"color_trc":       "bt709",                                                                                                // BT.709 transfer
				"colorspace":      "bt709",                                                                                                // BT.709 matrix
				"vf":              `crop=min(in_w\,in_h):min(in_w\,in_h):(in_w-min(in_w\,in_h))/2:(in_h-min(in_w\,in_h))/2,scale=400:400`, // scale filter
				"b:v":             "986k",                                                                                                 // 986 kb/s video bitrate
				"r":               "30",                                                                                                   // 30 fps
			},
			ffmpeg.KwArgs{
				"c:a":        "aac",     // AAC LC encoder
				"profile:a":  "aac_low", // Low Complexity profile
				"ac":         "1",       // mono
				"ar":         "48000",   // 48 kHz
				"sample_fmt": "fltp",    // float planar
				"b:a":        "64k",     // 64 kb/s audio bitrate
			},
		).Run()

	if err != nil {
		return nil, err
	}

	f, err := os.Open(fileOut)
	if err != nil {
		return nil, err
	}
	cropped, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	//Delete files
	if err := os.Remove(fileOut); err != nil {
		return nil, err
	}
	if err := os.Remove(fileIn); err != nil {
		return nil, err
	}

	//Return cropped version
	return cropped, nil
}
