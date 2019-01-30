package bpi

import (
	"net/mail"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// MailLoader represents any backend that returns mail.Message
type MailLoader interface {
	// All returns all messages in the directory
	All() ([]*mail.Message, error)
	// One returns message by Message-ID
	One(id string) (*mail.Message, error)
}

// DirLoader implements MailLoader recursively scanning directory with emails per file
type DirLoader struct {
	dir      string
	idToPath map[string]string
}

var _ MailLoader = &DirLoader{}

// NewDirLoader creates new DirLoader on dir path
func NewDirLoader(dir string) *DirLoader {
	return &DirLoader{
		dir:      dir,
		idToPath: make(map[string]string),
	}
}

// All implements MailLoader interface, returns all messages in the directory
func (l *DirLoader) All() ([]*mail.Message, error) {
	var result []*mail.Message

	err := filepath.Walk(l.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		m, err := parseMsgFile(path)
		if err != nil {
			return errors.Wrapf(err, "file path: %s", path)
		}

		id := getID(m.Header.Get("Message-Id"))
		if id != "" {
			l.idToPath[id] = path
			result = append(result, m)
		}

		return nil
	})

	return result, err
}

// One implements MailLoader interface, returns message by Message-ID
func (l *DirLoader) One(id string) (*mail.Message, error) {
	path, ok := l.idToPath[id]
	if !ok {
		return nil, errors.Errorf("mbox for id: '%s' not found", id)
	}

	return parseMsgFile(path)
}

func parseMsgFile(path string) (*mail.Message, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "can not parse mbox file")
	}

	return mail.ReadMessage(file)
}
