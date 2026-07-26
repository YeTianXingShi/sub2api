package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
)

type ModelMarketplaceGroup struct {
	ID                                        int64
	Name, Description, Platform, DisplayBrand string
	SortOrder                                 int
	RateMultiplier                            float64
	ImageRateIndependent                      bool
	ImageRateMultiplier                       float64
	DataSharingEnabled                        bool
	Capacity                                  *GroupCapacitySummary
	ModelCount                                int
	Models                                    []ModelMarketplaceModel
}

type ModelMarketplaceModel struct {
	ID          string
	DisplayName string
	Pricing     ModelDisplayPricing
}

type ModelMarketplaceService struct {
	groupRepo       GroupRepository
	gatewayService  *GatewayService
	billingService  *BillingService
	capacityService *GroupCapacityService
}

func NewModelMarketplaceService(
	groupRepo GroupRepository,
	gatewayService *GatewayService,
	billingService *BillingService,
	capacityService *GroupCapacityService,
) *ModelMarketplaceService {
	return &ModelMarketplaceService{
		groupRepo:       groupRepo,
		gatewayService:  gatewayService,
		billingService:  billingService,
		capacityService: capacityService,
	}
}

func (s *ModelMarketplaceService) ListPublic(ctx context.Context) ([]ModelMarketplaceGroup, error) {
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active groups: %w", err)
	}

	out := make([]ModelMarketplaceGroup, 0, len(groups))
	capacityMap := s.getPublicCapacityMap(ctx, groups)
	for i := range groups {
		group := &groups[i]
		if group.IsExclusive || group.ActiveAccountCount <= 0 {
			continue
		}

		models := s.listPublicModelsForGroup(ctx, group)
		if len(models) == 0 {
			continue
		}

		out = append(out, ModelMarketplaceGroup{
			ID:                   group.ID,
			Name:                 group.Name,
			Description:          group.Description,
			Platform:             group.Platform,
			DisplayBrand:         marketplaceDisplayBrand(group.Platform, group.Name),
			SortOrder:            group.SortOrder,
			RateMultiplier:       group.RateMultiplier,
			ImageRateIndependent: group.ImageRateIndependent,
			ImageRateMultiplier:  group.ImageRateMultiplier,
			DataSharingEnabled:   false,
			Capacity:             marketplaceGroupCapacity(capacityMap, group.ID),
			ModelCount:           len(models),
			Models:               models,
		})
	}

	return out, nil
}

func (s *ModelMarketplaceService) getPublicCapacityMap(ctx context.Context, groups []Group) map[int64]GroupCapacitySummary {
	if s.capacityService == nil {
		return nil
	}
	ids := make([]int64, 0, len(groups))
	for _, group := range groups {
		if !group.IsExclusive && group.ActiveAccountCount > 0 {
			ids = append(ids, group.ID)
		}
	}
	result, err := s.capacityService.GetGroupCapacityByIDs(ctx, ids)
	if err != nil {
		return nil
	}
	return result
}

func marketplaceGroupCapacity(capacities map[int64]GroupCapacitySummary, id int64) *GroupCapacitySummary {
	capacity, ok := capacities[id]
	if !ok {
		return nil
	}
	return &capacity
}

func (s *ModelMarketplaceService) listPublicModelsForGroup(ctx context.Context, group *Group) []ModelMarketplaceModel {
	modelDefs := s.resolveGroupModels(ctx, group)
	if len(modelDefs) == 0 {
		return nil
	}

	imageConfig := &ImagePriceConfig{
		Price1K: group.ImagePrice1K,
		Price2K: group.ImagePrice2K,
		Price4K: group.ImagePrice4K,
	}

	models := make([]ModelMarketplaceModel, 0, len(modelDefs))
	for _, modelDef := range modelDefs {
		pricing := ModelDisplayPricing{
			PricingMode: "unknown",
			PriceStatus: "unpriced",
		}
		if s.billingService != nil {
			pricing = s.billingService.GetDisplayPricing(modelDef.ID, group.RateMultiplier, imageConfig)
			if group.ImageRateIndependent && pricing.PricingMode == "image" {
				pricing = s.billingService.GetDisplayPricing(modelDef.ID, group.ImageRateMultiplier, imageConfig)
			}
		}

		models = append(models, ModelMarketplaceModel{
			ID:          modelDef.ID,
			DisplayName: modelDef.DisplayName,
			Pricing:     pricing,
		})
	}

	return models
}

func (s *ModelMarketplaceService) resolveGroupModels(ctx context.Context, group *Group) []marketplaceModelDef {
	if s.gatewayService != nil {
		groupID := group.ID
		modelIDs := s.gatewayService.GetAvailableModels(ctx, &groupID, "")
		if len(modelIDs) > 0 {
			return buildMarketplaceModelDefs(modelIDs, group.Platform)
		}
		if group.Platform == PlatformComposite {
			platforms := s.gatewayService.GetSchedulablePlatforms(ctx, &groupID)
			models := make([]marketplaceModelDef, 0)
			for _, platform := range []string{PlatformAnthropic, PlatformOpenAI, PlatformGemini, PlatformAntigravity, PlatformGrok} {
				if _, ok := platforms[platform]; ok {
					models = append(models, defaultMarketplaceModelDefs(platform)...)
				}
			}
			return deduplicateMarketplaceModelDefs(models)
		}
		// Accounts without explicit model mappings use the platform defaults;
		// ActiveAccountCount already guarantees that this group is schedulable.
		return defaultMarketplaceModelDefs(group.Platform)
	}

	return defaultMarketplaceModelDefs(group.Platform)
}

type marketplaceModelDef struct {
	ID          string
	DisplayName string
}

func buildMarketplaceModelDefs(modelIDs []string, platform string) []marketplaceModelDef {
	displayNames := marketplaceDisplayNameLookup(platform)
	seen := make(map[string]struct{}, len(modelIDs))
	models := make([]marketplaceModelDef, 0, len(modelIDs))

	for _, modelID := range modelIDs {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		if _, ok := seen[modelID]; ok {
			continue
		}
		seen[modelID] = struct{}{}

		models = append(models, marketplaceModelDef{
			ID:          modelID,
			DisplayName: lookupMarketplaceDisplayName(modelID, displayNames),
		})
	}

	return models
}

func defaultMarketplaceModelDefs(platform string) []marketplaceModelDef {
	switch platform {
	case PlatformOpenAI:
		models := make([]marketplaceModelDef, 0, len(openai.DefaultModels))
		for _, model := range openai.DefaultModels {
			models = append(models, marketplaceModelDef{
				ID:          model.ID,
				DisplayName: model.DisplayName,
			})
		}
		return models
	case PlatformAnthropic:
		models := make([]marketplaceModelDef, 0, len(claude.DefaultModels))
		for _, model := range claude.DefaultModels {
			models = append(models, marketplaceModelDef{
				ID:          model.ID,
				DisplayName: model.DisplayName,
			})
		}
		return models
	case PlatformGemini:
		models := make([]marketplaceModelDef, 0, len(geminicli.DefaultModels))
		for _, model := range geminicli.DefaultModels {
			models = append(models, marketplaceModelDef{
				ID:          model.ID,
				DisplayName: model.DisplayName,
			})
		}
		return models
	case PlatformAntigravity:
		defaultModels := antigravity.DefaultModels()
		models := make([]marketplaceModelDef, 0, len(defaultModels))
		for _, model := range defaultModels {
			models = append(models, marketplaceModelDef{
				ID:          model.ID,
				DisplayName: model.DisplayName,
			})
		}
		return models
	case PlatformGrok:
		defaultModels := xai.DefaultModels()
		models := make([]marketplaceModelDef, 0, len(defaultModels))
		for _, model := range defaultModels {
			models = append(models, marketplaceModelDef{ID: model.ID, DisplayName: model.DisplayName})
		}
		return models
	default:
		return nil
	}
}

func marketplaceDisplayNameLookup(platform string) map[string]string {
	switch platform {
	case PlatformOpenAI:
		out := make(map[string]string, len(openai.DefaultModels))
		for _, model := range openai.DefaultModels {
			registerMarketplaceDisplayName(out, model.ID, model.DisplayName)
		}
		return out
	case PlatformAnthropic:
		out := make(map[string]string, len(claude.DefaultModels))
		for _, model := range claude.DefaultModels {
			registerMarketplaceDisplayName(out, model.ID, model.DisplayName)
		}
		return out
	case PlatformGemini:
		out := make(map[string]string, len(geminicli.DefaultModels))
		for _, model := range geminicli.DefaultModels {
			registerMarketplaceDisplayName(out, model.ID, model.DisplayName)
		}
		return out
	case PlatformAntigravity:
		defaultModels := antigravity.DefaultModels()
		out := make(map[string]string, len(defaultModels))
		for _, model := range defaultModels {
			registerMarketplaceDisplayName(out, model.ID, model.DisplayName)
		}
		return out
	case PlatformGrok:
		defaultModels := xai.DefaultModels()
		out := make(map[string]string, len(defaultModels))
		for _, model := range defaultModels {
			registerMarketplaceDisplayName(out, model.ID, model.DisplayName)
		}
		return out
	default:
		return nil
	}
}

func deduplicateMarketplaceModelDefs(models []marketplaceModelDef) []marketplaceModelDef {
	seen := make(map[string]struct{}, len(models))
	out := make([]marketplaceModelDef, 0, len(models))
	for _, model := range models {
		if _, ok := seen[model.ID]; ok || model.ID == "" {
			continue
		}
		seen[model.ID] = struct{}{}
		out = append(out, model)
	}
	return out
}

func marketplaceDisplayBrand(platform, fallback string) string {
	switch platform {
	case PlatformAnthropic:
		return "Anthropic"
	case PlatformOpenAI:
		return "OpenAI"
	case PlatformGemini:
		return "Google"
	case PlatformGrok:
		return "xAI"
	case PlatformAntigravity:
		return "Antigravity"
	default:
		return fallback
	}
}

func lookupMarketplaceDisplayName(modelID string, displayNames map[string]string) string {
	for _, candidate := range marketplaceLookupCandidates(modelID) {
		if displayName, ok := displayNames[candidate]; ok && strings.TrimSpace(displayName) != "" {
			return displayName
		}
	}
	return modelID
}

func registerMarketplaceDisplayName(out map[string]string, modelID string, displayName string) {
	for _, key := range marketplaceLookupCandidates(modelID) {
		if _, exists := out[key]; exists {
			continue
		}
		out[key] = displayName
	}
}

func marketplaceLookupCandidates(modelID string) []string {
	candidates := []string{
		strings.TrimSpace(modelID),
		strings.TrimPrefix(strings.TrimSpace(modelID), "models/"),
	}

	trimmed := strings.TrimSpace(modelID)
	if idx := strings.LastIndex(trimmed, "/models/"); idx != -1 {
		candidates = append(candidates, trimmed[idx+len("/models/"):])
	}
	if idx := strings.LastIndex(trimmed, "/"); idx != -1 {
		candidates = append(candidates, trimmed[idx+1:])
	}

	seen := make(map[string]struct{}, len(candidates))
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	return out
}
