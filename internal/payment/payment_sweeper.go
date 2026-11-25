package payment

import (
	"context"
	"log/slog"
	"time"
)

type expirer interface {
	ExpireOverdue(ctx context.Context) (int64, error)
}

func RunSweeper(ctx context.Context, repo expirer, interval time.Duration) {
	slog.InfoContext(ctx, "payment sweeper started", "interval", interval)
	defer slog.InfoContext(ctx, "payment sweeper stopped")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := repo.ExpireOverdue(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.ErrorContext(ctx, "expire overdue payments", "error", err)
				continue
			}
			if n > 0 {
				slog.InfoContext(ctx, "expired overdue payments", "count", n)
			}
		}
	}
}
