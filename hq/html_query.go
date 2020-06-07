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
	"errors"
	"io"
	"syscall"

	"golang.org/x/net/html"
)

var (
	// ErrNotFound - not found
	ErrNotFound = syscall.ENOENT
	// ErrBreak - break
	ErrBreak = syscall.ELOOP
	// ErrTooManyNodes - too may nodes
	ErrTooManyNodes = errors.New("too many nodes")
	// ErrInvalidNode  - invalid node
	ErrInvalidNode = errors.New("invalid node")
	// ErrNotTextNode  - not a text node
	ErrNotTextNode = errors.New("not a text node")
)

// -----------------------------------------------------------------------------

type cachedNodeEnum interface {
	Cached() int
}

// NodeEnum - node enumerator
type NodeEnum interface {
	ForEach(filter func(node *html.Node) error)
}

// NodeSet - node set
type NodeSet struct {
	Data NodeEnum
	Err  error
}

// Ok returns if node set is valid or not.
func (p NodeSet) Ok() bool {
	return p.Err == nil
}

// CachedLen returns cached len.
func (p NodeSet) CachedLen() int {
	if cds, ok := p.Data.(cachedNodeEnum); ok {
		return cds.Cached()
	}
	return 0
}

// Cache caches node set.
func (p NodeSet) Cache() NodeSet {
	if _, ok := p.Data.(cachedNodeEnum); ok {
		return p
	}
	nodes, err := p.Collect()
	if err != nil {
		return NodeSet{Err: err}
	}
	return NodeSet{Data: &fixNodes{nodes}}
}

// ForEach visits the node set.
func (p NodeSet) ForEach(filter func(node NodeSet)) {
	if p.Err == nil {
		p.Data.ForEach(func(node *html.Node) error {
			t := NodeSet{Data: oneNode{node}}
			filter(t)
			return nil
		})
	}
}

// -----------------------------------------------------------------------------

type oneNode struct {
	*html.Node
}

func (p oneNode) ForEach(filter func(node *html.Node) error) {
	filter(p.Node)
}

func (p oneNode) Cached() int {
	return 1
}

// NewSource creates the html document, and treats it as a node set.
func NewSource(r io.Reader) (ret NodeSet) {
	doc, err := html.Parse(r)
	if err != nil {
		return NodeSet{Err: err}
	}
	return NodeSet{Data: oneNode{doc}}
}

// -----------------------------------------------------------------------------

type fixNodes struct {
	nodes []*html.Node
}

func (p *fixNodes) ForEach(filter func(node *html.Node) error) {
	for _, node := range p.nodes {
		if filter(node) == ErrBreak {
			return
		}
	}
}

func (p *fixNodes) Cached() int {
	return len(p.nodes)
}

// Nodes creates a fixed node set.
func Nodes(nodes ...*html.Node) (ret NodeSet) {
	return NodeSet{Data: &fixNodes{nodes}}
}

// -----------------------------------------------------------------------------

type anyNodes struct {
	data NodeEnum
}

func (p *anyNodes) ForEach(filter func(node *html.Node) error) {
	p.data.ForEach(func(node *html.Node) error {
		anyForEach(node, filter)
		return nil
	})
}

func (p *anyNodes) Cached() int {
	return -1
}

func anyForEach(p *html.Node, filter func(node *html.Node) error) error {
	if err := filter(p); err == nil || err == ErrBreak {
		return err
	}
	for node := p.FirstChild; node != nil; node = node.NextSibling {
		if anyForEach(node, filter) == ErrBreak {
			return ErrBreak
		}
	}
	return nil
}

// Any returns deeply visiting node set.
func (p NodeSet) Any() (ret NodeSet) {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &anyNodes{p.Data}}
}

// -----------------------------------------------------------------------------

type childLevelNodes struct {
	data  NodeEnum
	level int
}

func (p *childLevelNodes) ForEach(filter func(node *html.Node) error) {
	p.data.ForEach(func(node *html.Node) error {
		return childLevelForEach(node, p.level, filter)
	})
}

func childLevelForEach(p *html.Node, level int, filter func(node *html.Node) error) error {
	if level == 0 {
		return filter(p)
	}
	level--
	for node := p.FirstChild; node != nil; node = node.NextSibling {
		if childLevelForEach(node, level, filter) == ErrBreak {
			return ErrBreak
		}
	}
	return nil
}

type parentLevelNodes struct {
	data  NodeEnum
	level int
}

func (p *parentLevelNodes) ForEach(filter func(node *html.Node) error) {
	p.data.ForEach(func(node *html.Node) error {
		return parentLevelForEach(node, p.level, filter)
	})
}

func parentLevelForEach(p *html.Node, level int, filter func(node *html.Node) error) error {
	for level < 0 {
		if p = p.Parent; p == nil {
			return ErrNotFound
		}
		level++
	}
	return filter(p)
}

// Child returns child node set.
func (p NodeSet) Child() (ret NodeSet) {
	return p.ChildN(1)
}

// ChildN return N level's generation node set.
func (p NodeSet) ChildN(level int) (ret NodeSet) {
	if p.Err != nil || level == 0 {
		return p
	}
	if level > 0 {
		return NodeSet{Data: &childLevelNodes{p.Data, level}}
	}
	return NodeSet{Data: &parentLevelNodes{p.Data, level}}
}

// Parent return parent node set.
func (p NodeSet) Parent() (ret NodeSet) {
	return p.ChildN(-1)
}

// ParentN return N level's ancestor node set.
func (p NodeSet) ParentN(level int) (ret NodeSet) {
	return p.ChildN(-level)
}

// One returns the first node as a node set.
func (p NodeSet) One() (ret NodeSet) {
	if _, ok := p.Data.(oneNode); ok {
		return p
	}
	node, err := p.CollectOne()
	if err != nil {
		return NodeSet{Err: err}
	}
	return NodeSet{Data: oneNode{node}}
}

// -----------------------------------------------------------------------------

type siblingNodes struct {
	data  NodeEnum
	delta int
}

func (p *siblingNodes) ForEach(filter func(node *html.Node) error) {
	p.data.ForEach(func(node *html.Node) error {
		return siblingForEach(node, p.delta, filter)
	})
}

func siblingForEach(p *html.Node, delta int, filter func(node *html.Node) error) error {
	for delta > 0 {
		if p = p.NextSibling; p == nil {
			return ErrNotFound
		}
		delta--
	}
	for delta < 0 {
		if p = p.PrevSibling; p == nil {
			return ErrNotFound
		}
		delta++
	}
	return filter(p)
}

// NextSibling returns next sibling node set.
func (p NodeSet) NextSibling(delta int) (ret NodeSet) {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &siblingNodes{p.Data, delta}}
}

// PrevSibling returns prev sibling node set.
func (p NodeSet) PrevSibling(delta int) (ret NodeSet) {
	return p.NextSibling(-delta)
}

// -----------------------------------------------------------------------------

type prevSiblingNodes struct {
	data NodeEnum
}

func (p *prevSiblingNodes) ForEach(filter func(node *html.Node) error) {
	p.data.ForEach(func(node *html.Node) error {
		for p := node.PrevSibling; p != nil; p = p.PrevSibling {
			if filter(p) == ErrBreak {
				return ErrBreak
			}
		}
		return nil
	})
}

// PrevSiblings return all prev sibling node set.
func (p NodeSet) PrevSiblings() (ret NodeSet) {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &prevSiblingNodes{p.Data}}
}

// -----------------------------------------------------------------------------

type nextSiblingNodes struct {
	data NodeEnum
}

func (p *nextSiblingNodes) ForEach(filter func(node *html.Node) error) {
	p.data.ForEach(func(node *html.Node) error {
		for p := node.NextSibling; p != nil; p = p.NextSibling {
			if filter(p) == ErrBreak {
				return ErrBreak
			}
		}
		return nil
	})
}

// NextSiblings return all next sibling node set.
func (p NodeSet) NextSiblings() (ret NodeSet) {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &nextSiblingNodes{p.Data}}
}

// -----------------------------------------------------------------------------

type firstChildNodes struct {
	data     NodeEnum
	nodeType html.NodeType
}

func (p *firstChildNodes) ForEach(filter func(node *html.Node) error) {
	p.data.ForEach(func(node *html.Node) error {
		child, err := FirstChild(node, p.nodeType)
		if err != nil {
			return err
		}
		return filter(child)
	})
}

// FirstChild returns first nodeType node as a node set.
func (p NodeSet) FirstChild(nodeType html.NodeType) (ret NodeSet) {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &firstChildNodes{p.Data, nodeType}}
}

// FirstTextChild returns first text node as a node set.
func (p NodeSet) FirstTextChild() (ret NodeSet) {
	return p.FirstChild(html.TextNode)
}

// FirstElementChild returns first element node as a node set.
func (p NodeSet) FirstElementChild() (ret NodeSet) {
	return p.FirstChild(html.ElementNode)
}

// -----------------------------------------------------------------------------

type lastChildNodes struct {
	data     NodeEnum
	nodeType html.NodeType
}

func (p *lastChildNodes) ForEach(filter func(node *html.Node) error) {
	p.data.ForEach(func(node *html.Node) error {
		child, err := LastChild(node, p.nodeType)
		if err != nil {
			return err
		}
		return filter(child)
	})
}

// LastChild returns last nodeType node as a node set.
func (p NodeSet) LastChild(nodeType html.NodeType) (ret NodeSet) {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &lastChildNodes{p.Data, nodeType}}
}

// LastTextChild returns last text node as a node set.
func (p NodeSet) LastTextChild() (ret NodeSet) {
	return p.LastChild(html.TextNode)
}

// LastElementChild returns last element node as a node set.
func (p NodeSet) LastElementChild() (ret NodeSet) {
	return p.LastChild(html.ElementNode)
}

// -----------------------------------------------------------------------------

type matchedNodes struct {
	data   NodeEnum
	filter func(node *html.Node) bool
}

func (p *matchedNodes) ForEach(filter func(node *html.Node) error) {
	p.data.ForEach(func(node *html.Node) error {
		if p.filter(node) {
			return filter(node)
		}
		return ErrNotFound
	})
}

// Match filters the node set.
func (p NodeSet) Match(filter func(node *html.Node) bool) (ret NodeSet) {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &matchedNodes{p.Data, filter}}
}

// -----------------------------------------------------------------------------

type textNodes struct {
	data      NodeEnum
	doReplace bool
}

func (p *textNodes) ForEach(filter func(node *html.Node) error) {
	p.data.ForEach(func(t *html.Node) error {
		node := &html.Node{
			Parent: t,
			Type:   html.TextNode,
			Data:   Text(t),
		}
		if p.doReplace {
			t.FirstChild = node
			t.LastChild = node
		}
		return filter(node)
	})
}

// ChildrenAsText converts all children as text node.
func (p NodeSet) ChildrenAsText(doReplace bool) (ret NodeSet) {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &textNodes{p.Data, doReplace}}
}

// -----------------------------------------------------------------------------

// CollectOne collects one node of a node set.
// If exactly is true, it returns ErrTooManyNodes when node set is more than one.
func (p NodeSet) CollectOne(exactly ...bool) (item *html.Node, err error) {
	if p.Err != nil {
		return nil, p.Err
	}
	err = ErrNotFound
	if exactly != nil {
		if !exactly[0] {
			panic("please call `CollectOne()` instead of `CollectOne(false)`")
		}
		p.Data.ForEach(func(node *html.Node) error {
			if err == ErrNotFound {
				item, err = node, nil
				return nil
			}
			err = ErrTooManyNodes
			return ErrBreak
		})
	} else {
		p.Data.ForEach(func(node *html.Node) error {
			item, err = node, nil
			return ErrBreak
		})
	}
	return
}

// Collect collects all nodes of the node set.
func (p NodeSet) Collect() (items []*html.Node, err error) {
	if p.Err != nil {
		return nil, p.Err
	}
	p.Data.ForEach(func(node *html.Node) error {
		items = append(items, node)
		return nil
	})
	return
}

// -----------------------------------------------------------------------------
