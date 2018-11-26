package bpi

import (
	"sort"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Store represents any backend that returns Messages
type Store interface {
	// List returns N message headers, it will support pagination
	List() ([]*MessageHeader, error)
	// Get returns Message by Message-ID
	Get(id string) (*Message, error)
	// ThreadCount returns number of messages in thread by Message-ID
	ThreadCount(id string) (int, error)
	// Thread returns thread by Message-ID
	Thread(id string) (*TreeMessage, error)
}

// TreeMessage extends Message with Children and Level
type TreeMessage struct {
	*Message
	Children []*TreeMessage
	Level    int
}

// List returns flat list of all messages in the tree
func (m *TreeMessage) List() []*TreeMessage {
	var result []*TreeMessage
	stack := []*TreeMessage{m}
	for {
		if len(stack) == 0 {
			break
		}

		item := stack[0]
		stack = append(item.Children, stack[1:]...)

		result = append(result, item)
	}

	return result
}

type treeItem struct {
	ID       string
	Parent   string
	Children []*treeItem
}

// MemStore implements Store interface in memory
type MemStore struct {
	loader  MailLoader
	idIndex map[string]*MessageHeader
	tree    map[string]*treeItem

	roots []*MessageHeader
}

var _ Store = &MemStore{}

// NewMemStore creates new MemStore using MailLoader as underlying backend
func NewMemStore(l MailLoader) (*MemStore, error) {
	m := &MemStore{
		loader:  l,
		idIndex: make(map[string]*MessageHeader),
		tree:    make(map[string]*treeItem),
	}

	if err := m.init(); err != nil {
		return nil, errors.Wrap(err, "can not initialize memory store")
	}

	return m, nil
}

// List implements Store interface, returns N message headers, it will support pagination
func (s *MemStore) List() ([]*MessageHeader, error) {
	if len(s.roots) == 0 {
		for _, m := range s.idIndex {
			if m.ReplyTo == "" {
				s.roots = append(s.roots, m)
			}
		}
		sort.Slice(s.roots, func(i, j int) bool {
			return s.roots[i].Date.After(s.roots[j].Date)
		})
	}

	limit := len(s.roots)
	if limit > 20 {
		limit = 20
	}

	return s.roots[:limit], nil
}

// Get implements Store interface, returns Message by Message-ID
func (s *MemStore) Get(id string) (*Message, error) {
	mm, err := s.loader.One(id)
	if err != nil {
		return nil, err
	}

	return NewMessage(mm)
}

// ThreadCount implements Store interface, returns number of messages in thread by Message-ID
func (s *MemStore) ThreadCount(id string) (int, error) {
	parent, err := s.threadHead(id)
	if err != nil {
		return 0, err
	}

	var result int
	stack := []*treeItem{parent}
	for {
		if len(stack) == 0 {
			break
		}

		item := stack[0]
		stack = append(stack[1:], item.Children...)

		result++
	}

	return result, nil
}

// Thread implements Store interface, returns thread by Message-ID
func (s *MemStore) Thread(id string) (*TreeMessage, error) {
	parent, err := s.threadHead(id)
	if err != nil {
		return nil, err
	}

	return s.toTreeMessage(parent, 0)
}

func (s *MemStore) toTreeMessage(item *treeItem, level int) (*TreeMessage, error) {
	mm, err := s.loader.One(item.ID)
	if err != nil {
		return nil, err
	}

	m, err := NewMessage(mm)
	if err != nil {
		return nil, err
	}

	children := make([]*TreeMessage, len(item.Children))
	for i, child := range item.Children {
		tm, err := s.toTreeMessage(child, level+1)
		if err != nil {
			return nil, err
		}

		children[i] = tm
	}

	return &TreeMessage{
		Message:  m,
		Children: children,
		Level:    level,
	}, nil
}

func (s *MemStore) threadHead(id string) (*treeItem, error) {
	item, ok := s.tree[id]
	if !ok {
		return nil, errors.New("thread head not found")
	}

	for {
		if item.Parent == "" {
			break
		}

		item = s.tree[item.Parent]
	}

	return item, nil
}

func (s *MemStore) init() error {
	list, err := s.loader.All()
	if err != nil {
		return errors.Wrap(err, "can not load messages")
	}

	for _, m := range list {
		m, err := NewMessageHeader(m)
		if err != nil {
			return errors.Wrap(err, "can not parse message header")
		}

		s.idIndex[m.ID] = m
	}

	logrus.Debugf("loaded: %d messages", len(s.idIndex))

	// create treeItem for each message
	for id, m := range s.idIndex {
		var parent string
		if m.ReplyTo != "" {
			if _, ok := s.idIndex[m.ReplyTo]; ok {
				parent = m.ReplyTo
			}
		}

		s.tree[m.ID] = &treeItem{ID: id, Parent: parent}
	}

	// fill children
	// FIXME can be recursion
	for _, item := range s.tree {
		if item.Parent == "" {
			continue
		}

		parent := s.tree[item.Parent]
		parent.Children = append(parent.Children, item)
	}

	// sort children
	for _, item := range s.tree {
		if len(item.Children) <= 1 {
			continue
		}

		sort.Slice(item.Children, func(i, j int) bool {
			a := s.idIndex[item.Children[i].ID]
			b := s.idIndex[item.Children[j].ID]
			return a.Date.Before(b.Date)
		})
	}

	logrus.Debug("index is ready")

	return nil
}
