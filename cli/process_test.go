package cli_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/image-server/image-server/cli"
	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/paths"
	. "github.com/image-server/image-server/test"
)

const hash = "6ad5544baa6f5e852e1af26f8c2e45db"

// cliConfiguration copies wine.jpg (800×600) into a local store with no
// remote uploader and returns the original's path
func cliConfiguration(t *testing.T, source string) (*core.ServerConfiguration, string) {
	t.Helper()
	base := t.TempDir()
	sc := &core.ServerConfiguration{LocalBasePath: base, DefaultQuality: 90}
	p := &paths.Paths{LocalBasePath: base}
	sc.Adapters = &core.Adapters{Paths: p}

	original := p.LocalOriginalPath("p", hash)
	Ok(t, os.MkdirAll(filepath.Dir(original), 0700))
	data, err := os.ReadFile(source)
	Ok(t, err)
	Ok(t, os.WriteFile(original, data, 0600))
	return sc, original
}

func TestProcessCreatesOutputs(t *testing.T) {
	sc, original := cliConfiguration(t, "../test/images/wine.jpg")

	// Process prints the original's hash and size to stdout
	r, w, err := os.Pipe()
	Ok(t, err)
	stdout := os.Stdout
	os.Stdout = w
	err = cli.Process(sc, "p", []string{"w50.jpg", "c0.000_0.000_0.500_0.500.jpg"}, original)
	os.Stdout = stdout
	w.Close()
	Ok(t, err)
	printed, _ := io.ReadAll(r)
	Matches(t, "\t800\t600\n$", string(printed))

	ExpectFile(t, sc.Adapters.Paths.LocalImagePath("p", hash, "w50.jpg"))
	ExpectFile(t, sc.Adapters.Paths.LocalImagePath("p", hash, "c0.000_0.000_0.500_0.500.jpg"))
}

func TestProcessReturnsParseErrors(t *testing.T) {
	sc, original := cliConfiguration(t, "../test/images/wine.jpg")

	err := cli.Process(sc, "p", []string{"c0.390_0.000_0.050_0.500.jpg"}, original)
	Assert(t, errors.Is(err, core.ErrInvalidCrop), "expected ErrInvalidCrop, got %v", err)
}

func TestProcessReturnsProcessingErrors(t *testing.T) {
	sc, original := cliConfiguration(t, "../test/images/empty.jpg")

	err := cli.Process(sc, "p", []string{"w50.jpg"}, original)
	Assert(t, err != nil, "expected an error for an unreadable original")
}
