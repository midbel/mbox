package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/midbel/mbox"
)

type Date struct {
	time.Time
}

func (d *Date) Set(str string) error {
	w, err := time.Parse("2006-01-02", str)
	if err == nil {
		d.Time = w.UTC()
	}
	return err
}

func (d *Date) String() string {
	if d.IsZero() {
		return "yyyy-mm-dd"
	}
	return d.Format("2006-02-01")
}

func main() {
	var (
		dtstart Date
		dtend   Date
	)
	flag.Var(&dtstart, "f", "from date")
	flag.Var(&dtend, "t", "to date")
	flag.Parse()

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer r.Close()

	var (
		rs   = bufio.NewReader(r)
		mail int
	)
	for {
		m, err := mbox.ReadMessage(rs)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if !keepInterval(m, dtstart, dtend) {
			continue
		}
		mail++
		fmt.Println(mail, m.Date(), m.From(), m.Subject())
	}
}

func keepInterval(m mbox.Message, f, t Date) bool {
	when := m.Date()
	if !f.IsZero() && f.After(when) {
		return false
	}
	return !t.IsZero() && t.After(when)
}
