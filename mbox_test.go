package mbox

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestReadMessage(t *testing.T) {
	emails := []string{
		"simple",
		"alternative",
		"mixed",
		"mixedalt",
		"reply",
	}
	for _, e := range emails {
		if err := testReadMessage(e + ".txt"); err != nil {
			t.Errorf("%s: fail to parse mbox; %s", e, err)
		}
	}
}

func testReadMessage(file string) error {
	r, err := os.Open(filepath.Join("testdata", file))
	if err != nil {
		return err
	}
	defer r.Close()

	m, err := ReadMessage(bufio.NewReader(r))
	if err != nil {
		return err
	}
	if len(m.Parts) == 0 {
		err = fmt.Errorf("message has zero part! it should have at least one")
	}
	return err
}
