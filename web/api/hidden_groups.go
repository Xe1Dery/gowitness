package api

import (
	"encoding/json"
	"net/http"

	"github.com/sensepost/gowitness/pkg/log"
	"github.com/sensepost/gowitness/pkg/models"
	"gorm.io/gorm"
)

// hiddenGroupMaxLimit caps the number of hidden groups returned in one response.
const hiddenGroupMaxLimit = 1000

// HiddenGroupRequest is the request payload for hiding or unhiding a
// perception hash group from general results.
type HiddenGroupRequest struct {
	// PerceptionHashGroupID is the group to hide or unhide.
	PerceptionHashGroupID uint `json:"perception_hash_group_id"`
	// HiddenByResultID is the result from which the group was hidden (optional).
	HiddenByResultID uint `json:"hidden_by_result_id"`
	// Notes is a free-form note about why the group was hidden (optional).
	Notes string `json:"notes"`
}

// HiddenGroupItem is a single hidden group returned by the API.
type HiddenGroupItem struct {
	PerceptionHashGroupID uint   `json:"perception_hash_group_id"`
	HiddenByResultID      uint   `json:"hidden_by_result_id"`
	Count                 int64  `json:"count"`
	Notes                 string `json:"notes"`
	HiddenAt              string `json:"hidden_at"`
}

// hiddenGroupsResponse is the response of GET /api/results/hidden-groups.
type hiddenGroupsResponse struct {
	Total          int               `json:"total"`
	HiddenGroupIDs []uint            `json:"hidden_group_ids"`
	Groups         []HiddenGroupItem `json:"groups"`
}

// excludeHiddenGroups adds a filter to a results query that removes any
// result belonging to a perception hash group that has been hidden.
func excludeHiddenGroups(db *gorm.DB) *gorm.DB {
	hidden := db.Session(&gorm.Session{NewDB: true}).Model(&models.HiddenGroup{}).
		Select("perception_hash_group_id")
	return db.Where("perception_hash_group_id NOT IN (?)", hidden)
}

// GetHiddenGroupsHandler returns the list of hidden perception hash groups
// with the number of results currently in each group.
//
//	@Summary		Hidden groups
//	@Description	List perception hash groups hidden from general results.
//	@Tags			Results
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	hiddenGroupsResponse
//	@Router			/results/hidden-groups [get]
func (h *ApiHandler) GetHiddenGroupsHandler(w http.ResponseWriter, r *http.Request) {
	var hidden []models.HiddenGroup
	if err := h.DB.Order("created_at desc").Limit(hiddenGroupMaxLimit).Find(&hidden).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// current member count per group id
	counts := map[uint]int64{}
	type groupCount struct {
		PerceptionHashGroupID uint
		N                     int64
	}
	var gc []groupCount
	if err := h.DB.Model(&models.Result{}).
		Select("perception_hash_group_id, count(*) as n").
		Group("perception_hash_group_id").
		Scan(&gc).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, c := range gc {
		counts[c.PerceptionHashGroupID] = c.N
	}

	ids := make([]uint, 0, len(hidden))
	items := make([]HiddenGroupItem, 0, len(hidden))
	for _, hg := range hidden {
		ids = append(ids, hg.PerceptionHashGroupID)
		items = append(items, HiddenGroupItem{
			PerceptionHashGroupID: hg.PerceptionHashGroupID,
			HiddenByResultID:      hg.HiddenByResultID,
			Count:                 counts[hg.PerceptionHashGroupID],
			Notes:                 hg.Notes,
			HiddenAt:              hg.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}

	jsonData, err := json.Marshal(hiddenGroupsResponse{
		Total:          len(items),
		HiddenGroupIDs: ids,
		Groups:         items,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
}

// PostHiddenGroupsHandler hides or unhides a perception hash group. When the
// group is already hidden the request unhides it, otherwise it hides it.
//
//	@Summary		Toggle hidden group
//	@Description	Hide or unhide a perception hash group from general results.
//	@Tags			Results
//	@Accept			json
//	@Produce		json
//	@Param			query	body		HiddenGroupRequest	true	"The group to hide or unhide."
//	@Success		200		{object}	map[string]any
//	@Router			/results/hidden-groups [post]
func (h *ApiHandler) PostHiddenGroupsHandler(w http.ResponseWriter, r *http.Request) {
	var req HiddenGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("failed to read json request", "err", err)
		http.Error(w, "Error reading JSON request", http.StatusInternalServerError)
		return
	}

	if req.PerceptionHashGroupID == 0 {
		http.Error(w, "a non-zero perception_hash_group_id is required", http.StatusBadRequest)
		return
	}

	var existing models.HiddenGroup
	err := h.DB.Where("perception_hash_group_id = ?", req.PerceptionHashGroupID).First(&existing).Error
	if err == nil {
		// already hidden -> unhide
		if err := h.DB.Delete(&existing).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeHiddenToggleResponse(w, req.PerceptionHashGroupID, false)
		return
	} else if err != gorm.ErrRecordNotFound {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hg := models.HiddenGroup{
		PerceptionHashGroupID: req.PerceptionHashGroupID,
		HiddenByResultID:      req.HiddenByResultID,
		Notes:                 req.Notes,
	}
	if err := h.DB.Create(&hg).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeHiddenToggleResponse(w, req.PerceptionHashGroupID, true)
}

func writeHiddenToggleResponse(w http.ResponseWriter, groupID uint, hidden bool) {
	jsonData, err := json.Marshal(map[string]any{
		"hidden":                   hidden,
		"perception_hash_group_id": groupID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)
}
