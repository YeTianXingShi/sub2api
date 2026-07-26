package dto

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestModelMarketplaceDTOContainsPricingButNoAvailabilityProbeData(t *testing.T) {
	maxTokens := 100000
	groups := ModelMarketplaceGroupsFromService([]service.ModelMarketplaceGroup{{
		ID: 1, Name: "Public", Platform: service.PlatformOpenAI, ModelCount: 1,
		Models: []service.ModelMarketplaceModel{{
			ID: "gpt-5", DisplayName: "GPT-5",
			Pricing: service.ModelDisplayPricing{
				PricingMode: "token", PriceStatus: "priced",
				ContextIntervals: []service.ModelDisplayPricingInterval{{MinTokens: 0, MaxTokens: &maxTokens, InputPricePerToken: 0.000001}},
			},
		}},
	}})

	payload, err := json.Marshal(groups)
	require.NoError(t, err)
	jsonText := string(payload)
	require.Contains(t, jsonText, `"context_intervals"`)
	require.NotContains(t, strings.ToLower(jsonText), "availability")
	require.NotContains(t, strings.ToLower(jsonText), "probe")
}
