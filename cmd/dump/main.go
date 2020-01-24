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

func main() {
	flag.Parse()

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer r.Close()

	fields := make([]string, flag.NArg()-1)
	for i := 1; i < flag.NArg(); i++ {
		fields[i-1] = textproto.CanonicalMIMEHeaderKey(flag.Arg(i))
	}

	rs := bufio.NewReader(r)
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
		dumpMessage(m, fields)
	}
}

func dumpMessage(m mbox.Message, fields []string) {
	dumpHeader(m.Header, fields, "")
	fmt.Println()
	for i, p := range m.Parts {
		if i > 0 {
			fmt.Println()
		}
		dumpHeader(p.Header, fields, "> ")
	}
}

func dumpHeader(hdr mbox.Header, fields []string, prefix string) {
	if len(fields) == 0 {
		dumpAll(hdr, prefix)
		return
	}
	for _, f := range fields {
		for _, v := range hdr[f] {
			fmt.Printf("%s%-16s: %s\n", prefix, f, v)
		}
	}
}

func dumpAll(hdr mbox.Header, prefix string) {
	for k, vs := range hdr {
		if strings.HasPrefix(strings.ToLower(k), "x-") {
			continue
		}
		for _, v := range vs {
			fmt.Printf("%s%-16s: %s\n", prefix, k, v)
		}
	}
}
