package sigma

import (
	"strings"

	"gopkg.in/yaml.v3"
)

type SigmaRule struct {
	Title       string                 `yaml:"title" json:"title"`
	ID          string                 `yaml:"id" json:"id"`
	Status      string                 `yaml:"status" json:"status"`
	Description string                 `yaml:"description" json:"description"`
	Level       string                 `yaml:"level" json:"level"`
	Logsource   Logsource              `yaml:"logsource" json:"logsource"`
	Detection   map[string]interface{} `yaml:"detection" json:"detection"`
	Tags        []string               `yaml:"tags" json:"tags"`
}

type Logsource struct {
	Category string `yaml:"category" json:"category"`
	Product  string `yaml:"product" json:"product"`
	Service  string `yaml:"service" json:"service"`
}

func ParseRule(yamlData []byte) (*SigmaRule, error) {
	var rule SigmaRule
	err := yaml.Unmarshal(yamlData, &rule)
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *SigmaRule) Evaluate(logText string) bool {
	logTextLower := strings.ToLower(logText)

	conditionStr, _ := r.Detection["condition"].(string)
	if conditionStr == "" {
		conditionStr = "selection"
	}

	selections := make(map[string]bool)
	for key, val := range r.Detection {
		if key == "condition" {
			continue
		}
		selections[key] = r.matchSelection(val, logTextLower)
	}

	return evaluateCondition(conditionStr, selections)
}

func (r *SigmaRule) matchSelection(val interface{}, logTextLower string) bool {
	switch v := val.(type) {
	case []interface{}:
		for _, item := range v {
			if strVal, ok := item.(string); ok {
				if strings.Contains(logTextLower, strings.ToLower(strVal)) {
					return true
				}
			}
		}
	case string:
		return strings.Contains(logTextLower, strings.ToLower(v))
	case map[string]interface{}:
		for _, subVal := range v {
			if r.matchSelection(subVal, logTextLower) {
				return true
			}
		}
	}
	return false
}

func evaluateCondition(cond string, selections map[string]bool) bool {
	cond = strings.TrimSpace(cond)
	cond = strings.ToLower(cond)

	if strings.Contains(cond, " or ") {
		parts := strings.SplitN(cond, " or ", 2)
		return evaluateCondition(parts[0], selections) || evaluateCondition(parts[1], selections)
	}

	if strings.Contains(cond, " and ") {
		parts := strings.SplitN(cond, " and ", 2)
		return evaluateCondition(parts[0], selections) && evaluateCondition(parts[1], selections)
	}

	if strings.HasPrefix(cond, "not ") {
		rest := strings.TrimSpace(strings.TrimPrefix(cond, "not "))
		return !evaluateCondition(rest, selections)
	}

	if strings.HasPrefix(cond, "1 of selection_") {
		for key, matched := range selections {
			if strings.HasPrefix(key, "selection_") && matched {
				return true
			}
		}
		return false
	}

	if strings.HasPrefix(cond, "all of selection_") {
		for key, matched := range selections {
			if strings.HasPrefix(key, "selection_") && !matched {
				return false
			}
		}
		return true
	}

	if strings.HasPrefix(cond, "1 of ") {
		prefix := strings.TrimPrefix(cond, "1 of ")
		if matched, ok := selections[prefix]; ok {
			return matched
		}
		return false
	}

	if strings.HasPrefix(cond, "all of ") {
		prefix := strings.TrimPrefix(cond, "all of ")
		if matched, ok := selections[prefix]; ok {
			return matched
		}
		return false
	}

	if matched, ok := selections[cond]; ok {
		return matched
	}

	for _, matched := range selections {
		if matched {
			return true
		}
	}
	return false
}
