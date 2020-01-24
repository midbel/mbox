package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/midbel/mbox"
)

type Date struct {
	time.Time
}

var patterns = []string{
	"2006-01-02",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"02-01-2006",
	"02/01/2006",
	"2006/002",
}

func (d *Date) Set(str string) error {
	var (
		when time.Time
		err  error
	)
	for _, p := range patterns {
		when, err = time.Parse(p, str)
		if err == nil {
			d.Time = when.UTC()
			break
		}
	}
	return err
}

func (d *Date) String() string {
	if d.IsZero() {
		return "yyyy-mm-dd"
	}
	return d.Format("2006-02-01")
}

type FilterFunc func(mbox.Message) bool

func main() {
	files, keep := parseArgs()

	rs := make([]io.Reader, len(files))
	for i := 0; i < len(files); i++ {
		r, err := os.Open(files[i])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		defer r.Close()
		rs[i] = r
	}

	var (
		r    = bufio.NewReader(io.MultiReader(rs...))
		mail int
	)
	for {
		m, err := mbox.ReadMessage(r)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if !keep(m) {
			continue
		}
		mail++
		var (
			attach = m.Files()
			when   = m.Date().Format("2006-01-02 15:04:05")
		)
		fmt.Printf("%4d | %s | %32s | %3d | %s\n", mail, when, m.From(), len(attach), m.Subject())
	}
}

func parseArgs() ([]string, FilterFunc) {
	var (
		dtstart  Date
		dtend    Date
		noreply  = flag.Bool("no-reply", false, "only e-mails that are not replies")
		attached = flag.Bool("with-attachment", false, "only e-mails that have attachments")
		subject  = flag.String("subject", "", "only e-mails with given subject")
		faddr    = flag.String("from", "", "only e-mails from given address")
		taddr    = flag.String("to", "", "only e-mails to given address")
	)
	flag.Var(&dtstart, "starts", "only e-mails after given date")
	flag.Var(&dtend, "ends", "only e-mails before given date")
	flag.Parse()

	filters := []FilterFunc{
		filterInterval(dtstart.Time, dtend.Time),
		filterFromAddr(*faddr),
		filterToAddr(*taddr),
		filterSubject(*subject),
		filterReply(*noreply),
		filterAttachments(*attached),
	}

	return flag.Args(), keepMessage(filters...)
}

func keepMessage(filters ...FilterFunc) FilterFunc {
	return func(m mbox.Message) bool {
		for _, fn := range filters {
			if !fn(m) {
				return false
			}
		}
		return true
	}
}

func filterFromAddr(from string) FilterFunc {
	return func(m mbox.Message) bool {
		return from == "" || m.From() == from
	}
}

func filterToAddr(to string) FilterFunc {
	return func(m mbox.Message) bool {
		list := m.To()
		sort.Strings(list)
		i := sort.SearchStrings(list, to)
		return to == "" || (i < len(list) && list[i] == to)
	}
}

func filterSubject(subj string) FilterFunc {
	return func(m mbox.Message) bool {
		return subj == "" || strings.Contains(m.Subject(), subj)
	}
}

func filterReply(reply bool) FilterFunc {
	return func(m mbox.Message) bool {
		return !reply || m.IsReply()
	}
}

func filterAttachments(attached bool) FilterFunc {
	return func(m mbox.Message) bool {
		return !attached || len(m.Files()) > 0
	}
}

func filterInterval(fd, td time.Time) FilterFunc {
	return func(m mbox.Message) bool {
		if fd.IsZero() && td.IsZero() {
			return true
		}
		when := m.Date()
		if !fd.IsZero() && fd.After(when) {
			return false
		}
		return !td.IsZero() && td.After(when)
	}
}
