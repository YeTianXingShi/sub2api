package dto

import "github.com/Wei-Shaw/sub2api/internal/service"

type ModelMarketplacePricing struct {
	PricingMode                 string                            `json:"pricing_mode"`
	PriceStatus                 string                            `json:"price_status"`
	InputPricePerToken          float64                           `json:"input_price_per_token,omitempty"`
	ImageInputPricePerToken     float64                           `json:"image_input_price_per_token,omitempty"`
	OutputPricePerToken         float64                           `json:"output_price_per_token,omitempty"`
	CacheWritePricePerToken     float64                           `json:"cache_write_price_per_token,omitempty"`
	CacheReadPricePerToken      float64                           `json:"cache_read_price_per_token,omitempty"`
	ImageOutputPricePerToken    float64                           `json:"image_output_price_per_token,omitempty"`
	ImagePrice1K                float64                           `json:"image_price_1k,omitempty"`
	ImagePrice2K                float64                           `json:"image_price_2k,omitempty"`
	ImagePrice4K                float64                           `json:"image_price_4k,omitempty"`
	FastInputPricePerToken      float64                           `json:"fast_input_price_per_token,omitempty"`
	FastOutputPricePerToken     float64                           `json:"fast_output_price_per_token,omitempty"`
	FastCacheWritePricePerToken float64                           `json:"fast_cache_write_price_per_token,omitempty"`
	FastCacheReadPricePerToken  float64                           `json:"fast_cache_read_price_per_token,omitempty"`
	ContextIntervals            []ModelMarketplacePricingInterval `json:"context_intervals,omitempty"`
}

type ModelMarketplacePricingInterval struct {
	MinTokens                   int     `json:"min_tokens"`
	MaxTokens                   *int    `json:"max_tokens,omitempty"`
	InputPricePerToken          float64 `json:"input_price_per_token,omitempty"`
	ImageInputPricePerToken     float64 `json:"image_input_price_per_token,omitempty"`
	OutputPricePerToken         float64 `json:"output_price_per_token,omitempty"`
	CacheWritePricePerToken     float64 `json:"cache_write_price_per_token,omitempty"`
	CacheReadPricePerToken      float64 `json:"cache_read_price_per_token,omitempty"`
	ImageOutputPricePerToken    float64 `json:"image_output_price_per_token,omitempty"`
	FastInputPricePerToken      float64 `json:"fast_input_price_per_token,omitempty"`
	FastOutputPricePerToken     float64 `json:"fast_output_price_per_token,omitempty"`
	FastCacheWritePricePerToken float64 `json:"fast_cache_write_price_per_token,omitempty"`
	FastCacheReadPricePerToken  float64 `json:"fast_cache_read_price_per_token,omitempty"`
}

type ModelMarketplaceModel struct {
	ID          string                  `json:"id"`
	DisplayName string                  `json:"display_name"`
	Pricing     ModelMarketplacePricing `json:"pricing"`
}

type ModelMarketplaceGroup struct {
	ID                   int64                     `json:"id"`
	Name                 string                    `json:"name"`
	Description          string                    `json:"description"`
	Platform             string                    `json:"platform"`
	DisplayBrand         string                    `json:"display_brand"`
	SortOrder            int                       `json:"sort_order"`
	RateMultiplier       float64                   `json:"rate_multiplier"`
	ImageRateIndependent bool                      `json:"image_rate_independent"`
	ImageRateMultiplier  float64                   `json:"image_rate_multiplier"`
	DataSharingEnabled   bool                      `json:"data_sharing_enabled"`
	Capacity             *ModelMarketplaceCapacity `json:"capacity,omitempty"`
	ModelCount           int                       `json:"model_count"`
	Models               []ModelMarketplaceModel   `json:"models"`
}

type ModelMarketplaceCapacity struct {
	ConcurrencyUsed int `json:"concurrency_used"`
	ConcurrencyMax  int `json:"concurrency_max"`
	SessionsUsed    int `json:"sessions_used"`
	SessionsMax     int `json:"sessions_max"`
	RPMUsed         int `json:"rpm_used"`
	RPMMax          int `json:"rpm_max"`
}

func ModelMarketplaceGroupsFromService(groups []service.ModelMarketplaceGroup) []ModelMarketplaceGroup {
	out := make([]ModelMarketplaceGroup, 0, len(groups))
	for _, group := range groups {
		models := make([]ModelMarketplaceModel, 0, len(group.Models))
		for _, model := range group.Models {
			intervals := make([]ModelMarketplacePricingInterval, 0, len(model.Pricing.ContextIntervals))
			for _, interval := range model.Pricing.ContextIntervals {
				intervals = append(intervals, ModelMarketplacePricingInterval{
					MinTokens: interval.MinTokens, MaxTokens: interval.MaxTokens,
					InputPricePerToken: interval.InputPricePerToken, ImageInputPricePerToken: interval.ImageInputPricePerToken,
					OutputPricePerToken: interval.OutputPricePerToken, CacheWritePricePerToken: interval.CacheWritePricePerToken,
					CacheReadPricePerToken: interval.CacheReadPricePerToken, ImageOutputPricePerToken: interval.ImageOutputPricePerToken,
					FastInputPricePerToken: interval.FastInputPricePerToken, FastOutputPricePerToken: interval.FastOutputPricePerToken,
					FastCacheWritePricePerToken: interval.FastCacheWritePricePerToken, FastCacheReadPricePerToken: interval.FastCacheReadPricePerToken,
				})
			}
			models = append(models, ModelMarketplaceModel{
				ID:          model.ID,
				DisplayName: model.DisplayName,
				Pricing: ModelMarketplacePricing{
					PricingMode:                 model.Pricing.PricingMode,
					PriceStatus:                 model.Pricing.PriceStatus,
					InputPricePerToken:          model.Pricing.InputPricePerToken,
					ImageInputPricePerToken:     model.Pricing.ImageInputPricePerToken,
					OutputPricePerToken:         model.Pricing.OutputPricePerToken,
					CacheWritePricePerToken:     model.Pricing.CacheWritePricePerToken,
					CacheReadPricePerToken:      model.Pricing.CacheReadPricePerToken,
					ImageOutputPricePerToken:    model.Pricing.ImageOutputPricePerToken,
					ImagePrice1K:                model.Pricing.ImagePrice1K,
					ImagePrice2K:                model.Pricing.ImagePrice2K,
					ImagePrice4K:                model.Pricing.ImagePrice4K,
					FastInputPricePerToken:      model.Pricing.FastInputPricePerToken,
					FastOutputPricePerToken:     model.Pricing.FastOutputPricePerToken,
					FastCacheWritePricePerToken: model.Pricing.FastCacheWritePricePerToken,
					FastCacheReadPricePerToken:  model.Pricing.FastCacheReadPricePerToken,
					ContextIntervals:            intervals,
				},
			})
		}

		out = append(out, ModelMarketplaceGroup{
			ID:                   group.ID,
			Name:                 group.Name,
			Description:          group.Description,
			Platform:             group.Platform,
			DisplayBrand:         group.DisplayBrand,
			SortOrder:            group.SortOrder,
			RateMultiplier:       group.RateMultiplier,
			ImageRateIndependent: group.ImageRateIndependent,
			ImageRateMultiplier:  group.ImageRateMultiplier,
			DataSharingEnabled:   group.DataSharingEnabled,
			Capacity:             modelMarketplaceCapacity(group.Capacity),
			ModelCount:           group.ModelCount,
			Models:               models,
		})
	}

	return out
}

func modelMarketplaceCapacity(capacity *service.GroupCapacitySummary) *ModelMarketplaceCapacity {
	if capacity == nil {
		return nil
	}
	return &ModelMarketplaceCapacity{ConcurrencyUsed: capacity.ConcurrencyUsed, ConcurrencyMax: capacity.ConcurrencyMax, SessionsUsed: capacity.SessionsUsed, SessionsMax: capacity.SessionsMax, RPMUsed: capacity.RPMUsed, RPMMax: capacity.RPMMax}
}
