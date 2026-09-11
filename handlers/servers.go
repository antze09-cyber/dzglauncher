package handlers

import (
	"net/http"
	"os"
	"path/filepath"

	"dzglauncher/a2s"
	"dzglauncher/config"
	"dzglauncher/steam"
)

type App struct {
	Cfg     *config.Config
	CfgPath string
	Steam   *steam.Client
	A2S     *a2s.Client
}

func NewApp(cfg *config.Config) *App {
	return &App{Cfg: cfg, Steam: steam.New(cfg.SteamAPIKey), A2S: a2s.New()}
}

// ConfigHandler возвращает публичную конфигурацию лаунчера.
func (h *App) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"gameId":        h.Cfg.GameID,
		"gameName":      h.Cfg.GameName,
		"defaultServer": h.Cfg.DefaultServer,
		"mods":          h.Cfg.Mods,
		"modParam":      h.Cfg.ModParam(),
	})
}

// ServerListHandler возвращает список серверов из Steam API.
func (h *App) ServerListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	servers, err := h.Steam.ServerList(h.Cfg.GameID, 10000)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "steam api: "+err.Error())
		return
	}
	// Отбрасываем служебные/прокси-записи и тестовые инстансы.
	clean := make([]steam.GameServer, 0, len(servers))
	seen := make(map[string]bool)
	for _, s := range servers {
		if s.Proxy || s.Addr == "" || seen[s.Addr] {
			continue
		}
		seen[s.Addr] = true
		clean = append(clean, s)
		if len(clean) >= 2000 {
			break
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"servers": clean, "count": len(clean)})
}

// ServerLiveHandler отдаёт актуальные данные одного сервера по A2S.
func (h *App) ServerLiveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	addr := r.URL.Query().Get("addr")
	if addr == "" {
		writeErr(w, http.StatusBadRequest, "missing addr")
		return
	}
	info, ping, err := h.A2S.Info(addr)
	if err != nil {
		writeErr(w, http.StatusGatewayTimeout, "a2s info: "+err.Error())
		return
	}
	players, err := h.A2S.Players(addr)
	if err != nil {
		players = nil
	}
	// Переиспользуем живые данные A2S вместо Steam
	writeJSON(w, http.StatusOK, map[string]any{
		"addr":   addr,
		"name":   info.Name,
		"map":    info.Map,
		"game":   info.Game,
		"folder": info.Folder,
		"appId":  info.GameID,
		"version": info.Version,
		"players":   info.Players,
		"maxPlayers": info.MaxPlayers,
		"bots":       info.Bots,
		"keywords":   info.Keywords,
		"password":   false,
		"ping":       ping.Milliseconds(),
		"playerList": players,
	})
}

// ServerModsHandler возвращает моды сервера: A2S keywords -> workshop IDs ->
// названия через Steam API + статус установки локально.
func (h *App) ServerModsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	addr := r.URL.Query().Get("addr")
	if addr == "" {
		writeErr(w, http.StatusBadRequest, "missing addr")
		return
	}
	info, _, err := h.A2S.Info(addr)
	if err != nil {
		writeErr(w, http.StatusGatewayTimeout, "a2s info: "+err.Error())
		return
	}
	ids := a2s.ParseModsFromKeywords(info.Keywords)
	if len(ids) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"addr":     addr,
			"keywords": info.Keywords,
			"mods":     []map[string]any{},
		})
		return
	}

	_, workshopDir := h.effectiveDirs()
	details, _ := h.Steam.PublishedFileDetails(ids)
	byID := make(map[string]struct {
		Title string
	}, len(details))
	for _, d := range details {
		byID[d.ID] = struct {
			Title string
		}{Title: d.Title}
	}

	mods := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		installed := false
		valid := false
		var issues []string
		path := ""
		if workshopDir != "" {
			p := filepath.Join(workshopDir, id)
			if fi, err := os.Stat(p); err == nil && fi.IsDir() {
				installed = true
				path = p
				v := validateMod(p)
				issues = v.Issues
				valid = v.HasMeta && v.PBOCount > 0
			}
		}
		mods = append(mods, map[string]any{
			"workshopId": id,
			"title":      byID[id].Title,
			"installed":  installed,
			"valid":      valid,
			"path":       path,
			"issues":     issues,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"addr":       addr,
		"workshopDir": workshopDir,
		"keywords":   info.Keywords,
		"count":      len(mods),
		"mods":       mods,
	})
}

// ServersLiveHandler массово обновляет A2S-информацию для списка адресов.
func (h *App) ServersLiveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Addrs []string `json:"addrs"`
	}
	if err := decodeJSON(r, &body); err != nil || len(body.Addrs) == 0 {
		writeErr(w, http.StatusBadRequest, "invalid body: expected {\"addrs\": [...]}")
		return
	}

	const limit = 20
	sem := make(chan struct{}, limit)
	results := make(chan map[string]any, len(body.Addrs))
	for _, addr := range body.Addrs {
		sem <- struct{}{}
		go func(a string) {
			defer func() { <-sem }()
			info, ping, err := h.A2S.Info(a)
			if err != nil {
				results <- map[string]any{"addr": a, "error": err.Error()}
				return
			}
			results <- map[string]any{
				"addr":   a,
				"name":   info.Name,
				"map":    info.Map,
				"players":   info.Players,
				"maxPlayers": info.MaxPlayers,
				"bots":       info.Bots,
				"version":    info.Version,
				"keywords":   info.Keywords,
				"appId":      info.GameID,
				"ping":       ping.Milliseconds(),
			}
		}(addr)
	}
	out := make([]map[string]any, 0, len(body.Addrs))
	for range body.Addrs {
		out = append(out, <-results)
	}
	writeJSON(w, http.StatusOK, map[string]any{"servers": out})
}