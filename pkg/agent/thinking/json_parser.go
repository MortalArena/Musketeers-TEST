package thinking

import (
	"encoding/json"
	"fmt"
	"strings"
)

// JSONParser يوفر وظائف متقدمة لتحليل JSON مع error handling
type JSONParser struct {
	strictMode bool
}

// NewJSONParser ينشئ JSON parser جديد
func NewJSONParser(strictMode bool) *JSONParser {
	return &JSONParser{
		strictMode: strictMode,
	}
}

// ParseResponse يحلل استجابة LLM ويفصل JSON عن النص العادي
func (jp *JSONParser) ParseResponse(response string) (map[string]interface{}, error) {
	// البحث عن JSON في الاستجابة
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("لا يوجد JSON في الاستجابة")
	}

	jsonStr := response[jsonStart : jsonEnd+1]
	var result map[string]interface{}

	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, fmt.Errorf("فشل تحليل JSON: %w", err)
	}

	return result, nil
}

// ParseWithSchema يحلل JSON مع التحقق من schema
func (jp *JSONParser) ParseWithSchema(response string, schema map[string]interface{}) (map[string]interface{}, error) {
	result, err := jp.ParseResponse(response)
	if err != nil {
		return nil, err
	}

	if jp.strictMode {
		err = jp.ValidateSchema(result, schema)
		if err != nil {
			return nil, fmt.Errorf("فشل التحقق من schema: %w", err)
		}
	}

	return result, nil
}

// ValidateSchema يتحقق من أن النتيجة تطابق schema المطلوب
func (jp *JSONParser) ValidateSchema(result, schema map[string]interface{}) error {
	for key, expectedType := range schema {
		if _, exists := result[key]; !exists {
			return fmt.Errorf("المفتاح %s مفقود في النتيجة", key)
		}

		// التحقق من النوع (بسيط)
		switch expectedType.(type) {
		case string:
			if _, ok := result[key].(string); !ok {
				return fmt.Errorf("المفتاح %s يجب أن يكون string", key)
			}
		case int, int64:
			if _, ok := result[key].(float64); !ok {
				return fmt.Errorf("المفتاح %s يجب أن يكون number", key)
			}
		case bool:
			if _, ok := result[key].(bool); !ok {
				return fmt.Errorf("المفتاح %s يجب أن يكون bool", key)
			}
		case []interface{}:
			if _, ok := result[key].([]interface{}); !ok {
				return fmt.Errorf("المفتاح %s يجب أن يكون array", key)
			}
		case map[string]interface{}:
			if _, ok := result[key].(map[string]interface{}); !ok {
				return fmt.Errorf("المفتاح %s يجب أن يكون object", key)
			}
		}
	}

	return nil
}

// ParseStringArray يحلل array من strings
func (jp *JSONParser) ParseStringArray(jsonStr string) ([]string, error) {
	var result []string
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, fmt.Errorf("فشل تحليل string array: %w", err)
	}
	return result, nil
}

// ParseStringMap يحلل map من strings
func (jp *JSONParser) ParseStringMap(jsonStr string) (map[string]string, error) {
	var result map[string]string
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, fmt.Errorf("فشل تحليل string map: %w", err)
	}
	return result, nil
}

// ExtractJSONCodeBlocks يستخرج JSON من code blocks في Markdown
func (jp *JSONParser) ExtractJSONCodeBlocks(response string) (string, error) {
	// البحث عن ```json ... ```
	startMarker := "```json"
	endMarker := "```"

	startIdx := strings.Index(response, startMarker)
	if startIdx == -1 {
		// محاولة البحث عن ``` فقط
		startMarker = "```"
		startIdx = strings.Index(response, startMarker)
		if startIdx == -1 {
			return "", fmt.Errorf("لا يوجد code block")
		}
	}

	startIdx += len(startMarker)
	endIdx := strings.Index(response[startIdx:], endMarker)
	if endIdx == -1 {
		return "", fmt.Errorf("code block غير مكتمل")
	}

	jsonStr := strings.TrimSpace(response[startIdx : startIdx+endIdx])
	return jsonStr, nil
}

// SafeParse يحلل JSON بأمان مع fallback
func (jp *JSONParser) SafeParse(response string, fallback map[string]interface{}) map[string]interface{} {
	result, err := jp.ParseResponse(response)
	if err != nil {
		return fallback
	}
	return result
}

// GetField يحصل على قيمة حقل من JSON
func (jp *JSONParser) GetField(data map[string]interface{}, field string, defaultValue interface{}) interface{} {
	if value, exists := data[field]; exists {
		return value
	}
	return defaultValue
}

// GetStringField يحصل على قيمة string من JSON
func (jp *JSONParser) GetStringField(data map[string]interface{}, field string, defaultValue string) string {
	if value, exists := data[field]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// GetFloatField يحصل على قيمة float من JSON
func (jp *JSONParser) GetFloatField(data map[string]interface{}, field string, defaultValue float64) float64 {
	if value, exists := data[field]; exists {
		if num, ok := value.(float64); ok {
			return num
		}
	}
	return defaultValue
}

// GetBoolField يحصل على قيمة bool من JSON
func (jp *JSONParser) GetBoolField(data map[string]interface{}, field string, defaultValue bool) bool {
	if value, exists := data[field]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// GetArrayField يحصل على array من JSON
func (jp *JSONParser) GetArrayField(data map[string]interface{}, field string) []interface{} {
	if value, exists := data[field]; exists {
		if arr, ok := value.([]interface{}); ok {
			return arr
		}
	}
	return []interface{}{}
}

// GetStringArrayField يحصل على array من strings من JSON
func (jp *JSONParser) GetStringArrayField(data map[string]interface{}, field string) []string {
	arr := jp.GetArrayField(data, field)
	result := []string{}
	for _, item := range arr {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}
