package service

import (
	"encoding/json"
	"errors"
	"strings"

	"AIM/internal/model"

	"gorm.io/datatypes"
)

func normalizeAIBotAPISource(source, baseURL string) string {
	source = strings.TrimSpace(source)
	switch source {
	case model.AIBotAPISourceSystem, model.AIBotAPISourceThirdParty:
		return source
	case "":
		if strings.TrimSpace(baseURL) != "" {
			return model.AIBotAPISourceThirdParty
		}
		return model.AIBotAPISourceSystem
	default:
		return source
	}
}

func isValidAIBotAPISource(source string) bool {
	return source == model.AIBotAPISourceSystem || source == model.AIBotAPISourceThirdParty
}

func normalizePluginConfig(raw json.RawMessage) (datatypes.JSON, error) {
	rawText := strings.TrimSpace(string(raw))
	if rawText == "" || rawText == "null" {
		return nil, nil
	}
	if !json.Valid(raw) {
		return nil, errors.New("插件配置必须是合法 JSON")
	}
	return datatypes.JSON(append([]byte(nil), raw...)), nil
}

func uniqueUintIDs(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func valueOrDefault(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
