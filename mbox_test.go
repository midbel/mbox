package mbox

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

const (
	defaultSubject = "mbox test"
	defaultFrom    = "midbel@foobar.org"
)

type TestCase struct {
	File        string
	Mime        bool
	Multipart   bool
	Reply       bool
	Parts       int
	Attachments int
}

func TestReadMessage(t *testing.T) {
	emails := []TestCase{
		{
			File:        "simple.txt",
			Multipart:   false,
			Reply:       false,
			Parts:       1,
			Attachments: 0,
		},
		{
			File:        "alternative.txt",
			Multipart:   true,
			Reply:       false,
			Parts:       2,
			Attachments: 0,
		},
		{
			File:        "mixed.txt",
			Multipart:   true,
			Reply:       false,
			Parts:       2,
			Attachments: 1,
		},
		{
			File:        "mixedalt.txt",
			Multipart:   true,
			Reply:       false,
			Parts:       3,
			Attachments: 1,
		},
		{
			File:        "reply.txt",
			Multipart:   false,
			Reply:       true,
			Parts:       1,
			Attachments: 0,
		},
	}
	for _, e := range emails {
		if err := testReadMessage(e); err != nil {
			t.Errorf("%s: fail to parse mbox; %s", e.File, err)
		}
	}
}

func testReadMessage(tc TestCase) error {
	r, err := os.Open(filepath.Join("testdata", tc.File))
	if err != nil {
		return err
	}
	defer r.Close()

	m, err := ReadMessage(bufio.NewReader(r))
	if err != nil {
		return err
	}
	if got := len(m.Parts); got != tc.Parts {
		return fmt.Errorf("wrong number of part! want %d, got %d", tc.Parts, got)
	}
	if got := m.IsMultipart(); got != tc.Multipart {
		return fmt.Errorf("message should be multipart")
	}
	if got := m.IsReply(); got != tc.Reply {
		return fmt.Errorf("message should be a reply")
	}
	if files := m.Files(); len(files) != tc.Attachments {
		return fmt.Errorf("wrong number of attachment! want %d, got %d", tc.Attachments, len(files))
	}
	if got := m.Subject(); got != defaultSubject {
		return fmt.Errorf("wrong subject header! want %s, got %s", got, defaultSubject)
	}
	if got := m.From(); got != defaultFrom {
		return fmt.Errorf("wrong from header! want %s, got %s", got, defaultFrom)
	}
	return nil
}
