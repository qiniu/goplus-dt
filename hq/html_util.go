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
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// -----------------------------------------------------------------------------

// ContainsClass checks class v is in source classes or not.
func ContainsClass(source string, v string) bool {
	for {
		pos := strings.IndexByte(source, ' ')
		if pos < 0 {
			return source == v
		}
		if source[:pos] == v {
			return true
		}
		source = source[pos+1:]
	}
}

// AttributeVal returns attribute k's value of a node.
func AttributeVal(node *html.Node, k string) (v string, err error) {
	if node.Type != html.ElementNode {
		return "", ErrInvalidNode
	}
	for _, attr := range node.Attr {
		if attr.Key == k {
			return attr.Val, nil
		}
	}
	return "", ErrNotFound
}

// FirstChild returns first nodeType node.
func FirstChild(node *html.Node, nodeType html.NodeType) (p *html.Node, err error) {
	for p = node.FirstChild; p != nil; p = p.NextSibling {
		if p.Type == nodeType {
			return p, nil
		}
	}
	return nil, ErrNotFound
}

// LastChild returns last nodeType node.
func LastChild(node *html.Node, nodeType html.NodeType) (p *html.Node, err error) {
	for p = node.LastChild; p != nil; p = p.PrevSibling {
		if p.Type == nodeType {
			return p, nil
		}
	}
	return nil, ErrNotFound
}

// -----------------------------------------------------------------------------

// ChildEqualText checks if child node is TextNode and value is equal to text or not.
func ChildEqualText(node *html.Node, text string) bool {
	p := node.FirstChild
	if p == nil || p.NextSibling != nil {
		return false
	}
	return EqualText(p, text)
}

// EqualText checks if node is TextNode and value is equal to text or not.
func EqualText(node *html.Node, text string) bool {
	if node.Type != html.TextNode {
		return false
	}
	return node.Data == text
}

// ContainsText checks if node is TextNode and value contains text or not.
func ContainsText(node *html.Node, text string) bool {
	if node.Type != html.TextNode {
		return false
	}
	return strings.Contains(node.Data, text)
}

// ExactText returns a text node's text.
func ExactText(node *html.Node) (string, error) {
	if node.Type != html.TextNode {
		return node.Data, nil
	}
	return "", ErrNotTextNode
}

// Text extracts formatted node text.
func Text(node *html.Node) string {
	var printer textPrinter
	printer.printNode(node)
	return string(printer.data)
}

type textPrinter struct {
	data         []byte
	notLineStart bool
}

func (p *textPrinter) printText(v string) {
	if v == "" {
		return
	}
	if p.notLineStart {
		p.data = append(p.data, ' ')
	} else {
		p.notLineStart = true
	}
	p.data = append(p.data, v...)
}

func (p *textPrinter) printNode(node *html.Node) {
	if node == nil {
		return
	}
	if node.Type == html.TextNode {
		p.printText(strings.Trim(node.Data, " \t\r\n"))
		return
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		p.printNode(child)
	}
	switch node.DataAtom {
	case atom.P:
		p.data = append(p.data, '\n')
		p.notLineStart = false
	}
}

// -----------------------------------------------------------------------------
