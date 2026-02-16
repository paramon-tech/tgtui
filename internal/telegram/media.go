package telegram

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"github.com/gotd/td/tg"
)

type DownloadPhotoMsg struct {
	MessageID int
	Data      []byte
}

type DownloadPhotoErrorMsg struct {
	MessageID int
	Err       error
}

func (c *Client) DownloadPhoto(msgID int, info *MediaInfo) func() interface{} {
	return func() interface{} {
		if info == nil || info.PhotoThumbSize == "" {
			return DownloadPhotoErrorMsg{MessageID: msgID, Err: fmt.Errorf("no thumbnail available")}
		}

		loc := &tg.InputPhotoFileLocation{
			ID:            info.PhotoID,
			AccessHash:    info.PhotoAccessHash,
			FileReference: info.PhotoFileRef,
			ThumbSize:     info.PhotoThumbSize,
		}

		var buf bytes.Buffer
		// Use upload.getFile to download the thumbnail
		offset := 0
		for {
			result, err := c.api.UploadGetFile(c.ctx, &tg.UploadGetFileRequest{
				Location: loc,
				Offset:   int64(offset),
				Limit:    1024 * 1024, // 1MB chunks
			})
			if err != nil {
				return DownloadPhotoErrorMsg{MessageID: msgID, Err: err}
			}

			file, ok := result.(*tg.UploadFile)
			if !ok {
				return DownloadPhotoErrorMsg{MessageID: msgID, Err: fmt.Errorf("unexpected upload response type")}
			}

			if len(file.Bytes) == 0 {
				break
			}

			buf.Write(file.Bytes)

			if len(file.Bytes) < 1024*1024 {
				break
			}
			offset += len(file.Bytes)
		}

		if buf.Len() == 0 {
			return DownloadPhotoErrorMsg{MessageID: msgID, Err: io.ErrUnexpectedEOF}
		}

		return DownloadPhotoMsg{MessageID: msgID, Data: buf.Bytes()}
	}
}

func extractMediaInfo(media tg.MessageMediaClass) *MediaInfo {
	if media == nil {
		return nil
	}

	switch m := media.(type) {
	case *tg.MessageMediaPhoto:
		photo, ok := m.Photo.(*tg.Photo)
		if !ok {
			return nil
		}
		info := &MediaInfo{
			Type:            MediaPhoto,
			Label:           "[Photo]",
			PhotoID:         photo.ID,
			PhotoAccessHash: photo.AccessHash,
			PhotoFileRef:    photo.FileReference,
			PhotoDCID:       photo.DCID,
		}
		// Find best thumbnail for download
		thumbType, w, h := findThumbSize(photo.Sizes)
		info.PhotoThumbSize = thumbType
		info.Width = w
		info.Height = h
		return info

	case *tg.MessageMediaDocument:
		doc, ok := m.Document.(*tg.Document)
		if !ok {
			return nil
		}
		return extractDocMediaInfo(doc)

	case *tg.MessageMediaContact:
		name := m.FirstName
		if m.LastName != "" {
			name += " " + m.LastName
		}
		return &MediaInfo{
			Type:  MediaContact,
			Label: fmt.Sprintf("[Contact: %s]", name),
		}

	case *tg.MessageMediaGeo:
		return &MediaInfo{
			Type:  MediaLocation,
			Label: "[Location]",
		}

	case *tg.MessageMediaGeoLive:
		return &MediaInfo{
			Type:  MediaLocation,
			Label: "[Live Location]",
		}

	case *tg.MessageMediaVenue:
		return &MediaInfo{
			Type:  MediaLocation,
			Label: fmt.Sprintf("[Venue: %s]", m.Title),
		}

	case *tg.MessageMediaPoll:
		question := m.Poll.Question.Text
		if len([]rune(question)) > 40 {
			question = string([]rune(question)[:37]) + "..."
		}
		return &MediaInfo{
			Type:  MediaPoll,
			Label: fmt.Sprintf("[Poll: %s]", question),
		}

	case *tg.MessageMediaDice:
		return &MediaInfo{
			Type:  MediaOther,
			Label: fmt.Sprintf("[Dice: %s %d]", m.Emoticon, m.Value),
		}

	case *tg.MessageMediaWebPage:
		return nil

	case *tg.MessageMediaEmpty:
		return nil

	case *tg.MessageMediaUnsupported:
		return nil
	}

	return &MediaInfo{
		Type:  MediaOther,
		Label: "[Media]",
	}
}

func extractDocMediaInfo(doc *tg.Document) *MediaInfo {
	var (
		isSticker   bool
		isGif       bool
		isVoice     bool
		isVideo     bool
		isAudio     bool
		stickerAlt  string
		duration    float64
		audioTitle  string
		fileName    string
	)

	for _, attr := range doc.Attributes {
		switch a := attr.(type) {
		case *tg.DocumentAttributeSticker:
			isSticker = true
			stickerAlt = a.Alt
		case *tg.DocumentAttributeAnimated:
			isGif = true
		case *tg.DocumentAttributeAudio:
			duration = float64(a.Duration)
			if a.Voice {
				isVoice = true
			} else {
				isAudio = true
				audioTitle = a.Title
			}
		case *tg.DocumentAttributeVideo:
			duration = a.Duration
			if !isGif {
				isVideo = true
			}
		case *tg.DocumentAttributeFilename:
			fileName = a.FileName
		}
	}

	switch {
	case isSticker:
		label := "[Sticker"
		if stickerAlt != "" {
			label += " " + stickerAlt
		}
		label += "]"
		return &MediaInfo{
			Type:  MediaSticker,
			Label: label,
		}

	case isGif:
		return &MediaInfo{
			Type:  MediaAnimation,
			Label: "[GIF]",
		}

	case isVoice:
		return &MediaInfo{
			Type:  MediaVoice,
			Label: fmt.Sprintf("[Voice %s]", formatDuration(duration)),
		}

	case isVideo:
		return &MediaInfo{
			Type:     MediaVideo,
			Label:    fmt.Sprintf("[Video %s]", formatDuration(duration)),
			FileName: fileName,
			FileSize: doc.Size,
		}

	case isAudio:
		label := "[Audio"
		if audioTitle != "" {
			label += ": " + audioTitle
		}
		label += fmt.Sprintf(" (%s)]", formatDuration(duration))
		return &MediaInfo{
			Type:     MediaAudio,
			Label:    label,
			FileName: fileName,
			FileSize: doc.Size,
		}
	}

	// Generic document / file
	name := fileName
	if name == "" {
		name = "file"
		if ext := doc.MimeType; ext != "" {
			if parts := filepath.Ext(ext); parts != "" {
				name += parts
			}
		}
	}
	label := fmt.Sprintf("[Document: %s (%s)]", name, formatFileSize(doc.Size))
	return &MediaInfo{
		Type:     MediaDocument,
		Label:    label,
		FileName: fileName,
		FileSize: doc.Size,
	}
}

// findThumbSize picks the best thumbnail from a photo's sizes.
// Prefers "m" (320px), falls back to largest available.
func findThumbSize(sizes []tg.PhotoSizeClass) (thumbType string, w, h int) {
	// Prefer "m" size (~320px)
	for _, s := range sizes {
		switch sz := s.(type) {
		case *tg.PhotoSize:
			if sz.Type == "m" {
				return sz.Type, sz.W, sz.H
			}
		}
	}
	// Fall back to largest PhotoSize
	var best *tg.PhotoSize
	for _, s := range sizes {
		if sz, ok := s.(*tg.PhotoSize); ok {
			if best == nil || sz.W*sz.H > best.W*best.H {
				best = sz
			}
		}
	}
	if best != nil {
		return best.Type, best.W, best.H
	}
	// Last resort: any stripped/cached
	for _, s := range sizes {
		switch sz := s.(type) {
		case *tg.PhotoStrippedSize:
			return sz.Type, 0, 0
		case *tg.PhotoCachedSize:
			return sz.Type, sz.W, sz.H
		}
	}
	return "", 0, 0
}

func formatDuration(seconds float64) string {
	total := int(seconds)
	m := total / 60
	s := total % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func formatFileSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
