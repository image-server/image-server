package server_test

import (
	"net/http"
	"testing"

	"github.com/image-server/image-server/processor/cli"
	"github.com/image-server/image-server/processor/vips"
	"github.com/image-server/image-server/server"
)

func TestProbesNeedOnlyOneProcessor(t *testing.T) {
	vipsAvailable, cliAvailable := vips.Available, cli.Available
	t.Cleanup(func() { vips.Available, cli.Available = vipsAvailable, cliAvailable })
	h := server.NewAdminServerHandler()

	cases := []struct {
		name      string
		vips, cli bool
		want      int
	}{
		{"libvips only", true, false, http.StatusOK},
		{"ImageMagick only", false, true, http.StatusOK},
		{"neither", false, false, http.StatusNotImplemented},
	}
	for _, c := range cases {
		vips.Available, cli.Available = c.vips, c.cli
		for _, path := range []string{"/probe/ready", "/probe/live"} {
			if got := requestStatus(h, "GET", path); got != c.want {
				t.Errorf("%s: %s = %d, want %d", c.name, path, got, c.want)
			}
		}
	}
}
