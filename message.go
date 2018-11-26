package bpi

import (
	"bufio"
	"io"
	"net/mail"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// MessageHeader contains headers of a message
type MessageHeader struct {
	ID      string
	ReplyTo string
	Title   string
	Author  *mail.Address
	Date    time.Time
	To      string
	Cc      string
}

// MessageHeader contains headers of a message and body as a list of blocks
type Message struct {
	*MessageHeader

	Body      []*BodyBlock
	SignedOff bool
}

// BodyBlock represents part of message body
type BodyBlock struct {
	Type string
	Body string
}

// NewMessageHeader parses mail.Message to MessageHeader
func NewMessageHeader(mm *mail.Message) (*MessageHeader, error) {
	subject := mm.Header.Get("Subject")

	date, err := mail.ParseDate(mm.Header.Get("Date"))
	if err != nil {
		return nil, err
	}

	author, err := mail.ParseAddress(mm.Header.Get("From"))
	if err != nil {
		return nil, err
	}

	var to string
	if mm.Header.Get("To") != "" {
		toList, err := mail.ParseAddressList(mm.Header.Get("To"))
		if err != nil {
			return nil, err
		}

		names := make([]string, len(toList))
		for i, addr := range toList {
			names[i] = addr.Name
		}

		to = strings.Join(names, ", ")
	}

	var cc string
	if mm.Header.Get("Cc") != "" {
		ccList, err := mail.ParseAddressList(mm.Header.Get("Cc"))
		// just skip incorrect CC
		if err == nil {
			names := make([]string, len(ccList))
			for i, addr := range ccList {
				names[i] = addr.Name
			}

			cc = strings.Join(names, ", ")
		} else {
			logrus.Warnf("incorrect cc in message '%s': %s", mm.Header.Get("Message-Id"), mm.Header.Get("Cc"))
		}
	}

	h := &MessageHeader{
		ID:      getID(mm.Header.Get("Message-Id")),
		ReplyTo: getID(mm.Header.Get("In-Reply-To")),
		Author:  author,
		Date:    date,
		Title:   subject,
		To:      to,
		Cc:      cc,
	}

	return h, nil
}

// NewMessage parses mail.Message to Message
func NewMessage(mm *mail.Message) (*Message, error) {
	h, err := NewMessageHeader(mm)
	if err != nil {
		return nil, err
	}

	m := &Message{MessageHeader: h}

	blocks, _, err := parseBody(mm.Body)
	if err != nil {
		return nil, err
	}

	m.Body = blocks

	return m, nil
}

func parseBody(body io.Reader) ([]*BodyBlock, bool, error) {
	state := ""
	inQuotes := "inQuotes"
	inPatch := "inPatch"

	var signedOff bool
	var blocks []*BodyBlock
	currentBlock := &BodyBlock{}

	newBlock := func(t string) {
		if currentBlock != nil {
			blocks = append(blocks, currentBlock)
		}
		currentBlock = &BodyBlock{
			Type: t,
		}
	}

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()

		switch true {
		case state == "" && strings.HasPrefix(line, "Signed-off-by:"):
			signedOff = true // FIXME: most probably need to compare with From
		case state == "" && strings.HasPrefix(line, ">"):
			// start new quotes block
			state = inQuotes
			newBlock("quotes")
			currentBlock.Body = currentBlock.Body + line + "\n"
		case state == "" && patchbreak(line):
			state = inPatch
			newBlock("patch")
			currentBlock.Body = currentBlock.Body + line + "\n"
		case state == inQuotes && (len(line) == 0 || line[0] != '>'):
			state = ""
			newBlock("")
			currentBlock.Body = currentBlock.Body + line + "\n"
		default:
			currentBlock.Body = currentBlock.Body + line + "\n"
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, false, err
	}

	if currentBlock != nil {
		blocks = append(blocks, currentBlock)
	}

	return blocks, signedOff, nil
}

func getID(id string) string {
	if len(id) > 3 && id[0] == '<' && id[len(id)-1] == '>' {
		return id[1 : len(id)-1]
	}

	return ""
}
