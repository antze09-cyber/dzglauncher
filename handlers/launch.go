package handlers

import (
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

// LaunchHandler запускает DayZ через Steam с параметрами подключения к серверу.
func (h *App) LaunchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Server   string `json:"server"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	addr := strings.TrimSpace(body.Server)
	if addr == "" {
		addr = h.Cfg.DefaultServer
	}
	launchURL := buildLaunchURL(h.Cfg.GameID, addr, body.Password, h.Cfg.ModParam(), h.Cfg.LaunchParams)
	if err := openURL(launchURL); err != nil {
		writeErr(w, http.StatusInternalServerError, "не удалось запустить Steam: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "url": launchURL})
}

// buildLaunchURL собирает URL вида:
// steam://rungameid/221100//-connect=IP:PORT -password=... -mod=@CF;@Dabs;... [launchParams]
func buildLaunchURL(gameID int, addr, password, modParam, launchParams string) string {
	s := "steam://rungameid/" + strconv.Itoa(gameID) + "//-connect=" + addr
	if password != "" {
		s += " -password=" + password
	}
	if modParam != "" {
		s += " -mod=" + modParam
	}
	if launchParams != "" {
		s += " " + launchParams
	}
	return s
}

func openURL(target string) error {
	cmd := exec.Command("cmd", "/c", "start", "", target)
	return cmd.Start()
}
