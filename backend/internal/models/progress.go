package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Progress struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	GoalID           uuid.UUID `json:"goal_id" gorm:"type:uuid;not null;index"`
	UserID           uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"` // For faster queries
	Value            float64   `json:"value" gorm:"not null"`                   // Actual progress value in base units
	CompletionRate   float64   `json:"completion_rate" gorm:"not null"`         // Calculated percentage (value/target * 100)
	Notes            string    `json:"notes" gorm:"type:text"`                  // Optional notes for this entry
	TrackedDate      time.Time `json:"tracked_date" gorm:"not null;index"`      // The date this progress represents
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// Relationships
	Goal Goal `json:"goal,omitempty" gorm:"foreignKey:GoalID;constraint:OnDelete:CASCADE"`
	User User `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

type CreateProgressRequest struct {
	GoalID      uuid.UUID `json:"goal_id" validate:"required"`
	Value       float64   `json:"value" validate:"required,min=0"`
	Notes       string    `json:"notes" validate:"max=1000"`
	TrackedDate time.Time `json:"tracked_date" validate:"required"`
}

type CreateProgressTimeRequest struct {
	GoalID      uuid.UUID `json:"goal_id" validate:"required"`
	Hours       int       `json:"hours" validate:"min=0,max=23"`
	Minutes     int       `json:"minutes" validate:"min=0,max=59"`
	Notes       string    `json:"notes" validate:"max=1000"`
	TrackedDate time.Time `json:"tracked_date" validate:"required"`
}

type UpdateProgressRequest struct {
	Value       *float64   `json:"value,omitempty" validate:"omitempty,min=0"`
	Notes       *string    `json:"notes,omitempty" validate:"omitempty,max=1000"`
	TrackedDate *time.Time `json:"tracked_date,omitempty"`
}

type ProgressFilter struct {
	GoalID    *uuid.UUID `json:"goal_id,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	Page      int        `json:"page" validate:"min=1"`
	Limit     int        `json:"limit" validate:"min=1,max=100"`
}

type ProgressSummary struct {
	GoalID             uuid.UUID `json:"goal_id"`
	GoalTitle          string    `json:"goal_title"`
	GoalType           GoalType  `json:"goal_type"`
	TotalEntries       int       `json:"total_entries"`
	AverageCompletion  float64   `json:"average_completion"`
	BestCompletion     float64   `json:"best_completion"`
	CurrentStreak      int       `json:"current_streak"`
	LongestStreak      int       `json:"longest_streak"`
	LastTrackedDate    time.Time `json:"last_tracked_date"`
}

type HeatmapData struct {
	Date           time.Time `json:"date"`
	CompletionRate float64   `json:"completion_rate"`
	Value          float64   `json:"value"`
	GoalTitle      string    `json:"goal_title"`
	GoalType       GoalType  `json:"goal_type"`
	ColorCode      string    `json:"color_code"`
	Notes          string    `json:"notes,omitempty"`
	FormattedValue string    `json:"formatted_value"`
}

type ProgressResponse struct {
	Progress Progress `json:"progress"`
	Goal     Goal     `json:"goal"`
}

func (p *Progress) CalculateCompletionRate(target float64) {
	if target <= 0 {
		p.CompletionRate = 0
		return
	}
	p.CompletionRate = (p.Value / target) * 100
	if p.CompletionRate > 100 {
		p.CompletionRate = 100
	}
}

func (p *Progress) GetIntensityLevel() int {
	switch {
	case p.CompletionRate >= 90:
		return 4 // Highest intensity
	case p.CompletionRate >= 70:
		return 3
	case p.CompletionRate >= 40:
		return 2
	case p.CompletionRate >= 10:
		return 1
	default:
		return 0 // No progress
	}
}

func (req *CreateProgressTimeRequest) ConvertToMinutes() float64 {
	return float64(req.Hours*60 + req.Minutes)
}

func (p *Progress) FormatValueByGoalType(goalType GoalType, unit string) string {
	switch goalType {
	case GoalTypeTime:
		hours := int(p.Value) / 60
		minutes := int(p.Value) % 60
		if hours > 0 && minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		} else if hours > 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dm", minutes)
	case GoalTypeBoolean:
		if p.Value > 0 {
			return "Completed"
		}
		return "Not completed"
	default:
		return fmt.Sprintf("%.1f %s", p.Value, unit)
	}
}

func (p *Progress) IsConsideredComplete() bool {
	return p.CompletionRate >= 100
}

func (p *Progress) GetCompletionLevel() string {
	switch {
	case p.CompletionRate >= 100:
		return "complete"
	case p.CompletionRate >= 75:
		return "high"
	case p.CompletionRate >= 50:
		return "medium"
	case p.CompletionRate >= 25:
		return "low"
	default:
		return "minimal"
	}
}