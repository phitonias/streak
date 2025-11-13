package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Stream struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	AthleteID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"athlete_id"`
	Title              string     `gorm:"not null;size:200" json:"title"`
	Description        *string    `gorm:"type:text" json:"description"`
	SportCategory      string     `gorm:"not null;size:50;index" json:"sport_category"`
	Tags               []string   `gorm:"type:varchar(50)[]" json:"tags"`
	ThumbnailURL       *string    `gorm:"size:500" json:"thumbnail_url"`
	IsLive             bool       `gorm:"default:false;index" json:"is_live"`
	StartedAt          *time.Time `json:"started_at"`
	EndedAt            *time.Time `json:"ended_at"`
	CurrentViewerCount int        `gorm:"default:0" json:"current_viewer_count"`
	PeakViewerCount    int        `gorm:"default:0" json:"peak_viewer_count"`
	CreatedAt          time.Time  `gorm:"default:CURRENT_TIMESTAMP;index:idx_created_at,sort:desc" json:"created_at"`

	// Associations
	Athlete User `gorm:"foreignKey:AthleteID" json:"athlete,omitempty"`
}

type StreamSession struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	StreamID       uuid.UUID  `gorm:"type:uuid;not null" json:"stream_id"`
	StartedAt      time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"started_at"`
	EndedAt        *time.Time `json:"ended_at"`
	DurationSeconds *int      `json:"duration_seconds"`
	AverageViewers int        `gorm:"default:0" json:"average_viewers"`
	PeakViewers    int        `gorm:"default:0" json:"peak_viewers"`
	TotalMessages  int        `gorm:"default:0" json:"total_messages"`

	// Association
	Stream Stream `gorm:"foreignKey:StreamID" json:"-"`
}

// TableName overrides
func (Stream) TableName() string {
	return "streams"
}

func (StreamSession) TableName() string {
	return "stream_sessions"
}

// BeforeCreate hooks
func (s *Stream) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

func (ss *StreamSession) BeforeCreate(tx *gorm.DB) error {
	if ss.ID == uuid.Nil {
		ss.ID = uuid.New()
	}
	return nil
}

// Sport categories
var SportCategories = []string{
	"basketball",
	"football",
	"soccer",
	"baseball",
	"tennis",
	"golf",
	"running",
	"cycling",
	"swimming",
	"hockey",
	"martial-arts",
	"esports",
	"fitness",
	"other",
}

// IsValidSportCategory checks if a sport category is valid
func IsValidSportCategory(category string) bool {
	for _, c := range SportCategories {
		if c == category {
			return true
		}
	}
	return false
}
