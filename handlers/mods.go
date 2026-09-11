package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ModStatus описывает состояние установки мода.
type ModStatus struct {
	Name       string   `json:"name"`
	WorkshopID string   `json:"workshopId"`
	Installed  bool     `json:"installed"`
	Valid      bool     `json:"valid"`
	Path       string   `json:"path"`
	Size       int64    `json:"size"`
	HasMeta    bool     `json:"hasMeta"`
	PBOCount   int      `json:"pboCount"`
	BikeyCount int      `json:"bikeyCount"`
	Issues     []string `json:"issues"`
}

// ModValidation содержит результат проверки структуры мода.
type ModValidation struct {
	HasMeta    bool
	PBOCount   int
	BikeyCount int
	Issues     []string
}

// validateMod проверяет структуру папки мода DayZ:
// наличие meta.cpp, PBO-файлов в Addons/ и .bikey-ключей.
// Валидным считается мод с meta.cpp и хотя бы одним PBO;
// отсутствие .bikey — предупреждение (возможно неподписанные PBO).
func validateMod(path string) ModValidation {
	v := ModValidation{}
	if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
		v.Issues = append(v.Issues, "папка мода не найдена")
		return v
	}
	_ = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := strings.ToLower(info.Name())
		switch {
		case name == "meta.cpp":
			v.HasMeta = true
		case strings.HasSuffix(name, ".pbo"):
			v.PBOCount++
		case strings.HasSuffix(name, ".bikey"):
			v.BikeyCount++
		}
		return nil
	})
	if !v.HasMeta {
		v.Issues = append(v.Issues, "нет meta.cpp — мод не распознается DayZ")
	}
	if v.PBOCount == 0 {
		v.Issues = append(v.Issues, "нет Addons/*.pbo — пустой мод")
	}
	if v.BikeyCount == 0 {
		v.Issues = append(v.Issues, "нет *.bikey — подпись PBO не проверяется")
	}
	return v
}

// ModsHandler проверяет наличие и валидность всех модов в мастерской.
func (h *App) ModsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	_, workshopDir := h.effectiveDirs()
	missingDir := ""
	if workshopDir == "" {
		missingDir = "папка мастерской DayZ не найдена (steamapps/workshop/content/" +
			fmt.Sprint(h.Cfg.GameID) + ")"
	}
	out := make([]ModStatus, 0, len(h.Cfg.Mods))
	for _, m := range h.Cfg.Mods {
		st := ModStatus{Name: m.Name, WorkshopID: m.WorkshopID}
		if workshopDir != "" {
			path := filepath.Join(workshopDir, m.Name)
			if fi, err := os.Stat(path); err == nil && fi.IsDir() {
				st.Installed = true
				st.Path = path
				st.Size = dirSize(path)
				v := validateMod(path)
				st.HasMeta = v.HasMeta
				st.PBOCount = v.PBOCount
				st.BikeyCount = v.BikeyCount
				st.Issues = v.Issues
				st.Valid = v.HasMeta && v.PBOCount > 0
			}
		}
		out = append(out, st)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"workshopDir": workshopDir,
		"missingDir":  missingDir,
		"mods":        out,
	})
}

// ModSearchHandler ищет мод в мастерской Steam.
func (h *App) ModSearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	term := strings.TrimPrefix(r.URL.Query().Get("q"), "@")
	if term == "" {
		writeErr(w, http.StatusBadRequest, "missing q")
		return
	}
	items, err := h.Steam.SearchWorkshop(h.Cfg.GameID, term, 8)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "workshop search: "+err.Error())
		return
	}
	for i := range items {
		items[i].URL = fmt.Sprintf("https://steamcommunity.com/sharedfiles/filedetails/?id=%s",
			url.QueryEscape(items[i].ID))
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": items})
}

func dirSize(path string) int64 {
	var total int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}