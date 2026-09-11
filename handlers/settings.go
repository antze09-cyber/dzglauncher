package handlers

import (
	"net/http"
	"strings"
)

// SettingsResponse описывает текущее состояние путей DayZ.
type SettingsResponse struct {
	GameDir             string `json:"gameDir"`
	WorkshopDir         string `json:"workshopDir"`
	OverrideGameDir     string `json:"overrideGameDir"`
	OverrideWorkshopDir string `json:"overrideWorkshopDir"`
	DetectedGameDir     string `json:"detectedGameDir"`
	DetectedWorkshopDir string `json:"detectedWorkshopDir"`
	GameInstalled       bool   `json:"gameInstalled"`
	WorkshopFound       bool   `json:"workshopFound"`
	DefaultServer       string `json:"defaultServer"`
	GameID              int    `json:"gameId"`
}

func (h *App) buildSettings() SettingsResponse {
	detGame, detWorkshop := detectPaths(h.Cfg.GameID)
	effGame, effWorkshop := h.effectiveDirs()
	return SettingsResponse{
		GameDir:             effGame,
		WorkshopDir:         effWorkshop,
		OverrideGameDir:     h.Cfg.GameDir,
		OverrideWorkshopDir: h.Cfg.WorkshopDir,
		DetectedGameDir:     detGame,
		DetectedWorkshopDir: detWorkshop,
		GameInstalled:       statDir(effGame),
		WorkshopFound:       statDir(effWorkshop),
		DefaultServer:       h.Cfg.DefaultServer,
		GameID:              h.Cfg.GameID,
	}
}

// SettingsHandler отдаёт статус путей (GET) или сохраняет пользовательские пути (POST).
// Пустая строка в POST означает "автоопределение".
func (h *App) SettingsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, h.buildSettings())
	case http.MethodPost:
		var body struct {
			GameDir     string `json:"gameDir"`
			WorkshopDir string `json:"workshopDir"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid body")
			return
		}
		gameDir := strings.TrimSpace(body.GameDir)
		workshopDir := strings.TrimSpace(body.WorkshopDir)
		if gameDir != "" && !statDir(gameDir) {
			writeErr(w, http.StatusBadRequest, "папка игры не найдена: "+gameDir)
			return
		}
		if workshopDir != "" && !statDir(workshopDir) {
			writeErr(w, http.StatusBadRequest, "папка мастерской не найдена: "+workshopDir)
			return
		}
		h.Cfg.GameDir = gameDir
		h.Cfg.WorkshopDir = workshopDir
		if h.CfgPath != "" {
			if err := h.Cfg.Save(h.CfgPath); err != nil {
				writeErr(w, http.StatusInternalServerError, "не удалось сохранить: "+err.Error())
				return
			}
		}
		writeJSON(w, http.StatusOK, h.buildSettings())
	default:
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}