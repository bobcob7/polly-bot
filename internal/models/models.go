package models

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/bobcob7/transmission-rpc"
	"github.com/upper/db/v4"
)

func timestamp(i uint64) time.Time {
	t := time.Unix(int64(i), 0)
	return t
}

func FromTransmission(tx transmission.Torrent) *Torrent {
	t := Torrent{
		ID:         strconv.Itoa(tx.ID),
		Name:       tx.Name,
		CreatedAt:  timestamp(tx.AddedDate),
		Status:     tx.Status,
		MagnetLink: tx.MagnetLink,
		TotalSize:  tx.SizeWhenDone,
		Downloaded: tx.DownloadedEver,
		Uploaded:   tx.UploadedEver,
	}
	if tx.StartDate != 0 {
		date := timestamp(tx.StartDate)
		t.StartedAt = &date
	}
	if tx.DoneDate != 0 {
		date := timestamp(tx.DoneDate)
		t.CompletedAt = &date
	}
	return &t
}

const torrentTableName = "torrents"

type TorrentMetadata struct {
	FriendlyName string `db:"friendly_name"`
	Categories   []string
	Labels       map[string]string
	UpdatedAt    *time.Time `db:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"`

	rawLabels     torrentLabels
	rawCategories torrentCategories
}

func (t *TorrentMetadata) Equal(s *TorrentMetadata) bool {
	if t == nil {
		return s == nil
	}
	if s == nil {
		return false
	}
	if t.FriendlyName != s.FriendlyName {
		return false
	}
	if t.UpdatedAt == nil && s.UpdatedAt != nil ||
		s.UpdatedAt == nil && t.UpdatedAt != nil {
		return false
	}
	if t.UpdatedAt != nil && *t.UpdatedAt != *s.UpdatedAt {
		return false
	}
	if t.DeletedAt == nil && s.DeletedAt != nil ||
		s.DeletedAt == nil && t.DeletedAt != nil {
		return false
	}
	if t.DeletedAt != nil && *t.DeletedAt != *s.DeletedAt {
		return false
	}
	return true
}

type Torrent struct {
	*TorrentMetadata
	ID          string     `db:"id"`
	Name        string     `db:"name"`
	CreatedAt   time.Time  `db:"created_at"`
	StartedAt   *time.Time `db:"started_at"`
	CompletedAt *time.Time `db:"completed_at"`
	Status      int        `db:"status"`
	MagnetLink  string     `db:"magnet_link"`
	TotalSize   uint64     `db:"total_size"`
	Downloaded  uint64     `db:"downloaded"`
	Uploaded    uint64     `db:"uploaded"`
}

func (t *Torrent) String() string {
	var completedString string
	var percentCompleted float32
	if t.TotalSize != 0 {
		percentCompleted = float32(t.Downloaded) / float32(t.TotalSize)
	}
	if percentCompleted >= 1 {
		completedString = "downloaded"
	} else {
		completedString = fmt.Sprintf("%d%% downloaded", uint(percentCompleted*100))
	}
	friendlyName := t.Name
	if t.TorrentMetadata != nil && t.TorrentMetadata.FriendlyName != "" {
		friendlyName = t.TorrentMetadata.FriendlyName
	}
	return fmt.Sprintf("%s: %s", friendlyName, completedString)
}

func (t *Torrent) Equal(s Torrent) bool {
	if t.ID != s.ID {
		return false
	}
	if t.Name != s.Name {
		return false
	}
	if !t.CreatedAt.Equal(s.CreatedAt) {
		return false
	}
	if t.StartedAt == nil && s.StartedAt != nil ||
		s.StartedAt == nil && t.StartedAt != nil {
		return false
	}
	if t.StartedAt != nil && *t.StartedAt != *s.StartedAt {
		return false
	}
	if t.CompletedAt == nil && s.CompletedAt != nil ||
		s.CompletedAt == nil && t.CompletedAt != nil {
		return false
	}
	if t.CompletedAt != nil && *t.CompletedAt != *s.CompletedAt {
		return false
	}
	if t.Status != s.Status {
		return false
	}
	if t.MagnetLink != s.MagnetLink {
		return false
	}
	if t.TotalSize != s.TotalSize {
		return false
	}
	if t.Downloaded != s.Downloaded {
		return false
	}
	if t.Uploaded != s.Uploaded {
		return false
	}
	return t.TorrentMetadata.Equal(s.TorrentMetadata)
}

func (t *Torrent) setRawValues() {
	if t.TorrentMetadata == nil {
		// Skip if there is no metadata
		return
	}
	t.rawLabels = make([]torrentLabel, 0, len(t.Labels))
	for k, v := range t.Labels {
		t.rawLabels = append(t.rawLabels, torrentLabel{
			TorrentID: t.ID,
			Key:       k,
			Value:     v,
		})
	}
	t.rawCategories = make([]torrentCategory, 0, len(t.Categories))
	for _, category := range t.Categories {
		t.rawCategories = append(t.rawCategories, torrentCategory{
			TorrentID: t.ID,
			Category:  category,
		})
	}
}

func (t *Torrent) getRawValues() {
	if t.TorrentMetadata == nil {
		// Skip if there is no metadata
		return
	}
	t.Labels = make(map[string]string, len(t.rawLabels))
	for _, rawLabel := range t.rawLabels {
		if rawLabel.TorrentID == t.ID {
			t.Labels[rawLabel.Key] = rawLabel.Value
		}
	}
	t.Categories = make([]string, 0, len(t.rawCategories))
	for _, rawCategory := range t.rawCategories {
		if rawCategory.TorrentID == t.ID {
			t.Categories = append(t.Categories, rawCategory.Category)
		}
	}
}

const torrentLabelsTableName = "torrent_labels"

type torrentLabel struct {
	TorrentID string `db:"torrent_id"`
	Key       string `db:"key"`
	Value     string `db:"value"`
}

type torrentLabels []torrentLabel

func (t torrentLabels) Diff(id string, existingLabels torrentLabels) (create, delete torrentLabels) {
	current := make(map[string]string, len(t))
	for _, label := range t {
		current[label.Key] = label.Value
	}
	existing := make(map[string]string, len(existingLabels))
	for _, label := range t {
		existing[label.Key] = label.Value
	}
	create = make(torrentLabels, 0)
	delete = make(torrentLabels, 0)
	for k, v1 := range current {
		v2, ok := existing[k]
		if !ok || v1 != v2 {
			create = append(create, torrentLabel{
				TorrentID: id,
				Key:       k,
				Value:     v1,
			})
		}
	}
	for k := range existing {
		if _, ok := current[k]; !ok {
			delete = append(delete, torrentLabel{
				TorrentID: id,
				Key:       k,
			})
		}
	}
	return
}

const torrentCategoriesTableName = "torrent_categories"

type torrentCategory struct {
	TorrentID string `db:"torrent_id"`
	Category  string `db:"category"`
}

type torrentCategories []torrentCategory

func (t torrentCategories) Diff(id string, existingCategories torrentCategories) (create, delete torrentCategories) {
	current := make(map[string]struct{}, len(t))
	for _, label := range t {
		current[label.Category] = struct{}{}
	}
	existing := make(map[string]struct{}, len(existingCategories))
	for _, label := range t {
		existing[label.Category] = struct{}{}
	}
	create = make(torrentCategories, 0)
	delete = make(torrentCategories, 0)
	for k := range current {
		_, ok := existing[k]
		if !ok {
			create = append(create, torrentCategory{
				TorrentID: id,
				Category:  k,
			})
		}
	}
	for k := range existing {
		if _, ok := current[k]; !ok {
			delete = append(delete, torrentCategory{
				TorrentID: id,
				Category:  k,
			})
		}
	}
	return
}

func (t *Torrent) Set(ctx context.Context, sess db.Session) error {
	t.setRawValues()
	return sess.TxContext(ctx, func(sess db.Session) error {
		// Get current torrent record
		var existing Torrent
		existingRecord := sess.Collection(torrentTableName).Find("id", t.ID)
		count, err := existingRecord.Count()
		if err != nil {
			return fmt.Errorf("failed counting existing records %w", err)
		}
		if count > 0 {
			if err := existingRecord.One(&existing); err != nil {
				return fmt.Errorf("failed inserting record: %w", err)
			}
			if t.TorrentMetadata == nil {
				t.TorrentMetadata = existing.TorrentMetadata
			}
			// Diff to see update can be skipped
			if t.Equal(existing) {
				return nil
			}
			// Clear existing labels and categories
			if err := sess.Collection(torrentLabelsTableName).Find("torrent_id", t.ID).Delete(); err != nil {
				return fmt.Errorf("failed deleting old labels: %w", err)
			}
			if err := sess.Collection(torrentCategoriesTableName).Find("torrent_id", t.ID).Delete(); err != nil {
				return fmt.Errorf("failed deleting old categories: %w", err)
			}
			if err := existingRecord.Update(t); err != nil {
				return fmt.Errorf("failed updating record: %w", err)
			}
		} else {
			// Insert new record
			if t.TorrentMetadata == nil {
				t.TorrentMetadata = &TorrentMetadata{}
			}
			if err := sess.Collection(torrentTableName).InsertReturning(t); err != nil {
				return fmt.Errorf("failed creating new record: %w", err)
			}
		}
		// Readd labels and categories
		for i, label := range t.rawLabels {
			_, err := sess.Collection(torrentLabelsTableName).Insert(label)
			if err != nil {
				return fmt.Errorf("failed creating new label[%d]: %w", i, err)
			}
		}
		for i, category := range t.rawCategories {
			_, err := sess.Collection(torrentCategoriesTableName).Insert(category)
			if err != nil {
				return fmt.Errorf("failed creating new category[%d]: %w", i, err)
			}
		}
		return nil
	}, nil)
}

func (t *Torrent) Get(ctx context.Context, sess db.Session) error {
	if err := sess.Collection(torrentTableName).Find("id", t.ID).One(t); err != nil {
		return fmt.Errorf("failed getting record: %w", err)
	}
	if err := sess.Collection(torrentLabelsTableName).Find("torrent_id", t.ID).All(t.rawLabels); err != nil {
		return fmt.Errorf("failed getting labels: %w", err)
	}
	if err := sess.Collection(torrentCategoriesTableName).Find("torrent_id", t.ID).All(t.rawCategories); err != nil {
		return fmt.Errorf("failed getting categories: %w", err)
	}
	t.getRawValues()
	return nil
}

func GetTorrents(ctx context.Context, sess db.Session, args ...interface{}) ([]*Torrent, error) {
	output := make([]*Torrent, 0)
	if err := sess.Collection(torrentTableName).Find(args...).OrderBy("created_at").Limit(10).All(&output); err != nil {
		return nil, fmt.Errorf("failed getting records: %w", err)
	}
	for _, torrent := range output {
		if err := sess.Collection(torrentLabelsTableName).Find("torrent_id", torrent.ID).All(&torrent.rawLabels); err != nil {
			return nil, fmt.Errorf("failed getting labels: %w", err)
		}
		if err := sess.Collection(torrentCategoriesTableName).Find("torrent_id", torrent.ID).All(&torrent.rawCategories); err != nil {
			return nil, fmt.Errorf("failed getting categories: %w", err)
		}
		torrent.getRawValues()
	}
	return output, nil
}
