package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/textproto"
	"os"
	"strings"

	"github.com/midbel/mbox"
)

var groups = map[string][]string{
	"date":           {"date"},
	"sender":         {"from", "sender", "reply-to"},
	"recipient":      {"to", "cc", "bcc"},
	"identification": {"message-id", "in-reply-to", "references"},
	"information":    {"subject", "comments", "keywords"},
	"trace":          {"return ", "path", "received", "user-agent"},
	"content":        {"content-type", "content-length", "content-disposition", "content-transfer-encoding", "content-language"},
}

func main() {
	extended := flag.Bool("x", false, "include extended headers")
	flag.Parse()

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer r.Close()

	var (
		rs = bufio.NewReader(r)
		fs = listFields(flag.Args())
	)

	for i := 0; ; i++ {
		if i > 0 {
			fmt.Println("---")
		}
		m, err := mbox.ReadMessage(rs)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(m.Get("message-id"))
		dumpMessage(m, fs, *extended)
	}
}

func listFields(args []string) []string {
	var (
		seen = make(map[string]struct{})
		fields = make([]string, 0, len(args))
	)
	for i := 1; i < len(args); i++ {
		field := args[i]
		if _, ok := seen[field]; ok {
			continue
		}
		if vs, ok := groups[strings.ToLower(field)]; ok {
			for _, v := range vs {
				if _, ok := seen[v]; ok {
					continue
				}
				fields = append(fields, textproto.CanonicalMIMEHeaderKey(v))
				seen[v] = struct{}{}
			}
		} else {
			fields = append(fields, textproto.CanonicalMIMEHeaderKey(field))
		}
		seen[field] = struct{}{}
	}
	return fields
}

func dumpMessage(m mbox.Message, fields []string, extended bool) {
	dumpHeader(m.Header, fields, extended, "")
	if len(m.Parts) == 0 {
		return
	}
	for _, p := range m.Parts {
		if len(p.Header) == 0 {
			continue
		}
		dumpHeader(p.Header, fields, extended, "> ")
	}
}

func dumpHeader(hdr mbox.Header, fields []string, extended bool, prefix string) {
	if len(fields) == 0 {
		dumpAll(hdr, extended, prefix)
		return
	}
	for _, f := range fields {
		vs := hdr[f]
		if len(vs) == 0 {
			continue
		}
		for _, v := range vs {
			if v == "" {
				continue
			}
			fmt.Printf("%s%-16s: %s\n", prefix, f, v)
		}
	}
}

func dumpAll(hdr mbox.Header, extended bool, prefix string) {
	for k, vs := range hdr {
		if !extended && strings.HasPrefix(strings.ToLower(k), "x-") {
			continue
		}
		for _, v := range vs {
			if v == "" {
				continue
			}
			fmt.Printf("%s%-16s: %s\n", prefix, k, v)
		}
	}
}
