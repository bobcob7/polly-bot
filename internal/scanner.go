package internal

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"
)

type history interface {
	Add(string) bool
	Cleanup()
}

type LinkResult struct {
	link string
	name string
}

func NewScanner(hist history, downloader Downloader) *Scanner {
	return &Scanner{
		parser:     gofeed.NewParser(),
		hist:       hist,
		downloader: downloader,
	}
}

type Scanner struct {
	parser     *gofeed.Parser
	hist       history
	downloader Downloader
}

func (s *Scanner) Run(ctx context.Context, db *sql.DB, period time.Duration) {
	ticker := time.NewTicker(period)
	logger.Info("Starting subject processor",
		zap.Duration("period", period),
	)
	results := make(chan LinkResult, 10)
	go s.downloader.Wait(ctx, results)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			subjects, err := GetSubjects(db)
			if err != nil {
				logger.Error("Error while getting subjects",
					zap.Error(err),
				)
				continue
			}
			s.ProcessSubjects(ctx, subjects, results)
		}
	}

}

func (s *Scanner) ProcessSubjects(ctx context.Context, subjects []*Subject, results chan<- LinkResult) error {
	logger.Info("Processing subjects",
		zap.Int("number", len(subjects)),
	)
	for _, subject := range subjects {
		feed, err := s.parser.ParseURLWithContext(subject.GetURL(), ctx)
		if err != nil {
			return fmt.Errorf("failed to parse feed: %w", err)
		}
		for _, item := range feed.Items {
			title := item.Title
			if subject.MatchString(title) && s.hist.Add(title) {
				results <- LinkResult{
					link: item.Link,
					name: item.Title,
				}
			}
		}
	}
	s.hist.Cleanup()
	return nil
}
