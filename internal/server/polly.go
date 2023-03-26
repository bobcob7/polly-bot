package server

import (
	"context"
	"net"
	"time"

	"github.com/bobcob7/polly-bot/internal/config"
	"github.com/bobcob7/polly-bot/internal/models"
	downloadsv1 "github.com/bobcob7/polly-bot/pkg/proto/downloads/v1"
	"github.com/bobcob7/transmission-rpc"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"github.com/upper/db/v4"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	downloadsv1.UnimplementedDownloadServiceServer

	logger *zap.Logger
	config config.ConfigGRPC
	sess   db.Session
	tx     *transmission.Client
}

var _ downloadsv1.DownloadServiceServer = &Server{}

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
	server := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.ChainUnaryServer(
			grpc_recovery.UnaryServerInterceptor(),
			grpc_opentracing.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(s.logger),
		)),
	)
	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return err
	}
	downloadsv1.RegisterDownloadServiceServer(server, s)
	return server.Serve(listener)
}

func (s *Server) RunScraper(ctx context.Context) error {
	const minPeriod = time.Second * 2
	const maxPeriod = time.Minute * 5
	var currentPeriod = minPeriod
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
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
		return err
	}
	s.logger.Debug("scraped torrents from transmission", zap.Int("num_torrents", len(torrents)))
	for _, torrent := range torrents {
		newTorrent := models.FromTransmission(torrent)
		if err := newTorrent.Set(ctx, s.sess); err != nil {
			return err
		}
	}
	return nil
}

func New(cfg *config.Config, sess db.Session, tx *transmission.Client) *Server {
	return &Server{
		logger: zap.L(),
		tx:     tx,
		sess:   sess,
		config: cfg.GRPC,
	}
}

func (s *Server) CreateDownload(ctx context.Context, request *downloadsv1.CreateDownloadRequest) (*downloadsv1.CreateDownloadResponse, error) {
	return nil, nil
}

func (s *Server) DeleteDownload(ctx context.Context, request *downloadsv1.DeleteDownloadRequest) (*downloadsv1.DeleteDownloadResponse, error) {
	return nil, nil
}

func (s *Server) GetDownloads(ctx context.Context, request *downloadsv1.GetDownloadsRequest) (*downloadsv1.GetDownloadsResponse, error) {
	return nil, nil
}
