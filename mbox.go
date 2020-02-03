package mbox

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime/quotedprintable"
	"net/textproto"
	"strings"
	"time"

	"github.com/midbel/mime"
)

const (
	fromLinePrefix = "From "

	hdrMimeVersion     = "MIME-Version"
	hdrContentType     = "Content-Type"
	hdrContentLength   = "Content-Length"
	hdrContentDispo    = "Content-Disposition"
	hdrContentEncoding = "Content-Transfer-Encoding"

	hdrDate       = "date"
	hdrFrom       = "from"
	hdrTo         = "to"
	hdrCc         = "cc"
	hdrSubject    = "subject"
	hdrInReplyTo  = "in-reply-to"
	hdrReferences = "references"

	encBit7   = "7bit"
	encBit8   = "8bit"
	encBase64 = "base64"
	encQuoted = "quoted-printable"

	multiPart  = "multipart"
	multiMixed = "mixed"
	multiAlt   = "alternative"
	multiBound = "boundary"
)

type Message struct {
	Header
	Parts []Part
}

func ReadMessage(rs *bufio.Reader) (Message, error) {
	var m Message
	for {
		line, err := rs.ReadString('\n')
		if err == io.EOF {
			return m, err
		}
		if line = strings.TrimSpace(line); line == "" {
			continue
		}
		if !strings.HasPrefix(line, fromLinePrefix) {
			return m, fmt.Errorf("expected From Line. Got %s", line)
		}
		break
	}
	hdr, err := readHeader(rs)
	if err != nil {
		return m, err
	}
	m.Header = hdr

	if !m.IsMultipart() {
		return m, readPlain(rs, &m)
	}
	mt, err := mime.Parse(m.Get(hdrContentType))
	if err != nil {
		return m, err
	}
	ps, err := readBody(rs, []byte("--"+mt.Params[multiBound]), nil)
	if err == nil {
		m.Parts = append(m.Parts, ps...)
	}
	return m, nil
}

func (m Message) Filter(fn func(Header) bool) []Part {
	as := make([]Part, 0, len(m.Parts))
	for _, p := range m.Parts {
		if fn(p.Header) {
			as = append(as, p)
		}
	}
	return as
}

func (m Message) Files() []string {
	files := make([]string, 0, len(m.Parts))
	for _, p := range m.Parts {
		if file := p.Filename(); file != "" {
			files = append(files, file)
		}
	}
	return files
}

func (m Message) Date() time.Time {
	return parseTime(m.Get(hdrDate)).UTC()
}

func (m Message) Subject() string {
	return m.Get(hdrSubject)
}

func (m Message) From() string {
	return parseAddress(m.Get(hdrFrom))
}

func (m Message) To() []string {
	return parseAddressList(m.Get(hdrTo))
}

func (m Message) Cc() []string {
	return parseAddressList(m.Get(hdrCc))
}

func (m Message) IsMime() bool {
	return m.Has(hdrMimeVersion)
}

func (m Message) IsMultipart() bool {
	if !m.IsMime() {
		return false
	}
	mt, _ := mime.Parse(m.Get(hdrContentType))
	return mt.MainType == multiPart
}

func (m Message) IsReply() bool {
	return m.Has(hdrInReplyTo)
}

type Part struct {
	Header
	Body []byte
}

func (p Part) Text() []byte {
	mt, err := mime.Parse(p.Get(hdrContentType))
	if err != nil {
		return nil
	}
	var str []byte
	if mt.MainType == "text" && mt.SubType == "plain" {
		str = p.decodeBody()
	}
	return str
}

func (p Part) HTML() []byte {
	mt, err := mime.Parse(p.Get(hdrContentType))
	if err != nil {
		return nil
	}
	var str []byte
	if mt.MainType == "text" && mt.SubType == "html" {
		str = p.decodeBody()
	}
	return str
}

func (p Part) Bytes() []byte {
	return p.decodeBody()
}

func (p Part) Filename() string {
	hdr, ps := parseValueField(p.Get(hdrContentDispo))
	if hdr == "attachment" || hdr == "inline" {
		hdr = ps["filename"]
		if hdr == "" {
			mt, err := mime.Parse(p.Get(hdrContentType))
			if err == nil {
				hdr = mt.Params["name"]
			}
		}
	} else {
		hdr = ""
	}
	return hdr
}

func (p Part) IsAttachment() bool {
	hdr, _ := parseValueField(p.Get(hdrContentDispo))
	return hdr == "attachment"
}

func (p Part) IsInline() bool {
	hdr, _ := parseValueField(p.Get(hdrContentDispo))
	return hdr == "inline"
}

func (p Part) IsMultipart() bool {
	mt, _ := mime.Parse(p.Get(hdrContentType))
	return mt.MainType == multiPart
}

func (p Part) decodeBody() []byte {
	var rs io.Reader
	switch enc := p.Get(hdrContentEncoding); strings.ToLower(enc) {
	case encBase64:
		var (
			scan = bufio.NewScanner(bytes.NewReader(p.Body))
			ws   bytes.Buffer
		)
		for scan.Scan() {
			ws.Write(scan.Bytes())
		}
		if err := scan.Err(); err != nil {
			return nil
		}
		rs = base64.NewDecoder(base64.StdEncoding, &ws)
	case encQuoted:
		rs = quotedprintable.NewReader(bytes.NewReader(p.Body))
	default:
		return p.Body
	}
	body, _ := ioutil.ReadAll(rs)
	return body
}

type Header map[string][]string

func (h Header) Has(k string) bool {
	k = textproto.CanonicalMIMEHeaderKey(k)
	_, ok := h[k]
	return ok
}

func (h Header) Add(k, v string) {
	k = textproto.CanonicalMIMEHeaderKey(k)
	h[k] = append(h[k], strings.TrimSpace(v))
}

func (h Header) Get(k string) string {
	k = textproto.CanonicalMIMEHeaderKey(k)
	if vs := h[k]; len(vs) > 0 {
		k = vs[len(vs)-1]
	} else {
		k = ""
	}
	return k
}

func (h Header) Split(k string) (string, map[string]string) {
	k = textproto.CanonicalMIMEHeaderKey(k)
	vs, ok := h[k]
	if !ok || len(vs) != 1 {
		return "", nil
	}
	return parseValueField(vs[0])
}

func (h Header) Set(k, v string) {
	k = textproto.CanonicalMIMEHeaderKey(k)
	if len(h[k]) > 0 {
		h[k] = h[k][:0]
	}
	h.Add(k, v)
}

func (h Header) Del(k string) {
	k = textproto.CanonicalMIMEHeaderKey(k)
	delete(h, k)
}

func readBody(rs *bufio.Reader, boundary, parent []byte) ([]Part, error) {
	if bytes.Equal(boundary, []byte("--")) {
		return nil, fmt.Errorf("empty boundary delimiter")
	}

	if err := skipProlog(rs, boundary); err != nil {
		return nil, err
	}
	var ps []Part
	for {
		xs, err := readPart(rs, boundary, parent)
		if err == nil || err == io.EOF {
			ps = append(ps, xs...)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}
	return ps, skipEpilog(rs, parent)
}

func readPart(rs *bufio.Reader, boundary, parent []byte) ([]Part, error) {
	var (
		part Part
		err  error
		str  []byte
		line []byte
	)
	if part.Header, err = readHeader(rs); err != nil {
		return nil, err
	}
	for {
		line, err = rs.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		if bytes.HasPrefix(line, boundary) {
			str = bytes.TrimSpace(line)
			break
		}
		part.Body = append(part.Body, line...)
	}
	if bytes.HasSuffix(str, []byte("--")) {
		err = io.EOF
	}
	ps, err1 := part2Parts(part, parent)
	if err1 != nil {
		return nil, err1
	}
	return ps, err
}

func part2Parts(p Part, parent []byte) ([]Part, error) {
	if !p.IsMultipart() {
		return []Part{p}, nil
	}
	mt, err := mime.Parse(p.Get(hdrContentType))
	if err != nil {
		return nil, err
	}
	r := bufio.NewReader(bytes.NewReader(p.Body))
	return readBody(r, []byte("--"+mt.Params[multiBound]), parent)
}

func skipEpilog(rs *bufio.Reader, boundary []byte) error {
	if boundary == nil {
		boundary = []byte(fromLinePrefix)
	}
	size := len(boundary)
	for {
		chunk, err := rs.Peek(size)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if bytes.Equal(chunk, boundary) {
			break
		}
		rs.ReadBytes('\n')
	}
	return nil
}

func skipProlog(rs *bufio.Reader, boundary []byte) error {
	for {
		line, err := rs.ReadBytes('\n')
		if err != nil {
			return err
		}
		if line = bytes.TrimSpace(line); bytes.Equal(line, boundary) {
			break
		}
	}
	return nil
}

func readPlain(rs *bufio.Reader, m *Message) error {
	var (
		buffer = make([]byte, 0, 32<<10)
		delim  = []byte(fromLinePrefix)
		size   = len(delim)
	)
	for {
		if chunk, _ := rs.Peek(size); bytes.Equal(chunk, delim) {
			break
		}
		bs, err := rs.ReadBytes('\n')
		if len(bs) > 0 {
			buffer = append(buffer, bs...)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	m.Parts = append(m.Parts, Part{Body: buffer})
	return nil
}

func readHeader(rs *bufio.Reader) (Header, error) {
	hdr := make(Header)
	for {
		line, err := rs.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			break
		}
		ix := strings.Index(line, ":")
		if ix < 0 {
			return nil, fmt.Errorf("missing colon in header: %s", line)
		}
		field, value := line[:ix], strings.TrimSpace(line[ix+1:])
		for {
			if next, _ := rs.ReadByte(); next == '\t' || next == ' ' {
				str, _ := rs.ReadString('\n')
				if next == ' ' {
					str = strings.TrimSpace(str)
				}
				value += " " + str
			} else {
				rs.UnreadByte()
				break
			}
		}
		hdr.Add(field, value)
	}
	return hdr, nil
}

var timePattern = []string{
	"Mon, _2 Jan 2006 15:04:05 -0700",
	"Mon, _2 Jan 2006 15:04:05 -0700 (MST)",
}

func parseTime(str string) time.Time {
	var (
		when time.Time
		err  error
	)
	for _, p := range timePattern {
		when, err = time.Parse(p, str)
		if err == nil {
			break
		}
	}
	return when
}

func parseValueField(str string) (string, map[string]string) {
	parts := strings.Split(str, ";")
	if len(parts) == 1 {
		return parts[0], nil
	}
	ps := make(map[string]string)
	for _, str := range parts[1:] {
		var (
			vs  = strings.Split(strings.TrimSpace(str), "=")
			key = strings.ToLower(vs[0])
			val = strings.Trim(vs[1], "\" ")
		)
		ps[strings.TrimSpace(key)] = val
	}
	return parts[0], ps
}

func parseAddress(str string) string {
	i := strings.Index(str, "<")
	if i < 0 {
		return str
	}
	str = str[i+1:]
	i = strings.Index(str, ">")
	if i > 0 {
		str = str[:i]
	}
	return str
}

func parseAddressList(str string) []string {
	var (
		ms = strings.Split(str, ",")
		as = make([]string, len(ms))
	)
	for i := 0; i < len(ms); i++ {
		as[i] = parseAddress(ms[i])
	}
	return as
}
