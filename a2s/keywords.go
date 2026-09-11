package a2s

import "strings"

// ParseModsFromKeywords извлекает список workshop-идентификаторов из поля
// keywords ответа A2S_INFO. Формат DayZ: "mod=2116157322,1559212036,act=27".
func ParseModsFromKeywords(kw string) []string {
	idx := strings.Index(kw, "mod=")
	if idx < 0 {
		return nil
	}
	rest := kw[idx+len("mod="):]
	// Обрезаем по концу строки или служебным маркерам.
	if i := strings.IndexAny(rest, "\x00 "); i >= 0 {
		rest = rest[:i]
	}
	seen := make(map[string]bool)
	var out []string
	for _, part := range strings.Split(rest, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		// Отбрасываем хвост вида "act=27" или "noslots".
		if i := strings.IndexAny(trimmed, "=&"); i >= 0 {
			trimmed = trimmed[:i]
		}
		if !isNumeric(trimmed) {
			continue
		}
		if seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}