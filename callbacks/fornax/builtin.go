package fornax

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"

	"code.byted.org/flow/flow-telemetry-common/go/obtag"
	"code.byted.org/flow/flow-telemetry-common/go/obtype"
	"code.byted.org/flowdevops/fornax_sdk/utils"
	"code.byted.org/gopkg/logs/v2"
	"code.byted.org/tiktok/buildinfo/deps"
)

func init() {
	mustInitEinoSdkVersion()
}

var staticRuntime = &struct {
	sync.Once
	data *obtype.Runtime
}{}

func getStaticRuntimeTags() *obtype.Runtime {
	staticRuntime.Do(func() {
		data := &obtype.Runtime{
			Language:         obtag.VLangGo,
			Library:          obtag.VLibEino,
			FornaxSDKVersion: utils.GetSdkVersion(),
			EinoVersion:      getEinoSdkVersion(),
		}
		err := fillSCMVersion(data)
		if err != nil {
			logs.Notice("load scm version failed, err = %v", err)
		}
		staticRuntime.data = data
	})

	return staticRuntime.data
}

func fillSCMVersion(rt *obtype.Runtime) error {
	appDir := getAppDir()
	if appDir == "" {
		return nil
	}
	filename := filepath.Join(appDir, "current_revision")
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	scan := bufio.NewScanner(bytes.NewReader(data))
	for scan.Scan() {
		line := scan.Text()
		idx := strings.Index(line, ":")
		if idx < 0 || idx == len(line)-1 {
			continue
		}

		key, value := line[:idx], line[idx+1:]
		value = strings.TrimSpace(value)
		switch key {
		case "revision":
			rt.SCMRevision = value
		case "version":
			rt.SCMVersion = value
		case "repo name":
			rt.SCMRepo = value
		default:
			// Ignore this line.
		}
	}
	return nil
}

func getAppDir() (appDir string) {
	if appDir = os.Getenv("APP_DIR"); appDir != "" {
		return appDir
	}

	if len(os.Args) == 0 {
		return ""
	}

	bin := os.Args[0]
	if !filepath.IsAbs(bin) {
		wd, err := os.Getwd()
		if err != nil {
			return ""
		}
		bin = filepath.Clean(filepath.Join(wd, bin))
	}

	binDir := filepath.Dir(bin)
	return strings.TrimSuffix(binDir, "/bin")
}

var einoSdkVersion string

func getEinoSdkVersion() string {
	return einoSdkVersion
}

func mustInitEinoSdkVersion() {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok { // try bazel
		buildDeps, ok := deps.ReadBuildInfo()
		if !ok {
			logs.Notice("failed to read build info")
			return
		}
		for _, dep := range buildDeps {
			if dep.ImportPath == "code.byted.org/flow/eino" {
				einoSdkVersion = dep.Version
				return
			}
		}

		logs.Notice("flow/eino build info not found")
		return
	}

	for _, dep := range buildInfo.Deps {
		if dep.Path == "code.byted.org/flow/eino" {
			if dep.Replace != nil {
				einoSdkVersion = dep.Replace.Version
			} else {
				einoSdkVersion = dep.Version
			}

			return
		}
	}
}
