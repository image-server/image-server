package cli

import (
	"fmt"
	"os"

	"github.com/image-server/image-server/core"
	"github.com/image-server/image-server/info"
)

// Item represents all image properties needed for the result of the processing
type Item struct {
	Hash   string
	URL    string
	Width  int
	Height int
}

// ToTabDelimited creates a tab delimited text representation of an Item
func (i Item) ToTabDelimited() string {
	return fmt.Sprintf("%s\t%s\t%d\t%d\n", i.Hash, i.URL, i.Width, i.Height)
}

func Process(sc *core.ServerConfiguration, namespace string, outputs []string, path string) error {

	processor := NewImageProcessor(namespace, path, outputs)
	err := processor.ProcessMissing(sc)
	if err != nil {
		return err
	}

	i := info.Info{Path: path}
	id, err := i.ImageDetails()
	if err != nil {
		return err
	}

	item := Item{Hash: id.Hash, Width: id.Width, Height: id.Height}
	fmt.Fprint(os.Stdout, item.ToTabDelimited())

	return nil
}
