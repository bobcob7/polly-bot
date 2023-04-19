package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/bobcob7/polly-bot/internal/config"
	"github.com/bobcob7/polly-bot/internal/models"
	downloadsv1 "github.com/bobcob7/polly-bot/pkg/proto/downloads/v1"
	"github.com/bobcob7/polly-bot/pkg/proto/downloads/v1/downloadsv1connect"
	"github.com/bobcob7/transmission-rpc"
	"github.com/bufbuild/connect-go"
	"github.com/upper/db/v4"
	"go.uber.org/zap"
)

type Server struct {
	downloadsv1connect.UnimplementedDownloadServiceHandler

	logger            *zap.Logger
	config            config.GRPC
	sess              db.Session
	tx                *transmission.Client
	completedTorrents chan<- *models.Torrent
}

var _ downloadsv1connect.DownloadServiceHandler = &Server{}

func (s *Server) Run(ctx context.Context) error {
	err := make(chan error, 2)
	go func() {
		err <- s.RunGRPC(ctx)
	}()
	go func() {
		err <- s.RunScraper(ctx)
	}()
	return <-err
}

func (s *Server) RunGRPC(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	mux := http.NewServeMux()
	mux.Handle(downloadsv1connect.NewDownloadServiceHandler(s))
	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	if err := server.Serve(listener); err != nil {
		return fmt.Errorf("failed to server: %w", err)
	}
	return nil
}

func (s *Server) RunScraper(ctx context.Context) error {
	const minPeriod = time.Second * 2
	const maxPeriod = time.Minute * 5
	var currentPeriod = minPeriod
	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context error: %w", err)
			}
			return nil
		default:
		}
		err := s.scrape(ctx)
		// Adjust scrape period
		if err != nil {
			s.logger.Error("failed to scrape", zap.Error(err))
			currentPeriod *= 2
			if currentPeriod > maxPeriod {
				currentPeriod = maxPeriod
			}
		} else if currentPeriod != minPeriod {
			currentPeriod /= 2
			if currentPeriod < minPeriod {
				currentPeriod = minPeriod
			}
		}
		time.Sleep(currentPeriod)
	}
}

func (s *Server) scrape(ctx context.Context) error {
	torrents, err := s.tx.GetTorrents(ctx)
	if err != nil {
		return fmt.Errorf("failed to scrape torrents from transmission: %w", err)
	}
	s.logger.Debug("scraped torrents from transmission", zap.Int("num_torrents", len(torrents)))
	for _, torrent := range torrents {
		newTorrent := models.FromTransmission(torrent)
		completed, err := newTorrent.Set(ctx, s.sess)
		if err != nil {
			return fmt.Errorf("failed setting torrent in db: %w", err)
		}
		if completed && s.completedTorrents != nil {
			s.completedTorrents <- newTorrent
		}
	}
	return nil
}

func New(cfg *config.Config, sess db.Session, tx *transmission.Client) *Server {
	return &Server{
		logger:            zap.L(),
		tx:                tx,
		sess:              sess,
		config:            cfg.GRPC,
		completedTorrents: nil,
	}
}

func (s *Server) SubscribeCompletedTorrents(c chan<- *models.Torrent) {
	s.completedTorrents = c
}

var errUnimplemented = errors.New("method is not implemented")

func (s *Server) DeleteDownload(context.Context, *connect.Request[downloadsv1.DeleteDownloadRequest]) (*connect.Response[downloadsv1.DeleteDownloadResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errUnimplemented)
}

func (s *Server) GetDownloads(context.Context, *connect.Request[downloadsv1.GetDownloadsRequest]) (*connect.Response[downloadsv1.GetDownloadsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errUnimplemented)
}
