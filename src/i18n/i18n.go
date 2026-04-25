package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"sync"
)

type contextKey struct{}

type Localizer struct {
	Lang     string
	messages map[string]string
	fallback *Localizer
}

func (l *Localizer) T(key string) string {
	if msg, ok := l.messages[key]; ok {
		return msg
	}
	if l.fallback != nil {
		return l.fallback.T(key)
	}
	return key
}

func (l *Localizer) Tf(key string, args ...any) string {
	return fmt.Sprintf(l.T(key), args...)
}

func (l *Localizer) Errorf(key string, args ...any) error {
	return fmt.Errorf(l.T(key), args...)
}

func NewContext(ctx context.Context, loc *Localizer) context.Context {
	return context.WithValue(ctx, contextKey{}, loc)
}

func FromContext(ctx context.Context) *Localizer {
	if loc, ok := ctx.Value(contextKey{}).(*Localizer); ok {
		return loc
	}
	return Default()
}

var (
	registry   = map[string]*Localizer{}
	defaultLoc *Localizer
	mu         sync.RWMutex
)

func Load(localesFS fs.FS) error {
	entries, err := fs.ReadDir(localesFS, ".")
	if err != nil {
		return fmt.Errorf("failed to read locales directory: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := fs.ReadFile(localesFS, entry.Name())
		if err != nil {
			return fmt.Errorf("failed to read locale file %s: %w", entry.Name(), err)
		}

		var messages map[string]string
		if err := json.Unmarshal(data, &messages); err != nil {
			return fmt.Errorf("failed to parse locale file %s: %w", entry.Name(), err)
		}

		lang := strings.TrimSuffix(entry.Name(), ".json")
		registry[lang] = &Localizer{
			Lang:     lang,
			messages: messages,
		}
	}

	if en, ok := registry["en"]; ok {
		for lang, loc := range registry {
			if lang != "en" {
				loc.fallback = en
			}
		}
		defaultLoc = en
	} else if len(registry) > 0 {
		for _, loc := range registry {
			defaultLoc = loc
			break
		}
	}

	return nil
}

func Get(lang string) *Localizer {
	mu.RLock()
	defer mu.RUnlock()

	if loc, ok := registry[lang]; ok {
		return loc
	}

	parts := strings.SplitN(lang, "-", 2)
	if len(parts) > 1 {
		if loc, ok := registry[parts[0]]; ok {
			return loc
		}
	}

	return Default()
}

func Default() *Localizer {
	mu.RLock()
	defer mu.RUnlock()

	if defaultLoc != nil {
		return defaultLoc
	}
	return &Localizer{Lang: "en", messages: map[string]string{}}
}

func Available() []string {
	mu.RLock()
	defer mu.RUnlock()

	langs := make([]string, 0, len(registry))
	for lang := range registry {
		langs = append(langs, lang)
	}
	return langs
}
