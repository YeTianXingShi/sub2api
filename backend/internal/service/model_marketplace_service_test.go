package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type marketplaceGroupRepoStub struct {
	GroupRepository
	groups []Group
	err    error
}

func (s *marketplaceGroupRepoStub) ListActive(context.Context) ([]Group, error) {
	return s.groups, s.err
}

func TestModelMarketplaceListPublicFiltersGroupsAndMapsPricing(t *testing.T) {
	repo := &marketplaceGroupRepoStub{groups: []Group{
		{ID: 1, Name: "Public", Description: "public models", Platform: PlatformOpenAI, SortOrder: 10, RateMultiplier: 2, ImageRateIndependent: true, ImageRateMultiplier: 0.5, ActiveAccountCount: 1},
		{ID: 2, Name: "Exclusive", Platform: PlatformOpenAI, IsExclusive: true, ActiveAccountCount: 1},
		{ID: 3, Name: "No accounts", Platform: PlatformOpenAI, ActiveAccountCount: 0},
		{ID: 4, Name: "Unsupported", Platform: "unsupported", ActiveAccountCount: 1},
	}}
	svc := NewModelMarketplaceService(repo, nil, NewBillingService(&config.Config{}, nil), nil)

	groups, err := svc.ListPublic(context.Background())

	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, int64(1), groups[0].ID)
	require.Equal(t, "OpenAI", groups[0].DisplayBrand)
	require.Equal(t, 10, groups[0].SortOrder)
	require.Equal(t, 2.0, groups[0].RateMultiplier)
	require.NotEmpty(t, groups[0].Models)
	require.Equal(t, len(groups[0].Models), groups[0].ModelCount)
	require.NotEqual(t, "", groups[0].Models[0].ID)
	var imagePricing *ModelDisplayPricing
	for i := range groups[0].Models {
		if groups[0].Models[i].ID == "gpt-image-1" {
			imagePricing = &groups[0].Models[i].Pricing
			break
		}
	}
	require.NotNil(t, imagePricing)
	require.Equal(t, "image", imagePricing.PricingMode)
	require.InDelta(t, defaultImageGenerationPrice*0.5, imagePricing.ImagePrice1K, 1e-12)
}

func TestModelMarketplaceIncludesGrokDefaults(t *testing.T) {
	models := defaultMarketplaceModelDefs(PlatformGrok)
	require.NotEmpty(t, models)
	require.Equal(t, "grok-4.5", models[0].ID)
	require.Equal(t, "xAI", marketplaceDisplayBrand(PlatformGrok, "fallback"))
}

func TestGetDisplayPricingIncludesLongContextIntervals(t *testing.T) {
	svc := NewBillingService(&config.Config{}, nil)

	pricing := svc.GetDisplayPricing("gpt-5.4", 2, nil)

	require.Equal(t, "token", pricing.PricingMode)
	require.Equal(t, "priced", pricing.PriceStatus)
	require.Len(t, pricing.ContextIntervals, 2)
	require.NotNil(t, pricing.ContextIntervals[0].MaxTokens)
	require.Equal(t, pricing.ContextIntervals[0].MaxTokens, &pricing.ContextIntervals[1].MinTokens)
	require.Greater(t, pricing.ContextIntervals[1].InputPricePerToken, pricing.ContextIntervals[0].InputPricePerToken)
	require.Greater(t, pricing.ContextIntervals[1].OutputPricePerToken, pricing.ContextIntervals[0].OutputPricePerToken)
}

func TestNormalizeUsageRankingLimit(t *testing.T) {
	require.Equal(t, 1, normalizeUsageRankingLimit("1"))
	require.Equal(t, 100, normalizeUsageRankingLimit("100"))
	for _, invalid := range []string{"", "0", "101", "not-a-number"} {
		require.Equal(t, DefaultUsageRankingLimit, normalizeUsageRankingLimit(invalid))
	}
}
