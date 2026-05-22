package service

import "runtime"

// PlatformUIInfo describes the native shell and visual defaults that the
// frontend should use for platform-specific styling.
type PlatformUIInfo struct {
	Platform             string `json:"platform"`
	GOOS                 string `json:"goos"`
	Appearance           string `json:"appearance"`
	SupportsTranslucency bool   `json:"supportsTranslucency"`
	SupportsMica         bool   `json:"supportsMica"`
}

// PlatformService exposes native platform details to the frontend.
type PlatformService struct{}

func NewPlatformService() *PlatformService {
	return &PlatformService{}
}

func (s *PlatformService) GetPlatformUIInfo() PlatformUIInfo {
	info := PlatformUIInfo{
		GOOS:       runtime.GOOS,
		Appearance: "system",
	}

	switch runtime.GOOS {
	case "darwin":
		info.Platform = "macos"
		info.SupportsTranslucency = true
	case "windows":
		info.Platform = "windows"
		info.SupportsTranslucency = true
		info.SupportsMica = true
	case "linux":
		info.Platform = "linux"
	default:
		info.Platform = runtime.GOOS
	}

	return info
}
