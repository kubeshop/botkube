package frontmatter

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	frontmatterFormat = `---
title: %s
---

`
)

// FilePrepender is a function which is used for custom formatting while generating markdown documentation.
func FilePrepender(filePath string) string {
	fileName := filepath.Base(filePath)
	fileNameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	title := strings.Replace(fileNameWithoutExt, "_", " ", -1)
	return fmt.Sprintf(frontmatterFormat, title)
}
