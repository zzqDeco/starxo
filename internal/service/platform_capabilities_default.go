//go:build !windows

package service

func supportsNativeMica() bool {
	return false
}
