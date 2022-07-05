package bot

import (
	"fmt"
	"strings"
)

func formatCodeBlock(msg string) string {
	return fmt.Sprintf("```\n%s\n```", strings.TrimSpace(msg))
}

//accessAndTypeCastToMap access and typecast of map object
func accessAndTypeCastToMap(key string, m map[string]interface{}) (map[string]interface{}, error) {
	value, ok := m[key]
	if !ok {
		return nil, fmt.Errorf("Missing key %s in the map", key)
	}
	newMap, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Failed to convert into map[string]interface{}")
	}
	return newMap, nil
}

//accessAndTypeCastToString access and typecast of string object
func accessAndTypeCastToString(key string, m map[string]interface{}) (string, error) {
	value, ok := m[key]
	if !ok {
		return "", fmt.Errorf("Missing key %s in the string", key)
	}
	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("Failed to convert into string")
	}
	return str, nil
}
