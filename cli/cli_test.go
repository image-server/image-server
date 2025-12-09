package cli

import (
	"testing"

	. "github.com/image-server/image-server/test"
)

func TestItemToTabDelimited(t *testing.T) {
	item := Item{"6ad5544baa6f5e852e1af26f8c2e45db", "http://example.com/image.jpg", 40, 30}
	expected := "6ad5544baa6f5e852e1af26f8c2e45db\thttp://example.com/image.jpg\t40\t30\n"
	Equals(t, expected, item.ToTabDelimited())
}
