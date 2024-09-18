package fornax

import (
	"sync"

	"code.byted.org/flow/flow-telemetry-common/go/obtag"
	"code.byted.org/flow/flow-telemetry-common/go/obtype"
	"code.byted.org/flowdevops/fornax_sdk/utils"
)

var staticRuntime = &struct {
	sync.Once
	data *obtype.Runtime
}{}

func getStaticRuntimeTags() *obtype.Runtime {
	staticRuntime.Do(func() {
		staticRuntime.data = &obtype.Runtime{
			Language:         obtag.VLangGo,
			Library:          obtag.VLibEino,
			FornaxSDKVersion: utils.GetSdkVersion(),
		}
	})

	return staticRuntime.data
}
