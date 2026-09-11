package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"dzglauncher/a2s"
)

// ModStatus описывает состояние установки мода.
type ModStatus struct {
	Name               string   `json:"name"`
	WorkshopID         string   `json:"workshopId"`
	ResolvedWorkshopID string   `json:"resolvedWorkshopId"`
	Installed          bool     `json:"installed"`
	Valid              bool     `json:"valid"`
	Path               string   `json:"path"`
	Size               int64    `json:"size"`
	HasMeta            bool     `json:"hasMeta"`
	PBOCount           int      `json:"pboCount"`
	BikeyCount         int      `json:"bikeyCount"`
	Issues             []string `json:"issues"`
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
	h.resolveWorkshopIDs(out)
	writeJSON(w, http.StatusOK, map[string]any{
		"workshopDir": workshopDir,
		"missingDir":  missingDir,
		"mods":        out,
	})
}

// serverModInfo — workshop-элемент, загруженный с сервера (A2S keywords + Steam API).
type serverModInfo struct {
	ID    string
	Title string
}

// fetchServerMods получает ID и названия модов, которые запускает сервер addr.
func (h *App) fetchServerMods(addr string) (keywords string, mods []serverModInfo, err error) {
	info, _, err := h.A2S.Info(addr)
	if err != nil {
		return "", nil, err
	}
	ids := a2s.ParseModsFromKeywords(info.Keywords)
	if len(ids) == 0 {
		return info.Keywords, nil, nil
	}
	details, _ := h.Steam.PublishedFileDetails(ids)
	byID := make(map[string]string, len(details))
	for _, d := range details {
		byID[d.ID] = d.Title
	}
	out := make([]serverModInfo, 0, len(ids))
	for _, id := range ids {
		out = append(out, serverModInfo{ID: id, Title: byID[id]})
	}
	return info.Keywords, out, nil
}

// normalizeModName приводит название мода к сопоставимому виду: без "@", нижний регистр, только буквы/цифры.
func normalizeModName(s string) string {
	s = strings.ToLower(strings.TrimPrefix(s, "@"))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// matchModTitle ищет workshop ID мода по его имени среди модов сервера
// (точное совпадение, затем — вхождение одной строки в другую).
func matchModTitle(modName string, serverMods []serverModInfo) (string, bool) {
	n := normalizeModName(modName)
	if n == "" || len(serverMods) == 0 {
		return "", false
	}
	for _, m := range serverMods {
		if normalizeModName(m.Title) == n {
			return m.ID, true
		}
	}
	for _, m := range serverMods {
		t := normalizeModName(m.Title)
		if len(n) >= 4 && len(t) >= 4 && (strings.Contains(n, t) || strings.Contains(t, n)) {
			return m.ID, true
		}
	}
	return "", false
}

// resolveWorkshopIDs заполняет ResolvedWorkshopID для модов без явного workshopId,
// подбирая их по модам, которые запускает сервер по умолчанию. Сервер опрашивается
// один раз за сессию — результат кэшируется.
func (h *App) resolveWorkshopIDs(mods []ModStatus) {
	if h.Cfg.DefaultServer == "" {
		return
	}
	if h.resolvedQueried {
		for i := range mods {
			if mods[i].WorkshopID == "" {
				mods[i].ResolvedWorkshopID = h.resolvedCache[mods[i].Name]
			}
		}
		return
	}
	_, sm, err := h.fetchServerMods(h.Cfg.DefaultServer)
	h.resolvedQueried = true
	if err != nil {
		return
	}
	for i := range mods {
		if mods[i].WorkshopID != "" {
			continue
		}
		if id, ok := matchModTitle(mods[i].Name, sm); ok {
			mods[i].ResolvedWorkshopID = id
			h.resolvedCache[mods[i].Name] = id
		}
	}
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
