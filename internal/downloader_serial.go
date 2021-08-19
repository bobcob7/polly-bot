package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"go.uber.org/zap"
)

type Downloader interface {
	Wait(context.Context, <-chan LinkResult)
}

type SerialDownloader struct {
	client  http.Client
	baseDir string
}

func NewSingleDownloader(path string) *SerialDownloader {
	return &SerialDownloader{
		client:  *http.DefaultClient,
		baseDir: path,
	}
}

func (d SerialDownloader) Wait(ctx context.Context, links <-chan LinkResult) {
	for {
		select {
		case <-ctx.Done():
		case request, ok := <-links:
			if !ok {
				return
			}
			d.download(request)
		}
	}
}

func (d SerialDownloader) download(request LinkResult) {
	logger.Info("Download link",
		zap.String("filename", request.name),
		zap.String("link", request.link),
	)
	resp, err := d.client.Get(request.link)
	if err != nil {
		logger.Error("Error downloading",
			zap.String("filename", request.name),
			zap.String("link", request.link),
			zap.Error(err),
		)
	}
	name := fmt.Sprint(d.baseDir, string(os.PathSeparator), request.name)
	f, err := os.Create(name)
	if err != nil {
		logger.Error("Error creating file",
			zap.String("filename", request.name),
			zap.Error(err),
		)
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		logger.Error("Error copying file",
			zap.Error(err),
		)
	}
}
