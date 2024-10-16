package fornax

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bytedance/mockey"
	. "github.com/smartystreets/goconvey/convey"

	"code.byted.org/flow/flow-telemetry-common/go/obtype"
)

func Test_loadSCMVersion(t *testing.T) {
	mockEnv := func(t *testing.T, k, v string) {
		origin := os.Getenv(k)
		if err := os.Setenv(k, v); err != nil {
			t.Skip()
		}
		t.Cleanup(func() { _ = os.Setenv(k, origin) })
	}

	data := []byte(`revision:fbd121f8ca5fc90d4109a23122c8f83729f6cedc
version:1.0.0.21
pub date:2024-10-10 12:27:33
arch:x86_64
region:cn
repo name:flow/devops/eino_app`)

	t.Run("read APP_DIR env", func(t *testing.T) {
		dir := t.TempDir()
		rt := &obtype.Runtime{}
		mockEnv(t, "APP_DIR", dir)
		Convey("", t, func() {
			err := os.WriteFile(filepath.Join(dir, "current_revision"), data, 0644)
			So(err, ShouldBeNil)
			err = fillSCMVersion(rt)
			So(err, ShouldBeNil)
			So(rt.SCMRepo, ShouldEqual, `flow/devops/eino_app`)
			So(rt.SCMVersion, ShouldEqual, `1.0.0.21`)
			So(rt.SCMRevision, ShouldEqual, `fbd121f8ca5fc90d4109a23122c8f83729f6cedc`)
		})
	})

	t.Run("use output directory", func(t *testing.T) {
		dir := t.TempDir()
		args := os.Args
		t.Cleanup(func() { os.Args = args })

		Convey("", t, func() {
			mockey.Mock(os.Getwd).Return(dir, nil).Build()
			err := os.WriteFile(filepath.Join(dir, "current_revision"), data, 0644)
			So(err, ShouldBeNil)
			err = os.MkdirAll(filepath.Join(dir, "bin"), 0755)
			So(err, ShouldBeNil)
			os.Args = []string{"./bin/some_binary"}
			rt := &obtype.Runtime{}
			err = fillSCMVersion(rt)
			So(err, ShouldBeNil)
			So(rt.SCMRepo, ShouldEqual, `flow/devops/eino_app`)
			So(rt.SCMVersion, ShouldEqual, `1.0.0.21`)
			So(rt.SCMRevision, ShouldEqual, `fbd121f8ca5fc90d4109a23122c8f83729f6cedc`)
		})
	})

	t.Run("no revision file", func(t *testing.T) {
		dir := t.TempDir()
		mockEnv(t, "APP_DIR", dir)

		Convey("", t, func() {
			rt := &obtype.Runtime{}
			err := fillSCMVersion(rt)
			So(err, ShouldNotBeNil)
		})
	})
}
