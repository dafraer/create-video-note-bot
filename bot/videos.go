package bot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/go-telegram/bot"
	tgbotapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
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
	videoNote, err := b.cropVideoNote(ctx, data, video.Height, video.Width)
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
		Data:     bytes.NewReader(videoNote)}, Length: min(video.Height, video.Width)})
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
func (b *Bot) cropVideoNote(ctx context.Context, data []byte, height, width int) ([]byte, error) {
	//Create input and output files
	fileIn := uuid.New().String() + ".mp4"
	fileOut := uuid.New().String() + ".mp4"
	if err := os.WriteFile(fileIn, data, 0644); err != nil {
		return nil, err
	}

	//height and width of the resulting video
	outSize := min(height, width)

	//x and y are starting coordinates from which the video will be cropped
	x, y := 0, 0
	if height > width {
		y = (height - outSize) / 2
	} else if width > height {
		x = (width - outSize) / 2
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", fileIn, "-vf", fmt.Sprintf("crop=%d:%d:%d:%d", outSize, outSize, x, y), fileOut)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	//Run the command
	if err := cmd.Run(); err != nil {
		b.logger.Debugw("stderr of ffmpeg", "error", stderr.String())
		return nil, err
	}

	//Read the resulting file
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
