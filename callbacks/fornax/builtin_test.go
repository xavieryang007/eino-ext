package fornax

import (
	"runtime/debug"
	"testing"

	"github.com/bytedance/mockey"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_ReadBuildVersion(t *testing.T) {
	t.Run("read go mod import", func(t *testing.T) {
		Convey("import without replace", t, func() {
			mockey.Mock(debug.ReadBuildInfo).
				Return(
					&debug.BuildInfo{
						Deps: []*debug.Module{
							{
								Path:    "code.byted.org/flow/eino",
								Version: "v1.0.0",
							},
						},
					}, true).
				Build()
			v := ReadBuildVersion(einoImportPath)
			So(v, ShouldEqual, "v1.0.0")
		})
	})

}
