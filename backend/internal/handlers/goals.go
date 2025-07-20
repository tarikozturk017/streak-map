package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"github.com/tarikozturk017/streak-map/backend/internal/models"
)

type GoalHandler struct {
	db *gorm.DB
}

func NewGoalHandler(db *gorm.DB) *GoalHandler {
	return &GoalHandler{db: db}
}

func (h *GoalHandler) CreateGoal(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)

	var req models.CreateGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !models.GoalType(req.Type).IsValid() {
		http.Error(w, "Invalid goal type", http.StatusBadRequest)
		return
	}

	if !models.TrackingFrequency(req.TrackingFrequency).IsValid() {
		http.Error(w, "Invalid tracking frequency", http.StatusBadRequest)
		return
	}

	goal := models.Goal{
		ID:                uuid.New(),
		UserID:            userID,
		Title:             req.Title,
		Description:       req.Description,
		Type:              req.Type,
		ColorCode:         req.ColorCode,
		TrackingFrequency: req.TrackingFrequency,
		Target:            req.Target,
		Unit:              req.Unit,
		IsActive:          true,
		GroupID:           req.GroupID,
		SortOrder:         req.SortOrder,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if req.GroupID != nil {
		var group models.GoalGroup
		if err := h.db.Where("id = ? AND user_id = ?", *req.GroupID, userID).First(&group).Error; err != nil {
			http.Error(w, "Goal group not found", http.StatusNotFound)
			return
		}
	}

	if err := h.db.Create(&goal).Error; err != nil {
		http.Error(w, "Failed to create goal", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(goal)
}

func (h *GoalHandler) GetGoals(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)

	var goals []models.Goal
	query := h.db.Where("user_id = ?", userID).Order("sort_order ASC, created_at ASC")

	if groupID := r.URL.Query().Get("group_id"); groupID != "" {
		if parsedGroupID, err := uuid.Parse(groupID); err == nil {
			query = query.Where("group_id = ?", parsedGroupID)
		}
	}

	if active := r.URL.Query().Get("active"); active != "" {
		if isActive, err := strconv.ParseBool(active); err == nil {
			query = query.Where("is_active = ?", isActive)
		}
	}

	if err := query.Preload("Group").Find(&goals).Error; err != nil {
		http.Error(w, "Failed to fetch goals", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(goals)
}

func (h *GoalHandler) GetGoal(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)
	goalID := r.PathValue("id")

	parsedGoalID, err := uuid.Parse(goalID)
	if err != nil {
		http.Error(w, "Invalid goal ID", http.StatusBadRequest)
		return
	}

	var goal models.Goal
	if err := h.db.Where("id = ? AND user_id = ?", parsedGoalID, userID).
		Preload("Group").
		First(&goal).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Goal not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch goal", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(goal)
}

func (h *GoalHandler) UpdateGoal(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)
	goalID := r.PathValue("id")

	parsedGoalID, err := uuid.Parse(goalID)
	if err != nil {
		http.Error(w, "Invalid goal ID", http.StatusBadRequest)
		return
	}

	var req models.UpdateGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var goal models.Goal
	if err := h.db.Where("id = ? AND user_id = ?", parsedGoalID, userID).First(&goal).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Goal not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch goal", http.StatusInternalServerError)
		return
	}

	if req.Title != nil {
		goal.Title = *req.Title
	}
	if req.Description != nil {
		goal.Description = *req.Description
	}
	if req.Type != nil {
		if !req.Type.IsValid() {
			http.Error(w, "Invalid goal type", http.StatusBadRequest)
			return
		}
		goal.Type = *req.Type
	}
	if req.ColorCode != nil {
		goal.ColorCode = *req.ColorCode
	}
	if req.TrackingFrequency != nil {
		if !req.TrackingFrequency.IsValid() {
			http.Error(w, "Invalid tracking frequency", http.StatusBadRequest)
			return
		}
		goal.TrackingFrequency = *req.TrackingFrequency
	}
	if req.Target != nil {
		goal.Target = *req.Target
	}
	if req.Unit != nil {
		goal.Unit = *req.Unit
	}
	if req.IsActive != nil {
		goal.IsActive = *req.IsActive
	}
	if req.GroupID != nil {
		var group models.GoalGroup
		if err := h.db.Where("id = ? AND user_id = ?", *req.GroupID, userID).First(&group).Error; err != nil {
			http.Error(w, "Goal group not found", http.StatusNotFound)
			return
		}
		goal.GroupID = req.GroupID
	}
	if req.SortOrder != nil {
		goal.SortOrder = *req.SortOrder
	}

	goal.UpdatedAt = time.Now()

	if err := h.db.Save(&goal).Error; err != nil {
		http.Error(w, "Failed to update goal", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(goal)
}

func (h *GoalHandler) DeleteGoal(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)
	goalID := r.PathValue("id")

	parsedGoalID, err := uuid.Parse(goalID)
	if err != nil {
		http.Error(w, "Invalid goal ID", http.StatusBadRequest)
		return
	}

	result := h.db.Where("id = ? AND user_id = ?", parsedGoalID, userID).Delete(&models.Goal{})
	if result.Error != nil {
		http.Error(w, "Failed to delete goal", http.StatusInternalServerError)
		return
	}
	if result.RowsAffected == 0 {
		http.Error(w, "Goal not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *GoalHandler) CreateGoalGroup(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)

	var req models.CreateGoalGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	group := models.GoalGroup{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		ColorCode:   req.ColorCode,
		SortOrder:   req.SortOrder,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.db.Create(&group).Error; err != nil {
		http.Error(w, "Failed to create goal group", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(group)
}

func (h *GoalHandler) GetGoalGroups(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)

	var groups []models.GoalGroup
	if err := h.db.Where("user_id = ?", userID).
		Order("sort_order ASC, created_at ASC").
		Preload("Goals").
		Find(&groups).Error; err != nil {
		http.Error(w, "Failed to fetch goal groups", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groups)
}