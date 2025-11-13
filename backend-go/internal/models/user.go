package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleAthlete  UserRole = "athlete"
	RoleFollower UserRole = "follower"
)

type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Username     string         `gorm:"uniqueIndex;not null;size:50" json:"username"`
	Email        string         `gorm:"uniqueIndex;not null;size:255" json:"email"`
	PasswordHash string         `gorm:"not null;size:255" json:"-"`
	Role         UserRole       `gorm:"not null;type:varchar(20)" json:"role"`
	DisplayName  *string        `gorm:"size:100" json:"display_name"`
	Bio          *string        `gorm:"type:text" json:"bio"`
	AvatarURL    *string        `gorm:"size:500" json:"avatar_url"`
	BannerURL    *string        `gorm:"size:500" json:"banner_url"`
	CreatedAt    time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	IsVerified   bool           `gorm:"default:false" json:"is_verified"`
	IsBanned     bool           `gorm:"default:false" json:"is_banned"`

	// Associations
	AthleteProfile *AthleteProfile `gorm:"foreignKey:UserID" json:"athlete_profile,omitempty"`
}

type AthleteProfile struct {
	UserID          uuid.UUID `gorm:"type:uuid;primary_key" json:"user_id"`
	SportCategories []string  `gorm:"type:varchar(50)[]" json:"sport_categories"`
	StreamKey       string    `gorm:"uniqueIndex;not null;size:100" json:"stream_key"`
	RTMPURL         *string   `gorm:"size:500" json:"rtmp_url"`
	FollowerCount   int       `gorm:"default:0" json:"follower_count"`
	TotalViewCount  int64     `gorm:"default:0" json:"total_view_count"`
	IsLive          bool      `gorm:"default:false" json:"is_live"`
	CreatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`

	// Association
	User User `gorm:"foreignKey:UserID" json:"-"`
}

type Follow struct {
	FollowerID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"follower_id"`
	AthleteID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"athlete_id"`
	CreatedAt            time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	NotificationsEnabled bool      `gorm:"default:true" json:"notifications_enabled"`

	// Associations
	Follower User `gorm:"foreignKey:FollowerID" json:"-"`
	Athlete  User `gorm:"foreignKey:AthleteID" json:"-"`
}

// TableName overrides
func (User) TableName() string {
	return "users"
}

func (AthleteProfile) TableName() string {
	return "athlete_profiles"
}

func (Follow) TableName() string {
	return "follows"
}

// BeforeCreate hook
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
