/*
 Copyright 2020 Qiniu Cloud (qiniu.com)

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package hq

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	// ErrEmptyText - empty text
	ErrEmptyText = errors.New("empty text")
	// ErrInvalidScanFormat - invalid fmt.Scan format
	ErrInvalidScanFormat = errors.New("invalid fmt.Scan format")
	// ErrUnmatchedScanFormat - unmatched fmt.Scan format
	ErrUnmatchedScanFormat = errors.New("unmatched fmt.Scan format")
)

// -----------------------------------------------------------------------------

// SourceCreator - hq source creator.
type SourceCreator struct{}

var (
	// Source - hq source creator
	Source SourceCreator
)

// Reader - a stream hq source
func (p SourceCreator) Reader(r io.Reader) (ret NodeSet) {
	return NewSource(r)
}

// Stdin - a stdin hq source
func (p SourceCreator) Stdin() (ret NodeSet) {
	return NewSource(os.Stdin)
}

// File - a local file hq source
func (p SourceCreator) File(htmlFile string) (ret NodeSet) {
	f, err := os.Open(htmlFile)
	if err != nil {
		return NodeSet{Err: err}
	}
	defer f.Close()
	return NewSource(f)
}

// Bytes - a bytes hq source
func (p SourceCreator) Bytes(text []byte) (ret NodeSet) {
	r := bytes.NewReader(text)
	return NewSource(r)
}

// String - a string hq source
func (p SourceCreator) String(text string) (ret NodeSet) {
	r := strings.NewReader(text)
	return NewSource(r)
}

// URI - a uri hq source
func (p SourceCreator) URI(uri string) (ret NodeSet) {
	switch {
	case strings.HasPrefix(uri, "http://"), strings.HasPrefix(uri, "https://"):
		return p.HTTP(uri)
	default:
		return p.File(uri)
	}
}

// HTTP - a http hq source
func (p SourceCreator) HTTP(url string) (ret NodeSet) {
	if ret = httpSource(url); ret.Err != nil {
		ret = httpSource(url)
	}
	return
}

func httpSource(url string) (ret NodeSet) {
	resp, err := http.Get(url)
	if err != nil {
		return NodeSet{Err: err}
	}
	defer resp.Body.Close()
	return NewSource(resp.Body)
}

// -----------------------------------------------------------------------------

// Printf prints all nodes.
func (p NodeSet) Printf(w io.Writer, format string, params ...interface{}) NodeSet {
	if p.Err != nil {
		return p
	}
	p.Data.ForEach(func(node *html.Node) error {
		html.Render(w, node)
		fmt.Fprintf(w, format, params...)
		return nil
	})
	return p
}

// Dump calls `Printf` to dump.
func (p NodeSet) Dump() NodeSet {
	return p.Printf(os.Stdout, "\n\n")
}

// -----------------------------------------------------------------------------

// ChildEqualText returns node set whose child is TextNode and value is equal to text.
func (p NodeSet) ChildEqualText(text string) (ret NodeSet) {
	return p.Match(func(node *html.Node) bool {
		return ChildEqualText(node, text)
	})
}

// EqualText returns node set who is TextNode and value is equal to text.
func (p NodeSet) EqualText(text string) (ret NodeSet) {
	return p.Match(func(node *html.Node) bool {
		return EqualText(node, text)
	})
}

// ContainsText returns node set who is TextNode and value contains text.
func (p NodeSet) ContainsText(text string) (ret NodeSet) {
	return p.Match(func(node *html.Node) bool {
		return ContainsText(node, text)
	})
}

func (p NodeSet) dataAtom(elem atom.Atom) (ret NodeSet) {
	return p.Match(func(node *html.Node) bool {
		return node.DataAtom == elem
	})
}

// Element returns nodes whose element type is v.
func (p NodeSet) Element(v interface{}) (ret NodeSet) {
	switch elem := v.(type) {
	case string:
		return p.Match(func(node *html.Node) bool {
			return node.Type == html.ElementNode && node.Data == elem
		})
	case atom.Atom:
		return p.Match(func(node *html.Node) bool {
			return node.DataAtom == elem
		})
	default:
		panic("unsupport argument type")
	}
}

// Attribute returns nodes whose attribute k's value is v.
func (p NodeSet) Attribute(k, v string) (ret NodeSet) {
	return p.Match(func(node *html.Node) bool {
		if node.Type != html.ElementNode {
			return false
		}
		for _, attr := range node.Attr {
			if attr.Key == k && attr.Val == v {
				return true
			}
		}
		return false
	})
}

// ContainsClass returns nodes whose attribute `class` contains v.
func (p NodeSet) ContainsClass(v string) (ret NodeSet) {
	return p.Match(func(node *html.Node) bool {
		if node.Type != html.ElementNode {
			return false
		}
		for _, attr := range node.Attr {
			if attr.Key == "class" {
				return ContainsClass(attr.Val, v)
			}
		}
		return false
	})
}

// H1 returns h1 nodes.
func (p NodeSet) H1() (ret NodeSet) {
	return p.dataAtom(atom.H1)
}

// H2 returns h2 nodes.
func (p NodeSet) H2() (ret NodeSet) {
	return p.dataAtom(atom.H2)
}

// H3 returns h3 nodes.
func (p NodeSet) H3() (ret NodeSet) {
	return p.dataAtom(atom.H3)
}

// H4 returns h4 nodes.
func (p NodeSet) H4() (ret NodeSet) {
	return p.dataAtom(atom.H4)
}

// Td returns td nodes.
func (p NodeSet) Td() (ret NodeSet) {
	return p.dataAtom(atom.Td)
}

// A returns a nodes.
func (p NodeSet) A() (ret NodeSet) {
	return p.dataAtom(atom.A)
}

// Img returns img nodes.
func (p NodeSet) Img() (ret NodeSet) {
	return p.dataAtom(atom.Img)
}

// Ol returns ol nodes.
func (p NodeSet) Ol() (ret NodeSet) {
	return p.dataAtom(atom.Ol)
}

// Ul returns ul nodes.
func (p NodeSet) Ul() (ret NodeSet) {
	return p.dataAtom(atom.Ul)
}

// Span returns span nodes.
func (p NodeSet) Span() (ret NodeSet) {
	return p.dataAtom(atom.Span)
}

// Div returns div nodes.
func (p NodeSet) Div() (ret NodeSet) {
	return p.dataAtom(atom.Div)
}

// Nav returns nav nodes.
func (p NodeSet) Nav() (ret NodeSet) {
	return p.dataAtom(atom.Nav)
}

// Li returns li nodes.
func (p NodeSet) Li() (ret NodeSet) {
	return p.dataAtom(atom.Li)
}

// Class returns nodes whose whose attribute `class`'s value is v.
func (p NodeSet) Class(v string) (ret NodeSet) {
	return p.Attribute("class", v)
}

// ID returns nodes whose whose attribute `id`'s value is v.
func (p NodeSet) ID(v string) (ret NodeSet) {
	return p.Attribute("id", v).One()
}

// Href returns nodes whose whose attribute `href`'s value is v.
func (p NodeSet) Href(v string) (ret NodeSet) {
	return p.Attribute("href", v)
}

// -----------------------------------------------------------------------------

// ExactText returns text node's text.
func (p NodeSet) ExactText(exactlyOne ...bool) (text string, err error) {
	node, err := p.CollectOne(exactlyOne)
	if err != nil {
		return
	}
	return ExactText(node)
}

// Text returns node's text
func (p NodeSet) Text(exactlyOne ...bool) (text string, err error) {
	node, err := p.CollectOne(exactlyOne)
	if err != nil {
		return
	}
	return Text(node), nil
}

// ScanInt gets node's text and scans it to an integer.
func (p NodeSet) ScanInt(format string, exactlyOne ...bool) (v int, err error) {
	text, err := p.Text(exactlyOne)
	if err != nil {
		return
	}
	err = fmtSscanf(text, format, &v)
	if err != nil {
		v = 0
	}
	return
}

func fmtSscanf(text, format string, v *int) (err error) {
	prefix, suffix, err := parseFormat(format)
	if err != nil {
		return
	}
	if strings.HasPrefix(text, prefix) && strings.HasSuffix(text, suffix) {
		text = text[len(prefix) : len(text)-len(suffix)]
		*v, err = strconv.Atoi(strings.Replace(text, ",", "", -1))
		return
	}
	return ErrUnmatchedScanFormat
}

func parseFormat(format string) (prefix, suffix string, err error) {
	pos := strings.Index(format, "%d")
	if pos < 0 {
		pos = strings.Index(format, "%v")
	}
	if pos < 0 {
		err = ErrInvalidScanFormat
		return
	}
	prefix = strings.Replace(format[:pos], "%%", "%", -1)
	suffix = strings.Replace(format[pos+2:], "%%", "%", -1)
	return
}

// UnitedFloat gets node's text and converts it into a united float.
func (p NodeSet) UnitedFloat(exactlyOne ...bool) (v float64, err error) {
	text, err := p.Text(exactlyOne)
	if err != nil {
		return
	}
	n := len(text)
	if n == 0 {
		return 0, ErrEmptyText
	}
	unit := 1.0
	switch text[n-1] {
	case 'k', 'K':
		unit = 1000
		text = text[:n-1]
	}
	v, err = strconv.ParseFloat(text, 64)
	if err != nil {
		return
	}
	return v * unit, nil
}

// Int gets node's text and converts it into an integer.
func (p NodeSet) Int(exactlyOne ...bool) (v int, err error) {
	text, err := p.Text(exactlyOne)
	if err != nil {
		return
	}
	return strconv.Atoi(strings.Replace(text, ",", "", -1))
}

// AttrVal returns node attriute k's value.
func (p NodeSet) AttrVal(k string, exactlyOne ...bool) (text string, err error) {
	node, err := p.CollectOne(exactlyOne)
	if err != nil {
		return
	}
	return AttributeVal(node, k)
}

// HrefVal returns node attriute href's value.
func (p NodeSet) HrefVal(exactlyOne ...bool) (text string, err error) {
	return p.AttrVal("href", exactlyOne)
}

// -----------------------------------------------------------------------------
