# DZLauncher

DayZ лаунчер для WOC сервера (146.66.12.4:2402).  
Go backend + SolidJS frontend, встроенный WebView2, onefile exe.

## Возможности

- Подключение к серверу через `steam://rungameid/221100` с автоматическим `-mod=...`
- Автопроверка модов перед запуском (meta.cpp / PBO / bikey валидация)
- Список модов сервера (A2S keywords → Steam API → локальный статус)
- Динамическое определение путей DayZ и мастерской Steam
- Вкладки: Сервер / Моды / Настройки
- Фоновое изображение (../backgroud/image0.png)

## Быстрый старт

### 1. Установить зависимости

```bash
# Go (1.24+)
# Node.js + npm
cd dzglauncher
npm install --prefix frontend
```

### 2. Настроить конфиг

Скопируйте `config.example.json` → `config.json` и вставьте свой Steam Web API ключ:

```bash
copy config.example.json config.json
```

### 3. Собрать

```bash
powershell -ExecutionPolicy Bypass -File build.ps1
```

Результат: `dist\dzglauncher.exe` (onefile, ~9 МБ).

### 4. Запуск

```bash
.\dist\dzglauncher.exe              # стандартный режим
.\dist\dzglauncher.exe -headless    # только HTTP-сервер (для тестов)
.\dist\dzglauncher.exe -dev         # фронтенд с Vite dev server
```

## Структура

```
dzglauncher/
├── main.go              # embed + HTTP + WebView2
├── build.ps1            # onefile сборка
├── config.json          # конфиг (НЕ коммитится — есть API ключ)
├── config.example.json  # шаблон конфига
├── a2s/                 # A2S клиент (UDP)
├── steam/               # Steam Web API клиент
├── handlers/            # REST обработчики
│   ├── servers.go       # live-запросы серверов
│   ├── mods.go          # валидация модов
│   ├── settings.go      # настройки путей
│   ├── paths.go         # автоопределение Steam/DayZ
│   └── launch.go        # steam:// запуск
├── frontend/            # SolidJS + Vite
│   ├── src/
│   │   ├── App.tsx
│   │   ├── api.ts
│   │   └── pages/
│   │       ├── ServerView.tsx     # сервер + моды сервера
│   │       ├── ModManager.tsx     # моды лаунчера
│   │       └── SettingsPanel.tsx  # настройки путей
│   └── public/
│       └── bg.png       # фоновое изображение
├── resources/
│   ├── winres.json      # иконка + манифест
│   └── app.ico          # иконка (авто)
├── icons/               # исходные PNG иконки
└── dist/                # результат сборки (НЕ коммитится)
```

## Development

```bash
# Frontend dev (порт 5173, проксирует /api → :8080)
cd frontend && npm run dev

# Backend dev
go run . -dev
```
