package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// /负责加载词条文件
type Bundle struct {
	messages map[string]map[string]string
}

func NewBundle(dir string, locales []string) (*Bundle, error) {
	b := &Bundle{
		messages: make(map[string]map[string]string),
	}

	for _, locale := range locales {
		path := filepath.Join(dir, locale+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read locale file failed, locale=%s err=%w", locale, err)
		}

		var dict map[string]string
		if err = json.Unmarshal(data, &dict); err != nil {
			return nil, fmt.Errorf("unmarshal locale file failed, locale=%s err=%w", locale, err)
		}
		b.messages[locale] = dict
	}

	return b, nil
}

func (b *Bundle) Get(locale, key string) (string, bool) {
	if b == nil {
		return "", false
	}
	dict, ok := b.messages[locale]
	if !ok {
		return "", false
	}
	val, ok := dict[key]
	return val, ok
}
