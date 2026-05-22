//go:build windows

package service

import "golang.org/x/sys/windows"

func supportsNativeMica() bool {
	version := windows.RtlGetVersion()
	if version == nil {
		return false
	}
	return version.MajorVersion > 10 ||
		(version.MajorVersion == 10 && version.MinorVersion == 0 && version.BuildNumber >= 22621)
}
