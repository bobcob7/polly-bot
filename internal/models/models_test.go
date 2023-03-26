package models

import (
	"testing"
	"time"
)

func TestTorrent_String(t *testing.T) {
	type fields struct {
		TorrentMetadata *TorrentMetadata
		ID              string
		Name            string
		CreatedAt       time.Time
		StartedAt       *time.Time
		CompletedAt     *time.Time
		Status          int
		MagnetLink      string
		TotalSize       uint64
		Downloaded      uint64
		Uploaded        uint64
	}
	tests := map[string]struct {
		fields fields
		want   string
	}{
		"No friendly name": {
			fields: fields{
				ID:         "0",
				Name:       "Something.Really.Bad",
				TotalSize:  10,
				Downloaded: 0,
			},
			want: `Something.Really.Bad: 0% downloaded`,
		},
		"With friendly name": {
			fields: fields{
				ID:   "0",
				Name: "Something.Really.Bad",
				TorrentMetadata: &TorrentMetadata{
					FriendlyName: "Something Good",
				},
				TotalSize:  10,
				Downloaded: 0,
			},
			want: `Something Good: 0% downloaded`,
		},
		"Fifty percent done": {
			fields: fields{
				ID:         "0",
				Name:       "Something.Really.Bad",
				TotalSize:  10,
				Downloaded: 5,
			},
			want: `Something.Really.Bad: 50% downloaded`,
		},
		"100% done": {
			fields: fields{
				ID:         "0",
				Name:       "Something.Really.Bad",
				TotalSize:  10,
				Downloaded: 10,
			},
			want: `Something.Really.Bad: downloaded`,
		},
		"120% done": {
			fields: fields{
				ID:         "0",
				Name:       "Something.Really.Bad",
				TotalSize:  10,
				Downloaded: 12,
			},
			want: `Something.Really.Bad: downloaded`,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tr := &Torrent{
				TorrentMetadata: tt.fields.TorrentMetadata,
				ID:              tt.fields.ID,
				Name:            tt.fields.Name,
				CreatedAt:       tt.fields.CreatedAt,
				StartedAt:       tt.fields.StartedAt,
				CompletedAt:     tt.fields.CompletedAt,
				Status:          tt.fields.Status,
				MagnetLink:      tt.fields.MagnetLink,
				TotalSize:       tt.fields.TotalSize,
				Downloaded:      tt.fields.Downloaded,
				Uploaded:        tt.fields.Uploaded,
			}
			if got := tr.String(); got != tt.want {
				t.Errorf("Torrent.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
