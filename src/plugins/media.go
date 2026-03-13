package plugins

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func init() {
	Register(&Command{
		Pattern:  "mp3",
		Category: "media",
		Func:     mp3Cmd,
	})
	Register(&Command{
		Pattern:  "black",
		Category: "media",
		Func:     blackCmd,
	})
	Register(&Command{
		Pattern:  "trim",
		Category: "media",
		Func:     trimCmd,
	})
}

func runFFmpeg(args ...string) error {
	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w\n%s", err, string(out))
	}
	return nil
}

func mp3Cmd(ctx *Context) error {
	quoted := ctx.Quoted()
	if quoted == nil || quoted.Message == nil {
		ctx.Reply(T().MediaNoReply)
		return nil
	}

	var data []byte
	var err error
	if quoted.Message.GetAudioMessage() != nil {
		data, err = ctx.Client.Download(context.Background(), quoted.Message.GetAudioMessage())
	} else if quoted.Message.GetVideoMessage() != nil {
		data, err = ctx.Client.Download(context.Background(), quoted.Message.GetVideoMessage())
	} else {
		ctx.Reply(T().MediaNoReply)
		return nil
	}
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	ctx.Reply(T().MediaProcessing)

	tmp, err := os.MkdirTemp("", "mack-mp3-*")
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}
	defer os.RemoveAll(tmp)

	inFile := filepath.Join(tmp, "input")
	outFile := filepath.Join(tmp, "output.mp3")

	if err = os.WriteFile(inFile, data, 0600); err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	if err = runFFmpeg("-y", "-i", inFile, "-vn", "-ar", "44100", "-ac", "2", "-b:a", "192k", outFile); err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	mp3Bytes, err := os.ReadFile(outFile)
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	if err := ctx.SendAudio(mp3Bytes, "audio/mpeg"); err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
	}
	return nil
}

func blackCmd(ctx *Context) error {
	quoted := ctx.Quoted()
	if quoted == nil || quoted.Message == nil || quoted.Message.GetAudioMessage() == nil {
		ctx.Reply(T().MediaNoReply)
		return nil
	}

	data, err := ctx.Client.Download(context.Background(), quoted.Message.GetAudioMessage())
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	ctx.Reply(T().MediaProcessing)

	tmp, err := os.MkdirTemp("", "mack-black-*")
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}
	defer os.RemoveAll(tmp)

	inFile := filepath.Join(tmp, "input")
	outFile := filepath.Join(tmp, "output.mp4")

	if err = os.WriteFile(inFile, data, 0600); err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	if err = runFFmpeg(
		"-y",
		"-f", "lavfi", "-i", "color=c=black:s=640x360:r=25",
		"-i", inFile,
		"-shortest",
		"-c:v", "libx264", "-c:a", "aac",
		outFile,
	); err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	mp4Bytes, err := os.ReadFile(outFile)
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	if err := ctx.SendVideo(mp4Bytes, "video/mp4", ""); err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
	}
	return nil
}

func trimCmd(ctx *Context) error {
	if len(ctx.Args) < 1 {
		ctx.Reply(T().TrimUsage)
		return nil
	}

	start := ctx.Args[0]
	end := ""
	if len(ctx.Args) >= 2 {
		end = ctx.Args[1]
	}

	quoted := ctx.Quoted()
	if quoted == nil || quoted.Message == nil {
		ctx.Reply(T().MediaNoReply)
		return nil
	}

	isAudio := quoted.Message.GetAudioMessage() != nil
	isVideo := quoted.Message.GetVideoMessage() != nil
	if !isAudio && !isVideo {
		ctx.Reply(T().MediaNoReply)
		return nil
	}

	var data []byte
	var err error
	if isAudio {
		data, err = ctx.Client.Download(context.Background(), quoted.Message.GetAudioMessage())
	} else {
		data, err = ctx.Client.Download(context.Background(), quoted.Message.GetVideoMessage())
	}
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	ctx.Reply(T().MediaProcessing)

	ext := ".mp3"
	mediaType := whatsmeow.MediaAudio
	if isVideo {
		ext = ".mp4"
		mediaType = whatsmeow.MediaVideo
	}

	tmp, err := os.MkdirTemp("", "mack-trim-*")
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}
	defer os.RemoveAll(tmp)

	inFile := filepath.Join(tmp, "input")
	outFile := filepath.Join(tmp, "output"+ext)

	if err = os.WriteFile(inFile, data, 0600); err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	ffArgs := []string{"-y", "-ss", start}
	if end != "" {
		ffArgs = append(ffArgs, "-to", end)
	}
	ffArgs = append(ffArgs, "-i", inFile, "-c", "copy", outFile)

	if err = runFFmpeg(ffArgs...); err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	outBytes, err := os.ReadFile(outFile)
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	uploadResp, err := ctx.Client.Upload(context.Background(), outBytes, mediaType)
	if err != nil {
		ctx.Reply(fmt.Sprintf(T().MediaFailed, err.Error()))
		return nil
	}

	var msg *waProto.Message

	if isAudio {
		mime := quoted.Message.GetAudioMessage().GetMimetype()
		if mime == "" {
			mime = "audio/mpeg"
		}

		if strings.HasSuffix(ext, ".mp3") && strings.Contains(mime, "ogg") {
			mime = "audio/ogg; codecs=opus"
		}
		msg = &waProto.Message{
			AudioMessage: &waProto.AudioMessage{
				Mimetype:      proto.String(mime),
				URL:           proto.String(uploadResp.URL),
				DirectPath:    proto.String(uploadResp.DirectPath),
				MediaKey:      uploadResp.MediaKey,
				FileEncSHA256: uploadResp.FileEncSHA256,
				FileSHA256:    uploadResp.FileSHA256,
				FileLength:    proto.Uint64(uploadResp.FileLength),
			},
		}
	} else {
		mime := quoted.Message.GetVideoMessage().GetMimetype()
		if mime == "" {
			mime = "video/mp4"
		}
		msg = &waProto.Message{
			VideoMessage: &waProto.VideoMessage{
				Mimetype:      proto.String(mime),
				URL:           proto.String(uploadResp.URL),
				DirectPath:    proto.String(uploadResp.DirectPath),
				MediaKey:      uploadResp.MediaKey,
				FileEncSHA256: uploadResp.FileEncSHA256,
				FileSHA256:    uploadResp.FileSHA256,
				FileLength:    proto.Uint64(uploadResp.FileLength),
			},
		}
	}

	ctx.SendRawMsg(msg)
	return nil
}
