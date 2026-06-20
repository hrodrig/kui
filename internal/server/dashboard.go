package server

import (
	"context"

	"github.com/hrodrig/kui/internal/ui"
)

func (s *Server) fillDashboardKiko(ctx context.Context, data *ui.DashboardData, host string, since, until string) {
	sum, err := s.kiko.Summary(ctx, host, since, until)
	if err != nil {
		data.KikoErr = err.Error()
		s.log.Warn("kiko summary: %v", err)
		return
	}
	data.Summary = &ui.DashboardSummary{
		Hits: sum.Hits, Uniques: sum.Uniques,
		TopPath: sum.TopPath, TopPathHits: sum.TopPathHits,
		Since: sum.Since, Until: sum.Until,
	}

	timeline, err := s.kiko.Timeline(ctx, host, since, until, "day")
	if err != nil {
		data.KikoErr = err.Error()
		s.log.Warn("kiko timeline: %v", err)
		return
	}
	data.Timeline = timeline

	paths, err := s.kiko.Paths(ctx, host, since, until, 10)
	if err != nil {
		data.KikoErr = err.Error()
		s.log.Warn("kiko paths: %v", err)
		return
	}
	data.Paths = paths

	refs, err := s.kiko.Refs(ctx, host, since, until, 10)
	if err != nil {
		data.KikoErr = err.Error()
		s.log.Warn("kiko refs: %v", err)
		return
	}
	data.Refs = refs

	channels, err := s.kiko.Channels(ctx, host, since, until, 8)
	if err != nil {
		data.KikoErr = err.Error()
		s.log.Warn("kiko channels: %v", err)
		return
	}
	data.Channels = channels
}
