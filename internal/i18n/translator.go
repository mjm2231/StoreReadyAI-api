package i18n

import (
	"fmt"
	"strings"
)

// 核心工具类
type Translator struct {
	bundle        *Bundle
	defaultLocale string
}

func NewTranslator(bundle *Bundle, defaultLocale string) *Translator {
	if defaultLocale == "" {
		defaultLocale = LocaleENUS
	}
	return &Translator{
		bundle:        bundle,
		defaultLocale: defaultLocale,
	}
}

func (t *Translator) T(locale, key string, params map[string]any) string {
	if strings.TrimSpace(key) == "" {
		return ""
	}

	locale = NormalizeLocale(locale)

	text, ok := t.bundle.Get(locale, key)
	if !ok && locale != t.defaultLocale {
		text, ok = t.bundle.Get(t.defaultLocale, key)
	}
	if !ok {
		return key
	}

	return replaceParams(text, params)
}

func replaceParams(text string, params map[string]any) string {
	if len(params) == 0 || text == "" {
		return text
	}

	result := text
	for k, v := range params {
		placeholder := "{" + k + "}"
		result = strings.ReplaceAll(result, placeholder, fmt.Sprint(v))
	}
	return result
}

func (t *Translator) HasKey(locale, key string) bool {
	locale = NormalizeLocale(locale)
	_, ok := t.bundle.Get(locale, key)
	if ok {
		return true
	}
	if locale != t.defaultLocale {
		_, ok = t.bundle.Get(t.defaultLocale, key)
		return ok
	}
	return false
}
