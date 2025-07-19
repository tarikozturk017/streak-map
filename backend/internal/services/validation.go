package services

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"github.com/tarikozturk017/streak-map/backend/internal/models"
)

type ValidationService struct {
	db *gorm.DB
}

func NewValidationService(db *gorm.DB) *ValidationService {
	return &ValidationService{db: db}
}

func (v *ValidationService) ValidateGoalOwnership(goalID, userID uuid.UUID) error {
	var goal models.Goal
	if err := v.db.Where("id = ? AND user_id = ?", goalID, userID).First(&goal).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("goal not found or access denied")
		}
		return err
	}
	return nil
}

func (v *ValidationService) ValidateProgressInput(goal models.Goal, value float64) error {
	switch goal.Type {
	case models.GoalTypeTime:
		if value < 0 || value > 24*60 { // Max 24 hours in minutes
			return errors.New("time value must be between 0 and 1440 minutes (24 hours)")
		}
	case models.GoalTypeQuantity:
		if value < 0 {
			return errors.New("quantity value must be non-negative")
		}
		if value > goal.Target*10 { // Reasonable upper bound
			return errors.New("quantity value seems unreasonably high")
		}
	case models.GoalTypeDistance:
		if value < 0 {
			return errors.New("distance value must be non-negative")
		}
		if value > goal.Target*10 { // Reasonable upper bound
			return errors.New("distance value seems unreasonably high")
		}
	case models.GoalTypeBoolean:
		if value != 0 && value != 1 {
			return errors.New("boolean value must be 0 or 1")
		}
	}
	return nil
}

func (v *ValidationService) ValidateProgressDate(goalID uuid.UUID, userID uuid.UUID, date time.Time, excludeProgressID *uuid.UUID) error {
	query := v.db.Where("goal_id = ? AND user_id = ? AND DATE(tracked_date) = DATE(?)", goalID, userID, date)
	
	if excludeProgressID != nil {
		query = query.Where("id != ?", *excludeProgressID)
	}
	
	var existingProgress models.Progress
	if err := query.First(&existingProgress).Error; err == nil {
		return errors.New("progress entry already exists for this date")
	} else if err != gorm.ErrRecordNotFound {
		return err
	}
	
	return nil
}

func (v *ValidationService) ValidateGoalType(goalType models.GoalType, target float64, unit string) error {
	if !goalType.IsValid() {
		return errors.New("invalid goal type")
	}

	switch goalType {
	case models.GoalTypeTime:
		if target <= 0 || target > 24*60 {
			return errors.New("time target must be between 1 and 1440 minutes")
		}
		if unit != "minutes" && unit != "hours" {
			return errors.New("time goals must use 'minutes' or 'hours' as unit")
		}
	case models.GoalTypeQuantity:
		if target <= 0 {
			return errors.New("quantity target must be positive")
		}
		if unit == "" {
			return errors.New("quantity goals must specify a unit")
		}
	case models.GoalTypeDistance:
		if target <= 0 {
			return errors.New("distance target must be positive")
		}
		if unit != "km" && unit != "miles" && unit != "meters" {
			return errors.New("distance goals must use 'km', 'miles', or 'meters' as unit")
		}
	case models.GoalTypeBoolean:
		if target != 1 {
			return errors.New("boolean goals must have target of 1")
		}
	}

	return nil
}

func (v *ValidationService) ValidateTrackingFrequency(frequency models.TrackingFrequency, trackedDate time.Time) error {
	if !frequency.IsValid() {
		return errors.New("invalid tracking frequency")
	}

	now := time.Now()
	switch frequency {
	case models.TrackingFrequencyDaily:
		// Allow dates within the last year and up to tomorrow
		if trackedDate.Before(now.AddDate(-1, 0, 0)) || trackedDate.After(now.AddDate(0, 0, 1)) {
			return errors.New("daily tracking date must be within the last year and not more than 1 day in the future")
		}
	case models.TrackingFrequencyWeekly:
		// For weekly, check if it's the start of a week
		weekday := trackedDate.Weekday()
		if weekday != time.Monday {
			return errors.New("weekly tracking must start on Monday")
		}
	case models.TrackingFrequencyMonthly:
		// For monthly, check if it's the first day of the month
		if trackedDate.Day() != 1 {
			return errors.New("monthly tracking must be on the first day of the month")
		}
	}

	return nil
}

func (v *ValidationService) GetProgressSummary(goalID uuid.UUID, userID uuid.UUID, startDate, endDate time.Time) (*models.ProgressSummary, error) {
	var goal models.Goal
	if err := v.db.Where("id = ? AND user_id = ?", goalID, userID).First(&goal).Error; err != nil {
		return nil, err
	}

	var progressEntries []models.Progress
	if err := v.db.Where("goal_id = ? AND user_id = ? AND tracked_date BETWEEN ? AND ?", 
		goalID, userID, startDate, endDate).
		Order("tracked_date ASC").
		Find(&progressEntries).Error; err != nil {
		return nil, err
	}

	if len(progressEntries) == 0 {
		return &models.ProgressSummary{
			GoalID:    goalID,
			GoalTitle: goal.Title,
			GoalType:  goal.Type,
		}, nil
	}

	// Calculate summary statistics
	var totalCompletion float64
	var bestCompletion float64
	currentStreak := 0
	longestStreak := 0
	tempStreak := 0

	for i, entry := range progressEntries {
		totalCompletion += entry.CompletionRate
		if entry.CompletionRate > bestCompletion {
			bestCompletion = entry.CompletionRate
		}

		// Calculate streaks (consecutive days with progress > 0)
		if entry.CompletionRate > 0 {
			tempStreak++
			if tempStreak > longestStreak {
				longestStreak = tempStreak
			}
			if i == len(progressEntries)-1 { // Last entry
				currentStreak = tempStreak
			}
		} else {
			if i == len(progressEntries)-1 { // Last entry had no progress
				currentStreak = 0
			}
			tempStreak = 0
		}
	}

	avgCompletion := totalCompletion / float64(len(progressEntries))

	return &models.ProgressSummary{
		GoalID:            goalID,
		GoalTitle:         goal.Title,
		GoalType:          goal.Type,
		TotalEntries:      len(progressEntries),
		AverageCompletion: avgCompletion,
		BestCompletion:    bestCompletion,
		CurrentStreak:     currentStreak,
		LongestStreak:     longestStreak,
		LastTrackedDate:   progressEntries[len(progressEntries)-1].TrackedDate,
	}, nil
}