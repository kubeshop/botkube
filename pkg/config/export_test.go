package config

func NormalizeConfigEnvName(name string) string {
	return normalizeConfigEnvName(name)
}

func GetCfgFilesToWatch(paths []string) []string {
	return getCfgFilesToWatch(paths)
}

func SortCfgFiles(paths []string) []string {
	return sortCfgFiles(paths)
}
