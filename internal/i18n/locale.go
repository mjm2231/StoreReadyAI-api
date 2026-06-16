package i18n

import "strings"

// 负责 locale 规范化和 fallback。
//
// 当前支持语言：
//   - en-US
//   - zh-CN
//   - zh-HK
//   - ja-JP
//
// 规范化示例：
//   - en            -> en-US
//   - en-US         -> en-US
//   - zh            -> zh-CN
//   - zh_CN         -> zh-CN
//   - zh-Hans       -> zh-CN
//   - zh-TW         -> zh-HK
//   - zh-Hant       -> zh-HK
//   - ja            -> ja-JP
//   - ja-JP         -> ja-JP
//
// 后续如果要新增语言，需要同步修改以下位置：
//  1. 在本文件 const 中增加新的 locale 常量。
//  2. 在 supportedLocales 中补充语言别名映射。
//  3. 在 app/app.go 初始化 i18n bundle 时，把新 locale 加入加载列表。
//  4. 在 internal/i18n/messages/ 下新增对应的词条文件，例如 fr-FR.json。
//  5. 如前端也需要该语言，前端 locales 资源与语言枚举也要同步补齐。
const (
	LocaleENUS = "en-US"
	LocaleZHCN = "zh-CN"
	LocaleZHHK = "zh-HK"
	LocaleJAJP = "ja-JP"
)

// supportedLocales 定义请求头/用户资料里常见语言写法到标准 locale 的映射。
// key 建议统一使用小写，value 统一使用标准格式（语言小写 + 地区大写）。
//
// 新增语言示例：
//
//	如果要增加法语（法国）：
//	1. 先增加常量：LocaleFRFR = "fr-FR"
//	2. 再补映射：
//	   "fr":    LocaleFRFR,
//	   "fr-fr": LocaleFRFR,
var supportedLocales = map[string]string{
	"en":    LocaleENUS,
	"en-us": LocaleENUS,

	"zh":      LocaleZHCN,
	"zh-cn":   LocaleZHCN,
	"zh-hans": LocaleZHCN,
	"zh-sg":   LocaleZHCN,

	"zh-hk":   LocaleZHHK,
	"zh-tw":   LocaleZHHK,
	"zh-hant": LocaleZHHK,
	"zh-mo":   LocaleZHHK,

	"ja":    LocaleJAJP,
	"ja-jp": LocaleJAJP,
}

// NormalizeLocale 将外部传入的 locale 归一到系统支持的标准 locale。
//
// 示例：
//
//	NormalizeLocale("en")      => "en-US"
//	NormalizeLocale("zh-cn")   => "zh-CN"
//	NormalizeLocale("zh-hant") => "zh-HK"
//	NormalizeLocale("ja")      => "ja-JP"
//	NormalizeLocale("")        => "en-US"
func NormalizeLocale(locale string) string {
	v := strings.TrimSpace(strings.ToLower(locale))
	if v == "" {
		return LocaleENUS
	}
	if normalized, ok := supportedLocales[v]; ok {
		return normalized
	}
	return LocaleENUS
}

// ResolveLocale 按优先级解析最终 locale：
//  1. X-Locale
//  2. 用户资料里的 locale
//  3. Accept-Language
//  4. 默认 en-US
//
// 示例：
//
//	ResolveLocale("zh-CN", "", "en-US,en;q=0.9") => "zh-CN"
//	ResolveLocale("", "ja-JP", "en-US,en;q=0.9") => "ja-JP"
//	ResolveLocale("", "", "zh-HK,zh;q=0.9")     => "zh-HK"
//	ResolveLocale("", "", "")                   => "en-US"
func ResolveLocale(xLocale, userLocale, acceptLanguage string) string {
	if v := NormalizeLocaleFromHeader(xLocale); v != "" {
		return v
	}
	if strings.TrimSpace(userLocale) != "" {
		return NormalizeLocale(userLocale)
	}
	if v := NormalizeLocaleFromAcceptLanguage(acceptLanguage); v != "" {
		return v
	}
	return LocaleENUS
}

// NormalizeLocaleFromHeader 处理单值语言头，例如 X-Locale。
//
// 示例：
//
//	NormalizeLocaleFromHeader("zh_CN") => "zh-CN"
//	NormalizeLocaleFromHeader("ja")    => "ja-JP"
func NormalizeLocaleFromHeader(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	return NormalizeLocale(v)
}

// NormalizeLocaleFromAcceptLanguage 从 Accept-Language 中提取第一个系统支持的语言。
//
// 示例：
//
//	NormalizeLocaleFromAcceptLanguage("zh-CN,zh;q=0.9,en;q=0.8") => "zh-CN"
//	NormalizeLocaleFromAcceptLanguage("fr-FR,ja-JP;q=0.9")      => "ja-JP"
//	NormalizeLocaleFromAcceptLanguage("")                       => ""
func NormalizeLocaleFromAcceptLanguage(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}

	parts := strings.Split(v, ",")
	for _, p := range parts {
		item := strings.TrimSpace(strings.Split(p, ";")[0])
		if item == "" {
			continue
		}
		if normalized, ok := supportedLocales[strings.ToLower(item)]; ok {
			return normalized
		}
	}
	return ""
}
