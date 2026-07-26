package repository

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// GetUsageRanking aggregates successful, billed usage across all users. It intentionally
// exposes only a username or masked email and never returns an email address verbatim.
func (r *usageLogRepository) GetUsageRanking(ctx context.Context, startTime, endTime time.Time, limit int) (result *usagestats.UsageRankingResponse, err error) {
	if limit <= 0 {
		limit = service.DefaultUsageRankingLimit
	}
	query := `
		WITH user_usage AS (
			SELECT user_id,
				COUNT(*) AS requests,
				COALESCE(SUM(input_tokens), 0) AS input_tokens,
				COALESCE(SUM(output_tokens), 0) AS output_tokens,
				COALESCE(SUM(cache_creation_tokens), 0) AS cache_creation_tokens,
				COALESCE(SUM(cache_read_tokens), 0) AS cache_read_tokens,
				COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) AS total_tokens,
				COALESCE(SUM(actual_cost), 0) AS actual_cost
			FROM usage_logs
			WHERE created_at >= $1 AND created_at < $2
			GROUP BY user_id
			HAVING COALESCE(SUM(actual_cost), 0) > 0
		), ranked AS (
			SELECT ROW_NUMBER() OVER (ORDER BY actual_cost DESC, total_tokens DESC, requests DESC, user_id ASC) AS rank,
				*, SUM(requests) OVER () AS total_requests,
				SUM(total_tokens) OVER () AS ranking_total_tokens,
				SUM(actual_cost) OVER () AS total_actual_cost
			FROM user_usage
			ORDER BY actual_cost DESC, total_tokens DESC, requests DESC, user_id ASC
			LIMIT $3
		)
		SELECT r.rank, r.user_id, COALESCE(us.email, ''), COALESCE(us.username, ''),
			COALESCE(ua.url, ''), r.requests, r.input_tokens, r.output_tokens,
			r.cache_creation_tokens, r.cache_read_tokens, r.total_tokens, r.actual_cost,
			r.total_requests, r.ranking_total_tokens, r.total_actual_cost
		FROM ranked r
		LEFT JOIN users us ON us.id = r.user_id
		LEFT JOIN user_avatars ua ON ua.user_id = r.user_id
		ORDER BY r.rank ASC`

	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err, result = closeErr, nil
		}
	}()

	ranking := make([]usagestats.UsageRankingItem, 0, limit)
	var totalRequests, totalTokens int64
	var totalActualCost float64
	for rows.Next() {
		var row usagestats.UsageRankingItem
		var email, username string
		if err = rows.Scan(&row.Rank, &row.UserID, &email, &username, &row.AvatarURL,
			&row.Requests, &row.InputTokens, &row.OutputTokens, &row.CacheCreationTokens,
			&row.CacheReadTokens, &row.TotalTokens, &row.ActualCost, &totalRequests,
			&totalTokens, &totalActualCost); err != nil {
			return nil, err
		}
		row.DisplayName = strings.TrimSpace(username)
		if row.DisplayName == "" {
			row.DisplayName = service.MaskEmail(strings.TrimSpace(email))
		}
		if row.DisplayName == "" {
			row.DisplayName = "User"
		}
		ranking = append(ranking, row)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return &usagestats.UsageRankingResponse{Ranking: ranking, TotalRequests: totalRequests, TotalTokens: totalTokens, TotalActualCost: totalActualCost}, nil
}
