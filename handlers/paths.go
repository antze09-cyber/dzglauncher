package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func statDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// steamInstallPath возвращает основной каталог Steam из реестра.
func steamInstallPath() string {
	specs := []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Valve\Steam`},
		{registry.LOCAL_MACHINE, `SOFTWARE\Valve\Steam`},
	}
	for _, s := range specs {
		k, err := registry.OpenKey(s.root, s.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		installPath, _, err := k.GetStringValue("InstallPath")
		k.Close()
		if err == nil && statDir(installPath) {
			return installPath
		}
	}
	return ""
}

// steamLibraries собирает все папки библиотек Steam:
// основная из реестра + дополнительные из libraryfolders.vdf.
func steamLibraries() []string {
	var dirs []string
	primary := steamInstallPath()
	if primary != "" {
		dirs = append(dirs, primary)
		if data, err := os.ReadFile(filepath.Join(primary, "steamapps", "libraryfolders.vdf")); err == nil {
			re := regexp.MustCompile(`"(\d+)"\s+"([^"]+)"`)
			for _, m := range re.FindAllStringSubmatch(string(data), -1) {
				p := strings.ReplaceAll(m[2], `\\`, `\`)
				if statDir(p) {
					dirs = append(dirs, p)
				}
			}
		}
	}
	return dedupe(dirs)
}

func dedupe(in []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(in))
	for _, s := range in {
		norm := strings.ToLower(strings.TrimRight(s, `\`))
		if seen[norm] {
			continue
		}
		seen[norm] = true
		out = append(out, s)
	}
	return out
}

// detectPaths ищет каталог DayZ и папку воркшопа по всем библиотекам Steam.
func detectPaths(gameID int) (gameDir, workshopDir string) {
	for _, lib := range steamLibraries() {
		steamapps := filepath.Join(lib, "steamapps")
		if !statDir(steamapps) {
			continue
		}
		if gameDir == "" {
			g := filepath.Join(steamapps, "common", "DayZ")
			if statDir(g) {
				gameDir = g
			}
		}
		if workshopDir == "" {
			w := filepath.Join(steamapps, "workshop", "content", fmt.Sprintf("%d", gameID))
			if statDir(w) {
				workshopDir = w
			}
		}
		if gameDir != "" && workshopDir != "" {
			break
		}
	}
	return
}

// effectiveDirs возвращает итоговые пути: пользовательское переопределение
// имеет приоритет, иначе — автоопределение.
func (h *App) effectiveDirs() (game, workshop string) {
	dGame, dWorkshop := detectPaths(h.Cfg.GameID)
	game, workshop = dGame, dWorkshop
	if h.Cfg.GameDir != "" && statDir(h.Cfg.GameDir) {
		game = h.Cfg.GameDir
	}
	if h.Cfg.WorkshopDir != "" && statDir(h.Cfg.WorkshopDir) {
		workshop = h.Cfg.WorkshopDir
	}
	return
}
