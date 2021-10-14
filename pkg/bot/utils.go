// Copyright (c) 2022 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
