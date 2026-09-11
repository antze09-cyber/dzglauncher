//go:build windows

package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	webview "github.com/jchv/go-webview2"

	"dzglauncher/config"
	"dzglauncher/handlers"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

//go:embed config.json
var defaultConfig []byte

//go:embed resources/app.ico
var appIcon []byte

var (
	user32        = syscall.NewLazyDLL("user32.dll")
	procLoadImage = user32.NewProc("LoadImageW")
	procSendMsg   = user32.NewProc("SendMessageW")
)

const (
	wmSetIcon     = 0x0080
	iconSmall     = 0
	iconBig       = 1
	imageIcon     = 1
	lrLoadFromFile = 0x10
)

// setWindowIcon заменяет иконку окна WebView2 на иконку лаунчера.
func setWindowIcon(hwnd uintptr, ico []byte) {
	f, err := os.CreateTemp("", "dzglauncher-*.ico")
	if err != nil {
		return
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(ico); err != nil {
		f.Close()
		return
	}
	f.Close()

	pathPtr, err := windows.UTF16PtrFromString(f.Name())
	if err != nil {
		return
	}
	hIcon, _, _ := procLoadImage.Call(0, uintptr(unsafe.Pointer(pathPtr)), imageIcon, 0, 0, lrLoadFromFile)
	if hIcon == 0 {
		return
	}
	procSendMsg.Call(hwnd, wmSetIcon, iconSmall, hIcon)
	procSendMsg.Call(hwnd, wmSetIcon, iconBig, hIcon)
}

func main() {
	dev := flag.Bool("dev", false, "development mode: не встраивать фронтенд, отдать на Vite")
	headless := flag.Bool("headless", false, "без окна WebView2 (только HTTP-сервер)")
	cfgPath := flag.String("config", "config.json", "путь к файлу конфигурации")
	flag.Parse()

	cfg, err := config.Load(*cfgPath, defaultConfig)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	app := handlers.NewApp(cfg)
	app.CfgPath = *cfgPath
	mux := http.NewServeMux()
	registerRoutes(mux, app, *dev)

	addr := fmt.Sprintf("127.0.0.1:%d", cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}
	log.Printf("API готов: http://%s", addr)

	go func() {
		if err := http.Serve(ln, handlers.CORS(mux)); err != nil {
			log.Fatalf("http: %v", err)
		}
	}()

	url := fmt.Sprintf("http://%s", addr)
	if *headless {
		select {}
	}

	w := webview.New(true)
	defer w.Destroy()
	if hwnd := uintptr(w.Window()); hwnd != 0 {
		setWindowIcon(hwnd, appIcon)
	}
	w.SetTitle("DayZ Launcher")
	w.SetSize(1280, 800, webview.HintNone)
	w.Navigate(url)
	w.Run()
}

func registerRoutes(mux *http.ServeMux, app *handlers.App, dev bool) {
	mux.HandleFunc("/api/config", app.ConfigHandler)
	mux.HandleFunc("/api/servers", app.ServerListHandler)
	mux.HandleFunc("/api/server", app.ServerLiveHandler)
	mux.HandleFunc("/api/server/mods", app.ServerModsHandler)
	mux.HandleFunc("/api/servers/live", app.ServersLiveHandler)
	mux.HandleFunc("/api/mods", app.ModsHandler)
	mux.HandleFunc("/api/mods/search", app.ModSearchHandler)
	mux.HandleFunc("/api/settings", app.SettingsHandler)
	mux.HandleFunc("/api/launch", app.LaunchHandler)

	if dev {
		return
	}
	sub, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		log.Fatalf("embed fs: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))
}