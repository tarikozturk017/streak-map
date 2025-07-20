package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TrackingFrequency string

const (
	TrackingFrequencyDaily   TrackingFrequency = "daily"
	TrackingFrequencyWeekly  TrackingFrequency = "weekly"
	TrackingFrequencyMonthly TrackingFrequency = "monthly"
)

type GoalType string

const (
	GoalTypeTime     GoalType = "time"     // Hours, minutes (e.g., "2h 30m study")
	GoalTypeQuantity GoalType = "quantity" // Pages, exercises, etc. (e.g., "10 pages read")
	GoalTypeBoolean  GoalType = "boolean"  // Yes/No completion (e.g., "did workout")
	GoalTypeDistance GoalType = "distance" // Kilometers, miles (e.g., "5km run")
)

type Goal struct {
	ID                uuid.UUID         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID            uuid.UUID         `json:"user_id" gorm:"type:uuid;not null;index"`
	Title             string            `json:"title" gorm:"not null;size:255"`
	Description       string            `json:"description" gorm:"type:text"`
	Type              GoalType          `json:"type" gorm:"not null;default:'quantity'"`
	ColorCode         string            `json:"color_code" gorm:"not null;size:7"` // Hex color like #FF0000
	TrackingFrequency TrackingFrequency `json:"tracking_frequency" gorm:"not null;default:'daily'"`
	Target            float64           `json:"target" gorm:"not null"`             // Target value in base units
	Unit              string            `json:"unit" gorm:"size:50"`                // Unit label (pages, km, hours, etc.)
	IsActive          bool              `json:"is_active" gorm:"default:true"`
	GroupID           *uuid.UUID        `json:"group_id,omitempty" gorm:"type:uuid;index"` // For future grouping
	SortOrder         int               `json:"sort_order" gorm:"default:0"`               // For ordering goals
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`

	// Relationships
	User       User        `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Group      *GoalGroup  `json:"group,omitempty" gorm:"foreignKey:GroupID"`
	Progresses []Progress  `json:"progresses,omitempty" gorm:"foreignKey:GoalID"`
}

type GoalGroup struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	Name        string    `json:"name" gorm:"not null;size:255"`
	Description string    `json:"description" gorm:"type:text"`
	ColorCode   string    `json:"color_code" gorm:"size:7"` // Optional group color
	SortOrder   int       `json:"sort_order" gorm:"default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relationships
	User  User   `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Goals []Goal `json:"goals,omitempty" gorm:"foreignKey:GroupID"`
}

type CreateGoalRequest struct {
	Title             string            `json:"title" validate:"required,max=255"`
	Description       string            `json:"description" validate:"max=1000"`
	Type              GoalType          `json:"type" validate:"required,oneof=time quantity boolean distance"`
	ColorCode         string            `json:"color_code" validate:"required,hexcolor"`
	TrackingFrequency TrackingFrequency `json:"tracking_frequency" validate:"required,oneof=daily weekly monthly"`
	Target            float64           `json:"target" validate:"required,min=0"`
	Unit              string            `json:"unit" validate:"required,max=50"`
	GroupID           *uuid.UUID        `json:"group_id,omitempty"`
	SortOrder         int               `json:"sort_order"`
}

type UpdateGoalRequest struct {
	Title             *string            `json:"title,omitempty" validate:"omitempty,max=255"`
	Description       *string            `json:"description,omitempty" validate:"omitempty,max=1000"`
	Type              *GoalType          `json:"type,omitempty" validate:"omitempty,oneof=time quantity boolean distance"`
	ColorCode         *string            `json:"color_code,omitempty" validate:"omitempty,hexcolor"`
	TrackingFrequency *TrackingFrequency `json:"tracking_frequency,omitempty" validate:"omitempty,oneof=daily weekly monthly"`
	Target            *float64           `json:"target,omitempty" validate:"omitempty,min=0"`
	Unit              *string            `json:"unit,omitempty" validate:"omitempty,max=50"`
	IsActive          *bool              `json:"is_active,omitempty"`
	GroupID           *uuid.UUID         `json:"group_id,omitempty"`
	SortOrder         *int               `json:"sort_order,omitempty"`
}

type CreateGoalGroupRequest struct {
	Name        string `json:"name" validate:"required,max=255"`
	Description string `json:"description" validate:"max=1000"`
	ColorCode   string `json:"color_code" validate:"omitempty,hexcolor"`
	SortOrder   int    `json:"sort_order"`
}

type UpdateGoalGroupRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000"`
	ColorCode   *string `json:"color_code,omitempty" validate:"omitempty,hexcolor"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

func (tf TrackingFrequency) IsValid() bool {
	switch tf {
	case TrackingFrequencyDaily, TrackingFrequencyWeekly, TrackingFrequencyMonthly:
		return true
	}
	return false
}

func (gt GoalType) IsValid() bool {
	switch gt {
	case GoalTypeTime, GoalTypeQuantity, GoalTypeBoolean, GoalTypeDistance:
		return true
	}
	return false
}

func (g *Goal) ConvertInputToBaseUnit(input float64) float64 {
	switch g.Type {
	case GoalTypeTime:
		// For time goals, input should be in minutes, target stored in minutes
		return input
	case GoalTypeQuantity, GoalTypeDistance:
		// For quantity/distance, direct conversion
		return input
	case GoalTypeBoolean:
		// For boolean goals, 1 = completed, 0 = not completed
		if input > 0 {
			return 1
		}
		return 0
	default:
		return input
	}
}

func (g *Goal) FormatTargetDisplay() string {
	switch g.Type {
	case GoalTypeTime:
		hours := int(g.Target) / 60
		minutes := int(g.Target) % 60
		if hours > 0 && minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		} else if hours > 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dm", minutes)
	case GoalTypeBoolean:
		return "Complete"
	default:
		return fmt.Sprintf("%.1f %s", g.Target, g.Unit)
	}
}

func (g *Goal) FormatValueDisplay(value float64) string {
	switch g.Type {
	case GoalTypeTime:
		hours := int(value) / 60
		minutes := int(value) % 60
		if hours > 0 && minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		} else if hours > 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dm", minutes)
	case GoalTypeBoolean:
		if value > 0 {
			return "Completed"
		}
		return "Not completed"
	default:
		return fmt.Sprintf("%.1f %s", value, g.Unit)
	}
}