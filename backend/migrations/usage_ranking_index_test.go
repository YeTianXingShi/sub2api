package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUsageRankingCoveringIndexMigration(t *testing.T) {
	content, err := FS.ReadFile("191_usage_ranking_covering_index_notx.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_ranking_cover")
	require.Contains(t, sql, "ON usage_logs ( created_at, user_id )")
	require.Contains(t, sql, "actual_cost")
	require.NotContains(t, strings.ToLower(sql), "update usage_logs")
	require.NotContains(t, strings.ToLower(sql), "delete from usage_logs")
}
