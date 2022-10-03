package config

func NormalizeConfigEnvName(name string) string {
	return normalizeConfigEnvName(name)
}

func SortCfgFiles(paths []string) []string {
	return sortCfgFiles(paths)
}
