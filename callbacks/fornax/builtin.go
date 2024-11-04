package fornax

import (
	"runtime/debug"

	"code.byted.org/gopkg/logs/v2"
	"code.byted.org/tiktok/buildinfo/deps"
)

func init() {
	mustInitEinoSdkVersion()
}

var einoSdkVersion string

func getEinoSdkVersion() string {
	return einoSdkVersion
}

func mustInitEinoSdkVersion() {
	einoSdkVersion = ReadBuildVersion("code.byted.org/flow/eino")
}

func ReadBuildVersion(path string) string {
	if v, ok := readVersionByGoMod(path); ok {
		return v
	}

	if v, ok := readVersionByBazel(path); ok {
		return v
	}

	return "unknown_build_info"
}

func readVersionByGoMod(path string) (string, bool) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}

	for _, dep := range buildInfo.Deps {
		if dep.Path == path {
			if dep.Replace != nil {
				return dep.Replace.Version, true
			} else {
				return dep.Version, true
			}
		}
	}

	logs.Warn("failed to read build info by go mod")
	return "", false
}

func readVersionByBazel(path string) (string, bool) {
	buildDeps, ok := deps.ReadBuildInfo()
	if !ok {
		return "", false
	}

	for _, dep := range buildDeps {
		if dep.ImportPath == path {
			return dep.Version, true
		}
	}
	logs.Warn("failed to read build info by bazel")
	return "", false
}
