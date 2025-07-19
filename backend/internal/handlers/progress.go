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

type ProgressHandler struct {
	db *gorm.DB
}

func NewProgressHandler(db *gorm.DB) *ProgressHandler {
	return &ProgressHandler{db: db}
}

func (h *ProgressHandler) CreateProgress(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)

	var req models.CreateProgressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var goal models.Goal
	if err := h.db.Where("id = ? AND user_id = ?", req.GoalID, userID).First(&goal).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Goal not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch goal", http.StatusInternalServerError)
		return
	}

	var existingProgress models.Progress
if err := h.db.Where("goal_id = ? AND user_id = ? AND DATE(tracked_date) = DATE(?)", 
	req.GoalID, userID, req.TrackedDate).First(&existingProgress).Error; err == nil {
		http.Error(w, "Progress entry already exists for this date", http.StatusConflict)
		return
	}

	convertedValue := goal.ConvertInputToBaseUnit(req.Value)

	progress := models.Progress{
		ID:          uuid.New(),
		GoalID:      req.GoalID,
		UserID:      userID,
		Value:       convertedValue,
		Notes:       req.Notes,
		TrackedDate: req.TrackedDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	progress.CalculateCompletionRate(goal.Target)

	if err := h.db.Create(&progress).Error; err != nil {
		http.Error(w, "Failed to create progress entry", http.StatusInternalServerError)
		return
	}

	if err := h.db.Preload("Goal").First(&progress, progress.ID).Error; err != nil {
		http.Error(w, "Failed to fetch created progress", http.StatusInternalServerError)
		return
	}

	response := models.ProgressResponse{
		Progress: progress,
		Goal:     progress.Goal,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *ProgressHandler) CreateTimeProgress(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)

	var req models.CreateProgressTimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var goal models.Goal
	if err := h.db.Where("id = ? AND user_id = ?", req.GoalID, userID).First(&goal).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Goal not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch goal", http.StatusInternalServerError)
		return
	}

	if goal.Type != models.GoalTypeTime {
		http.Error(w, "Goal type must be 'time' for this endpoint", http.StatusBadRequest)
		return
	}

	var existingProgress models.Progress
	if err := h.db.Where("goal_id = ? AND user_id = ? AND tracked_date = ?", 
		req.GoalID, userID, req.TrackedDate.Format("2006-01-02")).First(&existingProgress).Error; err == nil {
		http.Error(w, "Progress entry already exists for this date", http.StatusConflict)
		return
	}

	totalMinutes := req.ConvertToMinutes()

	progress := models.Progress{
		ID:          uuid.New(),
		GoalID:      req.GoalID,
		UserID:      userID,
		Value:       totalMinutes,
		Notes:       req.Notes,
		TrackedDate: req.TrackedDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	progress.CalculateCompletionRate(goal.Target)

	if err := h.db.Create(&progress).Error; err != nil {
		http.Error(w, "Failed to create progress entry", http.StatusInternalServerError)
		return
	}

	if err := h.db.Preload("Goal").First(&progress, progress.ID).Error; err != nil {
		http.Error(w, "Failed to fetch created progress", http.StatusInternalServerError)
		return
	}

	response := models.ProgressResponse{
		Progress: progress,
		Goal:     progress.Goal,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *ProgressHandler) GetProgress(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	query := h.db.Where("user_id = ?", userID)

	if goalID := r.URL.Query().Get("goal_id"); goalID != "" {
		if parsedGoalID, err := uuid.Parse(goalID); err == nil {
			query = query.Where("goal_id = ?", parsedGoalID)
		}
	}

	if startDate := r.URL.Query().Get("start_date"); startDate != "" {
		if parsed, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("tracked_date >= ?", parsed)
		}
	}

	if endDate := r.URL.Query().Get("end_date"); endDate != "" {
		if parsed, err := time.Parse("2006-01-02", endDate); err == nil {
			query = query.Where("tracked_date <= ?", parsed)
		}
	}

	var progress []models.Progress
	offset := (page - 1) * limit

	if err := query.Preload("Goal").
		Order("tracked_date DESC, created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&progress).Error; err != nil {
		http.Error(w, "Failed to fetch progress entries", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(progress)
}

func (h *ProgressHandler) GetProgressByID(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)
	progressID := r.PathValue("id")

	parsedProgressID, err := uuid.Parse(progressID)
	if err != nil {
		http.Error(w, "Invalid progress ID", http.StatusBadRequest)
		return
	}

	var progress models.Progress
	if err := h.db.Where("id = ? AND user_id = ?", parsedProgressID, userID).
		Preload("Goal").
		First(&progress).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Progress entry not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch progress entry", http.StatusInternalServerError)
		return
	}

	response := models.ProgressResponse{
		Progress: progress,
		Goal:     progress.Goal,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ProgressHandler) UpdateProgress(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)
	progressID := r.PathValue("id")

	parsedProgressID, err := uuid.Parse(progressID)
	if err != nil {
		http.Error(w, "Invalid progress ID", http.StatusBadRequest)
		return
	}

	var req models.UpdateProgressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var progress models.Progress
	if err := h.db.Where("id = ? AND user_id = ?", parsedProgressID, userID).
		Preload("Goal").
		First(&progress).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Progress entry not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch progress entry", http.StatusInternalServerError)
		return
	}

	if req.Value != nil {
		convertedValue := progress.Goal.ConvertInputToBaseUnit(*req.Value)
		progress.Value = convertedValue
		progress.CalculateCompletionRate(progress.Goal.Target)
	}
	if req.Notes != nil {
		progress.Notes = *req.Notes
	}
	if req.TrackedDate != nil {
		var existingProgress models.Progress
		if err := h.db.Where("goal_id = ? AND user_id = ? AND tracked_date = ? AND id != ?", 
			progress.GoalID, userID, req.TrackedDate.Format("2006-01-02"), progressID).First(&existingProgress).Error; err == nil {
			http.Error(w, "Progress entry already exists for this date", http.StatusConflict)
			return
		}
		progress.TrackedDate = *req.TrackedDate
	}

	progress.UpdatedAt = time.Now()

	if err := h.db.Save(&progress).Error; err != nil {
		http.Error(w, "Failed to update progress entry", http.StatusInternalServerError)
		return
	}

	response := models.ProgressResponse{
		Progress: progress,
		Goal:     progress.Goal,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ProgressHandler) DeleteProgress(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)
	progressID := r.PathValue("id")

	parsedProgressID, err := uuid.Parse(progressID)
	if err != nil {
		http.Error(w, "Invalid progress ID", http.StatusBadRequest)
		return
	}

	result := h.db.Where("id = ? AND user_id = ?", parsedProgressID, userID).Delete(&models.Progress{})
	if result.Error != nil {
		http.Error(w, "Failed to delete progress entry", http.StatusInternalServerError)
		return
	}
	if result.RowsAffected == 0 {
		http.Error(w, "Progress entry not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProgressHandler) GetHeatmapData(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)

	var startDate, endDate time.Time
	var err error

	if start := r.URL.Query().Get("start_date"); start != "" {
		startDate, err = time.Parse("2006-01-02", start)
		if err != nil {
			http.Error(w, "Invalid start_date format", http.StatusBadRequest)
			return
		}
	} else {
		startDate = time.Now().AddDate(-1, 0, 0) // Default to 1 year ago
	}

	if end := r.URL.Query().Get("end_date"); end != "" {
		endDate, err = time.Parse("2006-01-02", end)
		if err != nil {
			http.Error(w, "Invalid end_date format", http.StatusBadRequest)
			return
		}
	} else {
		endDate = time.Now() // Default to today
	}

	var heatmapData []models.HeatmapData
	query := `
		SELECT 
			p.tracked_date as date,
			p.completion_rate,
			p.value,
			g.title as goal_title,
			g.type as goal_type,
			g.color_code,
			p.notes
		FROM progress p
		JOIN goals g ON p.goal_id = g.id
		WHERE p.user_id = ? AND p.tracked_date BETWEEN ? AND ?
		ORDER BY p.tracked_date ASC
	`

	if err := h.db.Raw(query, userID, startDate, endDate).Scan(&heatmapData).Error; err != nil {
		http.Error(w, "Failed to fetch heatmap data", http.StatusInternalServerError)
		return
	}

	for i := range heatmapData {
		var goal models.Goal
		if err := h.db.Where("title = ?", heatmapData[i].GoalTitle).First(&goal).Error; err == nil {
			heatmapData[i].FormattedValue = goal.FormatValueDisplay(heatmapData[i].Value)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(heatmapData)
}