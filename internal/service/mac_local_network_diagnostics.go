package service

import (
	"context"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"starxo/internal/config"
)

const macLocalNetworkResetCommand = "tccutil reset LocalNetwork com.starxo.app"

type MacLocalNetworkCheckResult struct {
	Platform                 string   `json:"platform"`
	Host                     string   `json:"host"`
	Port                     int      `json:"port"`
	IsMac                    bool     `json:"isMac"`
	IsLocalNetworkHost       bool     `json:"isLocalNetworkHost"`
	AppDialOK                bool     `json:"appDialOK"`
	AppDialError             string   `json:"appDialError,omitempty"`
	CLIAttempted             bool     `json:"cliAttempted"`
	CLIReachable             bool     `json:"cliReachable"`
	CLIError                 string   `json:"cliError,omitempty"`
	LikelyPermissionIssue    bool     `json:"likelyPermissionIssue"`
	ConfirmedPermissionIssue bool     `json:"confirmedPermissionIssue"`
	Summary                  string   `json:"summary"`
	FixSteps                 []string `json:"fixSteps"`
	ResetCommand             string   `json:"resetCommand"`
}

func (s *SettingsService) CheckMacLocalNetworkAccess(sshCfg config.SSHConfig) (MacLocalNetworkCheckResult, error) {
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return checkMacLocalNetworkAccess(ctx, sshCfg), nil
}

func checkMacLocalNetworkAccess(ctx context.Context, sshCfg config.SSHConfig) MacLocalNetworkCheckResult {
	host := normalizeDiagnosticHost(sshCfg.Host)
	port := sshCfg.Port
	if port == 0 {
		port = 22
	}
	result := MacLocalNetworkCheckResult{
		Platform:           runtime.GOOS,
		Host:               host,
		Port:               port,
		IsMac:              runtime.GOOS == "darwin",
		IsLocalNetworkHost: isDiagnosticLocalNetworkHost(host),
		ResetCommand:       macLocalNetworkResetCommand,
		FixSteps: []string{
			"Open System Settings > Privacy & Security > Local Network.",
			"Enable Starxo for local network access.",
			"Quit and reopen Starxo before retrying SSH.",
		},
	}
	if host == "" {
		result.Summary = "SSH host is empty."
		return result
	}
	if !result.IsMac {
		result.Summary = "Local Network privacy diagnostics only apply to macOS app bundles."
		return result
	}
	if !result.IsLocalNetworkHost {
		result.Summary = "The configured SSH host is not a local network address."
		return result
	}

	dialErr := dialTCPForDiagnostic(ctx, host, port)
	if dialErr == nil {
		result.AppDialOK = true
		result.Summary = "Starxo can reach the SSH host from the app process."
		return result
	}
	result.AppDialError = dialErr.Error()

	result.CLIAttempted, result.CLIReachable, result.CLIError = runNCReachabilityCheck(ctx, host, port)
	result.LikelyPermissionIssue = isMacLocalNetworkPermissionLikeError(dialErr)
	result.ConfirmedPermissionIssue = result.CLIReachable && !result.AppDialOK
	if result.ConfirmedPermissionIssue {
		result.LikelyPermissionIssue = true
		result.Summary = "Terminal can reach the SSH host, but the Starxo app process cannot. macOS Local Network permission or bundle identity cache is the likely blocker."
		return result
	}
	if result.LikelyPermissionIssue {
		result.Summary = "Starxo cannot reach the local network SSH host from the app process. Check macOS Local Network permission for Starxo."
		return result
	}
	result.Summary = "Starxo cannot reach the SSH host from the app process. Check host, port, routing, firewall, and macOS Local Network permission."
	return result
}

func dialTCPForDiagnostic(ctx context.Context, host string, port int) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	dialer := net.Dialer{Timeout: 4 * time.Second}
	conn, err := dialer.DialContext(timeoutCtx, "tcp", addr)
	if err != nil {
		return err
	}
	return conn.Close()
}

func runNCReachabilityCheck(ctx context.Context, host string, port int) (attempted bool, reachable bool, message string) {
	ncPath, err := lookPathWithFallback("nc", "/usr/bin/nc")
	if err != nil {
		return false, false, "nc command is not available"
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(timeoutCtx, ncPath, "-vz", "-G", "3", host, strconv.Itoa(port))
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if timeoutCtx.Err() != nil {
		return true, false, timeoutCtx.Err().Error()
	}
	if err != nil {
		if output != "" {
			return true, false, output
		}
		return true, false, err.Error()
	}
	if output != "" {
		return true, true, output
	}
	return true, true, "reachable"
}

func lookPathWithFallback(name string, fallback string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	if fallback != "" {
		if info, err := os.Stat(fallback); err == nil && !info.IsDir() {
			return fallback, nil
		}
	}
	return "", exec.ErrNotFound
}

func isMacLocalNetworkPermissionLikeError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "no route to host") ||
		strings.Contains(text, "network is unreachable") ||
		strings.Contains(text, "operation not permitted") ||
		strings.Contains(text, "permission denied")
}

func normalizeDiagnosticHost(host string) string {
	value := strings.TrimSpace(strings.ToLower(host))
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "[") {
		if end := strings.Index(value, "]"); end > 0 {
			return value[1:end]
		}
	}
	if strings.Count(value, ":") == 1 {
		if h, _, err := net.SplitHostPort(value); err == nil {
			return strings.Trim(h, "[]")
		}
	}
	return value
}

func isDiagnosticLocalNetworkHost(host string) bool {
	host = normalizeDiagnosticHost(host)
	if host == "" {
		return false
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".lan") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return isDiagnosticLocalIP(ip)
	}
	return false
}

func isDiagnosticLocalIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() ||
		isCarrierGradeNAT(ip)
}

func isCarrierGradeNAT(ip net.IP) bool {
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
	return v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127
}
