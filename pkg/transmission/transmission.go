package transmission

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
)

var defaultDownloadDir string = "/downloads/complete"

type Transmission struct {
	rootURL             string
	sessionID           string
	downloadDir         string
	cli                 *http.Client
	downloadingTorrents []int
}

func New(ctx context.Context, rootURL string) (*Transmission, error) {
	tr := &Transmission{
		rootURL:     rootURL,
		downloadDir: defaultDownloadDir,
		cli: &http.Client{
			Timeout: time.Second * 10,
		},
		downloadingTorrents: make([]int, 0),
	}
	if err := tr.getSession(ctx); err != nil {
		return nil, fmt.Errorf("failed getting session: %w", err)
	}
	return tr, nil
}

func (t *Transmission) getSession(ctx context.Context) error {
	zap.L().Debug("Getting session")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.rootURL+"/transmission/rpc", nil)
	if err != nil {
		return err
	}
	resp, err := t.cli.Do(req)
	if err != nil {
		return err
	}
	sessionID := resp.Header.Get("X-Transmission-Session-Id")
	if sessionID == "" {
		return fmt.Errorf("missing header :%#v", resp.Header)
	}
	t.sessionID = sessionID
	return nil
}

type addTransmissionRequest struct {
	Method    string                     `json:"method"`
	Arguments addTransmissionRequestArgs `json:"arguments"`
	Tag       string                     `json:"tag"`
}

type addTransmissionResponse struct {
	Result    string                                 `json:"result"`
	Arguments map[string]addTransmissionResponseArgs `json:"arguments"`
	Tag       string                                 `json:"tag"`
}

type addTransmissionRequestArgs struct {
	Paused      string `json:"paused"`
	DownloadDir string `json:"download-dir"`
	Filename    string `json:"filename"`
}

type addTransmissionResponseArgs struct {
	TorrentAdded struct {
		HashString string `json:"hashString"`
		ID         int    `json:"id"`
		Name       string `json:"name"`
	} `json:"torrent-added"`
	TorrentDuplicate struct {
		HashString string `json:"hashString"`
		ID         int    `json:"id"`
		Name       string `json:"name"`
	} `json:"torrent-duplicate"`
	Tag string `json:"tag"`
}

func (t *Transmission) AddLink(ctx context.Context, link string) error {
	if t.sessionID == "" {
		if err := t.getSession(ctx); err != nil {
			return fmt.Errorf("error getting session ID: %w", err)
		}
	}
	requestArgs := addTransmissionRequestArgs{
		Filename:    link,
		DownloadDir: t.downloadDir,
	}
	var responseBody addTransmissionResponse
	if err := t.callRPC(ctx, "torrent-add", &requestArgs, &responseBody); err != nil {
		return err
	}
	if responseBody.Result != "success" {
		return fmt.Errorf("failed to add link: %s", responseBody.Result)
	}
	return nil
}

type sessionStatsRequest struct {
	Method string `json:"method"`
	Tag    string `json:"tag"`
}

type sessionStatsResponse struct {
	Result    string       `json:"result"`
	Arguments SessionStats `json:"arguments"`
	Tag       string       `json:"tag"`
}

type byteSpeed int

func (b byteSpeed) String() string {
	fB := float64(b)
	digits := math.Log10(fB)
	if digits >= 12 {
		// Tera
		return fmt.Sprintf("%.1fTbps", fB/math.Pow10(12))
	} else if digits >= 9 {
		// Giga
		return fmt.Sprintf("%.1fGbps", fB/math.Pow10(9))
	} else if digits >= 6 {
		// Mega
		return fmt.Sprintf("%.1fMbps", fB/math.Pow10(6))
	} else if digits >= 3 {
		// Kilo
		return fmt.Sprintf("%.1fKbps", fB/math.Pow10(3))
	} else {
		// Bytes
		return strconv.Itoa(int(b))
	}
}

type byteSize int

func (b byteSize) String() string {
	fB := float64(b)
	digits := math.Log10(fB)
	if digits >= 12 {
		// Tera
		return fmt.Sprintf("%.1fT", fB/math.Pow10(12))
	} else if digits >= 9 {
		// Giga
		return fmt.Sprintf("%.1fG", fB/math.Pow10(9))
	} else if digits >= 6 {
		// Mega
		return fmt.Sprintf("%.1fM", fB/math.Pow10(6))
	} else if digits >= 3 {
		// Kilo
		return fmt.Sprintf("%.1fK", fB/math.Pow10(3))
	} else {
		// Bytes
		return strconv.Itoa(int(b))
	}
}

type SessionStats struct {
	ActiveTorrentCount int                  `json:"activeTorrentCount"`
	PausedTorrentCount int                  `json:"pausedTorrentCount"`
	TorrentCount       int                  `json:"torrentCount"`
	UploadSpeed        byteSpeed            `json:"downloadSpeed"`
	DownloadSpeed      byteSpeed            `json:"uploadSpeed"`
	CumulativeStats    DetailedSessionStats `json:"cumulative-stats"`
	CurrentStats       DetailedSessionStats `json:"current-stats"`
}

func (s SessionStats) String() string {
	return fmt.Sprintf(`Active Torrent Count: %d
Paused Torrent Count: %d
Total Torrent Count:  %d
Upload Speed:         %v
Download Speed:       %v
Cumulative Stats:
        %v
Current Stats:
        %v`,
		s.ActiveTorrentCount,
		s.PausedTorrentCount,
		s.TorrentCount,
		s.UploadSpeed,
		s.DownloadSpeed,
		s.CumulativeStats,
		s.CurrentStats,
	)
}

type DetailedSessionStats struct {
	UploadedBytes   byteSize `json:"uploadedBytes"`
	DownloadedBytes byteSize `json:"downloadedBytes"`
	FilesAdded      int      `json:"filesAdded"`
	SessionCount    int      `json:"sessionCount"`
	SecondsActive   int      `json:"secondsActive"`
}

func (d DetailedSessionStats) String() string {
	return fmt.Sprintf(`Uploaded Bytes: %v
	Downloaded Bytes: %v
	Files Added:      %d
	Session Count:    %d
	Seconds Active:   %d`,
		d.UploadedBytes,
		d.DownloadedBytes,
		d.FilesAdded,
		d.SessionCount,
		d.SecondsActive,
	)
}

type genericRequest struct {
	Method    string      `json:"method"`
	Arguments interface{} `json:"arguments"`
	Tag       string      `json:"tag"`
}

func (t *Transmission) callRPC(ctx context.Context, requestMethod string, requestArguments, response interface{}) error {
	var resp *http.Response
	if t.sessionID == "" {
		if err := t.getSession(ctx); err != nil {
			return fmt.Errorf("error getting session ID: %w", err)
		}
	}
	buffer := new(bytes.Buffer)
	json.NewEncoder(buffer).Encode(genericRequest{
		Method:    requestMethod,
		Arguments: requestArguments,
	})
	logger := zap.L().With(zap.String("method", requestMethod))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.rootURL+"/transmission/rpc", buffer)
	if err != nil {
		return err
	}
	for {
		logger.Debug("Calling RPC")
		req.Header.Add("X-Transmission-Session-Id", t.sessionID)
		resp, err = t.cli.Do(req)
		if err != nil {
			logger.Error("Failed to call RPC", zap.Error(err))
			return err
		}
		logger = logger.With(zap.Int("statusCode", resp.StatusCode))
		logger.Debug("Finished RPC")
		if resp.StatusCode != 409 {
			break
		}
		t.getSession(ctx)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return err
	}
	return nil
}

func (t *Transmission) GetStats(ctx context.Context) (*SessionStats, error) {
	var response sessionStatsResponse
	if err := t.callRPC(ctx, "session-stats", nil, &response); err != nil {
		return nil, err
	}
	if response.Result != "success" {
		return nil, fmt.Errorf("failed to get session stats: %s", response.Result)
	}
	return &response.Arguments, nil
}

type listTorrentsRequest struct {
	Method    string                  `json:"method"`
	Arguments listTorrentsRequestArgs `json:"arguments"`
	Tag       string                  `json:"tag"`
}

type listTorrentsResponse struct {
	Result    string                   `json:"result"`
	Arguments listTorrentsResponseArgs `json:"arguments"`
	Tag       string                   `json:"tag"`
}

type listTorrentsResponseArgs struct {
	Torrents []TorrentWithName `json:"torrents"`
}

type TorrentWithName struct {
	ID            int           `json:"id"`
	Name          string        `json:"name"`
	PercentDone   float64       `json:"percentDone"`
	TotalSize     byteSize      `json:"totalSize"`
	Status        int           `json:"status"`
	LeftUntilDone time.Duration `json:"leftUntilDone"`
	RateDownload  byteSpeed     `json:"rateDownload"`
	IsStalled     bool          `json:"isStalled"`
}

func (t TorrentWithName) String() string {
	return fmt.Sprintf(`%s: %v(%f%%)`, t.Name, t.TotalSize, t.PercentDone*100)
}

type listTorrentsRequestArgs struct {
	IDs    []int    `json:"ids,omitempty"`
	Fields []string `json:"fields"`
}

func (t *Transmission) getDownloadingTorrents(ctx context.Context) (map[int]TorrentWithName, error) {
	requestArgs := listTorrentsRequestArgs{
		Fields: []string{"id", "name", "percentDone", "totalSize", "status", "leftUntilDone", "rateDownload"},
	}
	response := listTorrentsResponse{}
	if err := t.callRPC(ctx, "torrent-get", &requestArgs, &response); err != nil {
		return nil, err
	}
	if response.Result != "success" {
		return nil, fmt.Errorf("failed to get list torrents: %s", response.Result)
	}
	ids := make(map[int]TorrentWithName, len(response.Arguments.Torrents))
	for _, torrent := range response.Arguments.Torrents {
		if torrent.Status == 0 {
			continue
		}
		if torrent.PercentDone < 1 {
			if torrent.RateDownload <= 0 {
				torrent.LeftUntilDone = time.Duration(-1)
			} else {
				torrent.LeftUntilDone *= 100
			}
			ids[torrent.ID] = torrent
		}
	}
	return ids, nil
}

func (t *Transmission) getCompletedTorrents(ctx context.Context, ids []int) ([]TorrentWithName, error) {
	requestArgs := listTorrentsRequestArgs{
		IDs:    ids,
		Fields: []string{"id", "name", "percentDone", "totalSize", "status", "leftUntilDone", "rateDownload"},
	}
	response := listTorrentsResponse{}
	if err := t.callRPC(ctx, "torrent-get", &requestArgs, &response); err != nil {
		return nil, err
	}
	if response.Result != "success" {
		return nil, fmt.Errorf("failed to get list torrents: %s", response.Result)
	}
	output := make([]TorrentWithName, 0)
	for _, torrent := range response.Arguments.Torrents {
		if torrent.PercentDone == 1 {
			output = append(output, torrent)
		}
	}
	return output, nil
}

func (t *Transmission) pollDownloadingTorrents(ctx context.Context) ([]TorrentWithName, error) {
	requestArgs := listTorrentsRequestArgs{
		IDs:    t.downloadingTorrents,
		Fields: []string{"id", "name", "percentDone", "totalSize", "status", "leftUntilDone", "rateDownload"},
	}
	response := listTorrentsResponse{}
	if err := t.callRPC(ctx, "torrent-get", &requestArgs, &response); err != nil {
		return nil, err
	}
	if response.Result != "success" {
		return nil, fmt.Errorf("failed to get list torrents: %s", response.Result)
	}
	output := make([]TorrentWithName, 0, len(response.Arguments.Torrents))
	for _, torrent := range response.Arguments.Torrents {
		if torrent.Status == 0 {
			continue
		}
		if torrent.PercentDone < 1 {
			if torrent.RateDownload <= 0 {
				continue
			}
			torrent.LeftUntilDone *= 100
			output = append(output, torrent)
		}
	}
	return output, nil
}

func (t *Transmission) Start(ctx context.Context, events chan<- Event, downloadingPollPeriod, completePollPeriod time.Duration) {
	completeTicker := time.NewTicker(completePollPeriod)
	defer completeTicker.Stop()
	downloadingTicker := time.NewTicker(downloadingPollPeriod)
	defer downloadingTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-completeTicker.C:
			shortCtx, done := context.WithTimeout(ctx, downloadingPollPeriod)
			defer done()
			t.getDownloadingTorrents(shortCtx)
		case <-downloadingTicker.C:
			t.pollDownloadingTorrents(ctx)
		}
	}
}

type Action string

const (
	ActionAdd       Action = "ADD"
	ActionUpdate    Action = "UPDATE"
	ActionCompleted Action = "COMPLETED"
	ActionDeleted   Action = "DELETED"
)

type Event struct {
	Action  Action
	Torrent TorrentWithName
}
