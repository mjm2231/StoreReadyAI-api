package hander

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	CtxKeyDeviceID   = "device_id"
	CtxKeyPlatform   = "platform"
	CtxKeyAppVersion = "app_version"
	CtxKeyAppBuild   = "app_build"
	CtxKeyOSVersion  = "os_version"
	CtxKeyModel      = "device_model"
	CtxKeyChannel    = "channel"
	CtxKeyLocale     = "locale"
	CtxKeyTimezone   = "timezone"
	CtxKeyUserAgent  = "user_agent"
	CtxKeyTenantID   = "tenant_id"
)

type HeaderInfo struct {
	DeviceID   string `json:"device_id"`
	Platform   string `json:"platform"`
	AppVersion string `json:"app_version"`
	AppBuild   string `json:"app_build"`
	OSVersion  string `json:"os_version"`
	Model      string `json:"device_model"`
	Channel    string `json:"channel"`
	Locale     string `json:"locale"`
	Timezone   string `json:"timezone"`
	UserAgent  string `json:"user_agent"`
	TenantID   string `json:"tenant_id"`
}

func HeaderInfoMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		info := HeaderInfo{
			DeviceID:   cleanHeaderValue(firstNonEmpty(c.GetHeader("X-Device-Id"), c.GetHeader("X-Device-ID"), c.GetHeader("Device-Id"))),
			Platform:   cleanHeaderValue(firstNonEmpty(c.GetHeader("X-Platform"), c.GetHeader("Platform"))),
			AppVersion: cleanHeaderValue(firstNonEmpty(c.GetHeader("X-App-Version"), c.GetHeader("App-Version"))),
			AppBuild:   cleanHeaderValue(firstNonEmpty(c.GetHeader("X-App-Build"), c.GetHeader("App-Build"))),
			OSVersion:  cleanHeaderValue(firstNonEmpty(c.GetHeader("X-OS-Version"), c.GetHeader("OS-Version"))),
			Model:      cleanHeaderValue(firstNonEmpty(c.GetHeader("X-Device-Model"), c.GetHeader("Device-Model"))),
			Channel:    cleanHeaderValue(firstNonEmpty(c.GetHeader("X-Channel"), c.GetHeader("Channel"))),
			Locale:     normalizeLocale(cleanHeaderValue(firstNonEmpty(c.GetHeader("X-Locale"), c.GetHeader("Accept-Language")))),
			Timezone:   cleanHeaderValue(firstNonEmpty(c.GetHeader("X-Timezone"), c.GetHeader("Timezone"))),
			UserAgent:  cleanHeaderValue(c.Request.UserAgent()),
			TenantID:   cleanHeaderValue(firstNonEmpty(c.GetHeader("X-Tenant-Id"), c.GetHeader("X-Tenant-ID"), c.GetHeader("Tenant-Id"))),
		}

		setHeaderInfo(c, info)
		c.Next()
	}
}

func setHeaderInfo(c *gin.Context, info HeaderInfo) {
	c.Set(CtxKeyDeviceID, info.DeviceID)
	c.Set(CtxKeyPlatform, info.Platform)
	c.Set(CtxKeyAppVersion, info.AppVersion)
	c.Set(CtxKeyAppBuild, info.AppBuild)
	c.Set(CtxKeyOSVersion, info.OSVersion)
	c.Set(CtxKeyModel, info.Model)
	c.Set(CtxKeyChannel, info.Channel)
	c.Set(CtxKeyLocale, info.Locale)
	c.Set(CtxKeyTimezone, info.Timezone)
	c.Set(CtxKeyUserAgent, info.UserAgent)
	c.Set(CtxKeyTenantID, info.TenantID)
}

func GetHeaderInfo(c *gin.Context) HeaderInfo {
	if c == nil {
		return HeaderInfo{}
	}
	return HeaderInfo{
		DeviceID:   getString(c, CtxKeyDeviceID),
		Platform:   getString(c, CtxKeyPlatform),
		AppVersion: getString(c, CtxKeyAppVersion),
		AppBuild:   getString(c, CtxKeyAppBuild),
		OSVersion:  getString(c, CtxKeyOSVersion),
		Model:      getString(c, CtxKeyModel),
		Channel:    getString(c, CtxKeyChannel),
		Locale:     getString(c, CtxKeyLocale),
		Timezone:   getString(c, CtxKeyTimezone),
		UserAgent:  getString(c, CtxKeyUserAgent),
		TenantID:   getString(c, CtxKeyTenantID),
	}
}

func GetDeviceID(c *gin.Context) string {
	return getString(c, CtxKeyDeviceID)
}

func GetPlatform(c *gin.Context) string {
	return getString(c, CtxKeyPlatform)
}

func GetAppVersion(c *gin.Context) string {
	return getString(c, CtxKeyAppVersion)
}

func GetAppBuild(c *gin.Context) string {
	return getString(c, CtxKeyAppBuild)
}

func GetOSVersion(c *gin.Context) string {
	return getString(c, CtxKeyOSVersion)
}

func GetDeviceModel(c *gin.Context) string {
	return getString(c, CtxKeyModel)
}

func GetChannel(c *gin.Context) string {
	return getString(c, CtxKeyChannel)
}

func GetLocale(c *gin.Context) string {
	locale := normalizeLocale(getString(c, CtxKeyLocale))
	if locale == "" {
		return DefaultLocale()
	}
	return locale
}

func GetTimezone(c *gin.Context) string {
	return getString(c, CtxKeyTimezone)
}

func GetUserAgent(c *gin.Context) string {
	return getString(c, CtxKeyUserAgent)
}

func GetTenantID(c *gin.Context) string {
	return getString(c, CtxKeyTenantID)
}

func getString(c *gin.Context, key string) string {
	if c == nil {
		return ""
	}
	v, ok := c.Get(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func normalizeLocale(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}

	if idx := strings.Index(v, ","); idx >= 0 {
		v = v[:idx]
	}
	if idx := strings.Index(v, ";"); idx >= 0 {
		v = v[:idx]
	}

	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}

	v = strings.ReplaceAll(v, "_", "-")
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}

	parts := strings.Split(v, "-")
	if len(parts) == 0 {
		return ""
	}

	lang := strings.ToLower(strings.TrimSpace(parts[0]))
	if lang == "" {
		return ""
	}

	switch lang {
	case "en":
		if len(parts) == 1 {
			return "en-US"
		}
	case "ja":
		if len(parts) == 1 {
			return "ja-JP"
		}
	case "zh":
		if len(parts) == 1 {
			return "zh-CN"
		}
	}

	if lang == "zh" && len(parts) > 1 {
		scriptOrRegion := strings.ToLower(strings.TrimSpace(parts[1]))
		switch scriptOrRegion {
		case "hans":
			return "zh-CN"
		case "hant":
			return "zh-HK"
		case "cn", "sg":
			return "zh-CN"
		case "hk", "mo", "tw":
			return "zh-HK"
		}
	}

	if len(parts) == 1 {
		return lang
	}

	region := strings.TrimSpace(parts[1])
	if region == "" {
		return lang
	}

	if len(region) == 2 || len(region) == 3 {
		region = strings.ToUpper(region)
	} else {
		regionLower := strings.ToLower(region)
		if len(regionLower) > 0 {
			region = strings.ToUpper(regionLower[:1]) + regionLower[1:]
		} else {
			region = regionLower
		}
	}

	return fmt.Sprintf("%s-%s", lang, region)
}

func DefaultLocale() string {
	return "en-US"
}

func cleanHeaderValue(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	v = strings.ReplaceAll(v, "\n", " ")
	v = strings.ReplaceAll(v, "\r", " ")
	if len(v) > 256 {
		return v[:256]
	}
	return v
}
