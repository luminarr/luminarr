// Package health provides system health checks for Luminarr.
// Checks cover disk space on library paths, download client connectivity,
// and indexer reachability.
package health

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"syscall"

	"github.com/luminarr/luminarr/internal/core/downloader"
	"github.com/luminarr/luminarr/internal/core/indexer"
	"github.com/luminarr/luminarr/internal/core/library"
)

// Status represents the health state of a single check or the overall system.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// CheckResult describes the outcome of a single health check.
type CheckResult struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Message string `json:"message"`
}

// Report is the aggregated result of all health checks.
type Report struct {
	// Status is the worst status across all checks.
	Status Status        `json:"status"`
	Checks []CheckResult `json:"checks"`
}

// Service runs health checks against the system's subsystems.
type Service struct {
	libSvc *library.Service
	dlSvc  *downloader.Service
	idxSvc *indexer.Service
	logger *slog.Logger
}

// NewService creates a new health Service.
func NewService(
	libSvc *library.Service,
	dlSvc *downloader.Service,
	idxSvc *indexer.Service,
	logger *slog.Logger,
) *Service {
	return &Service{libSvc: libSvc, dlSvc: dlSvc, idxSvc: idxSvc, logger: logger}
}

// Check runs all health checks and returns an aggregated report.
func (s *Service) Check(ctx context.Context) Report {
	var checks []CheckResult

	checks = append(checks, s.checkDiskSpace(ctx))
	checks = append(checks, s.checkDownloadClients(ctx))
	checks = append(checks, s.checkIndexers(ctx))

	// Overall status is the worst individual status.
	overall := StatusHealthy
	for _, c := range checks {
		if c.Status == StatusUnhealthy {
			overall = StatusUnhealthy
			break
		}
		if c.Status == StatusDegraded && overall != StatusUnhealthy {
			overall = StatusDegraded
		}
	}

	return Report{Status: overall, Checks: checks}
}

// checkDiskSpace verifies that each library's root path is accessible and has
// at least 1 GB of free disk space.
func (s *Service) checkDiskSpace(ctx context.Context) CheckResult {
	const minFreeBytes = 1 << 30 // 1 GiB

	libs, err := s.libSvc.List(ctx)
	if err != nil {
		return CheckResult{
			Name:    "disk_space",
			Status:  StatusDegraded,
			Message: fmt.Sprintf("could not list libraries: %v", err),
		}
	}
	if len(libs) == 0 {
		return CheckResult{
			Name:    "disk_space",
			Status:  StatusHealthy,
			Message: "no libraries configured",
		}
	}

	var issues []string
	for _, lib := range libs {
		free, err := diskFreeBytes(lib.RootPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", lib.Name, err))
			continue
		}
		if free < minFreeBytes {
			issues = append(issues, fmt.Sprintf("%s: only %s free", lib.Name, formatBytes(free)))
		}
	}

	if len(issues) > 0 {
		return CheckResult{
			Name:    "disk_space",
			Status:  StatusDegraded,
			Message: joinIssues(issues),
		}
	}
	return CheckResult{
		Name:    "disk_space",
		Status:  StatusHealthy,
		Message: fmt.Sprintf("%d library path(s) healthy", len(libs)),
	}
}

// checkDownloadClients pings each enabled download client.
func (s *Service) checkDownloadClients(ctx context.Context) CheckResult {
	clients, err := s.dlSvc.List(ctx)
	if err != nil {
		return CheckResult{
			Name:    "download_clients",
			Status:  StatusDegraded,
			Message: fmt.Sprintf("could not list download clients: %v", err),
		}
	}

	var enabled, failed int
	var issues []string
	for _, c := range clients {
		if !c.Enabled {
			continue
		}
		enabled++
		if err := s.dlSvc.Test(ctx, c.ID); err != nil {
			failed++
			issues = append(issues, fmt.Sprintf("%s: %v", c.Name, err))
		}
	}

	if enabled == 0 {
		return CheckResult{
			Name:    "download_clients",
			Status:  StatusHealthy,
			Message: "no download clients configured",
		}
	}
	if failed == enabled {
		return CheckResult{
			Name:    "download_clients",
			Status:  StatusUnhealthy,
			Message: joinIssues(issues),
		}
	}
	if failed > 0 {
		return CheckResult{
			Name:    "download_clients",
			Status:  StatusDegraded,
			Message: joinIssues(issues),
		}
	}
	return CheckResult{
		Name:    "download_clients",
		Status:  StatusHealthy,
		Message: fmt.Sprintf("%d download client(s) reachable", enabled),
	}
}

// checkIndexers pings each enabled indexer.
func (s *Service) checkIndexers(ctx context.Context) CheckResult {
	indexers, err := s.idxSvc.List(ctx)
	if err != nil {
		return CheckResult{
			Name:    "indexers",
			Status:  StatusDegraded,
			Message: fmt.Sprintf("could not list indexers: %v", err),
		}
	}

	var enabled, failed int
	var issues []string
	for _, idx := range indexers {
		if !idx.Enabled {
			continue
		}
		enabled++
		if err := s.idxSvc.Test(ctx, idx.ID); err != nil {
			failed++
			issues = append(issues, fmt.Sprintf("%s: %v", idx.Name, err))
		}
	}

	if enabled == 0 {
		return CheckResult{
			Name:    "indexers",
			Status:  StatusHealthy,
			Message: "no indexers configured",
		}
	}
	if failed == enabled {
		return CheckResult{
			Name:    "indexers",
			Status:  StatusUnhealthy,
			Message: joinIssues(issues),
		}
	}
	if failed > 0 {
		return CheckResult{
			Name:    "indexers",
			Status:  StatusDegraded,
			Message: joinIssues(issues),
		}
	}
	return CheckResult{
		Name:    "indexers",
		Status:  StatusHealthy,
		Message: fmt.Sprintf("%d indexer(s) reachable", enabled),
	}
}

// diskFreeBytes returns the number of free bytes available on the filesystem
// containing path. Returns an error if path does not exist or is inaccessible.
func diskFreeBytes(path string) (uint64, error) {
	if _, err := os.Stat(path); err != nil {
		return 0, fmt.Errorf("path not accessible: %w", err)
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("statfs: %w", err)
	}
	return stat.Bavail * uint64(stat.Bsize), nil //nolint:gosec // G115: Bsize is always positive on Linux
}

// formatBytes returns a human-readable byte count (e.g. "1.2 GB").
func formatBytes(b uint64) string {
	const (
		gib = 1 << 30
		mib = 1 << 20
	)
	switch {
	case b >= gib:
		return fmt.Sprintf("%.1f GB", float64(b)/gib)
	case b >= mib:
		return fmt.Sprintf("%.1f MB", float64(b)/mib)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// joinIssues concatenates a list of issue strings with "; ".
func joinIssues(issues []string) string {
	return strings.Join(issues, "; ")
}
