package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/inconshreveable/log15"
	"github.com/russross/blackfriday/v2"
)

type evalSessionPiece struct {
	user     disgord.Snowflake
	contents string
}

type evalSession struct {
	channel   disgord.Snowflake
	expiresAt time.Time
	contents  []*evalCode
}

func (s *evalSession) addCode(code *evalCode) {
	s.contents = append(s.contents, code)
	s.expiresAt = time.Now().Add(1 * time.Hour)
}

type evalSessions struct {
	mu sync.Mutex
	// sessionID -> evalSessions
	sessions map[string]*evalSession

	// channelID -> sessionIDs
	channelSessions map[disgord.Snowflake][]string

	client *disgord.Client
}

func NewEvalSessions(client *disgord.Client) *evalSessions {
	return &evalSessions{
		sessions:        make(map[string]*evalSession),
		channelSessions: make(map[disgord.Snowflake][]string),
		client:          client,
	}
}

func (e *evalSessions) OnMessage(s disgord.Session, data *disgord.MessageCreate) {
	msg := data.Message
	codes, err := e.parseForBot(msg)
	if err != nil {
		msg.Reply(s, err.Error())
		return
	}

	sessionsToRun := map[string]*evalSession{}

Outer:
	for _, code := range codes {
		if code.id == "" {
			code.id = fmt.Sprintf("%s", msg.ID)
		}

		// We have code to run now. First let's see if this id has an existing session
		sessions := e.channelSessions[msg.ChannelID]
		for _, sess := range sessions {
			if sess == code.id {
				existingSess := e.sessions[sess]
				existingSess.addCode(code)
				sessionsToRun[sess] = existingSess
				continue Outer
			}
		}
		// new session
		newSess := &evalSession{
			channel:   msg.ChannelID,
			expiresAt: time.Now().Add(1 * time.Hour),
			contents:  []*evalCode{code},
		}
		e.sessions[code.id] = newSess
		if e.channelSessions[msg.ChannelID] == nil {
			e.channelSessions[msg.ChannelID] = []string{}
		}
		e.channelSessions[msg.ChannelID] = append(e.channelSessions[msg.ChannelID], code.id)
		sessionsToRun[code.id] = newSess
	}
}

func (e *evalSessions) OnUpdate(s disgord.Session, data *disgord.MessageCreate) {
	msg := data.Message
	_ = msg
	// TODO
}

type evalCode struct {
	id       string
	language string
	filename string
	contents string
}

func evalCodeFromCommand(s string) (*evalCode, error) {
	ret := &evalCode{}
	commands := strings.Split(s, ",")
	for _, cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		switch {
		case cmd == "":
			continue
		case strings.HasPrefix(cmd, "file="):
			ret.filename = cmd[len("file="):]
		case strings.HasPrefix(cmd, "id="):
			ret.id = cmd[len("id="):]
		case strings.HasPrefix(cmd, "lang="):
			ret.language = cmd[len("lang="):]
		default:
			return nil, fmt.Errorf("unrecognized command: %v", cmd)
		}
	}
	return ret, nil
}

func (e *evalCode) merge(rhs *evalCode) {
	if rhs.id != "" {
		e.id = rhs.id
	}
	if rhs.language != "" {
		e.language = rhs.language
	}
	if rhs.filename != "" {
		e.filename = rhs.filename
	}
	if rhs.contents != "" {
		e.contents = rhs.contents
	}
}

func (e *evalCode) copy() *evalCode {
	c := *e
	return &c
}

func (e *evalSessions) parseForBot(msg *disgord.Message) ([]*evalCode, error) {
	parsed := blackfriday.New().Parse([]byte(msg.Content))

	var retErr error
	var evaling bool
	result := []*evalCode{}
	evalCommandCode := &evalCode{}

	parsed.Walk(func(n *blackfriday.Node, entering bool) blackfriday.WalkStatus {
		if !entering {
			return blackfriday.GoToNext
		}

		switch n.Type {
		case blackfriday.Document, blackfriday.Paragraph, blackfriday.Text:
			if strings.HasPrefix(string(n.Literal), "!eval") {
				log15.Info("evaling code")
				evaling = true
				newEvalCode, err := evalCodeFromCommand(strings.TrimSpace(strings.TrimPrefix(string(n.Literal), "!eval")))
				if err != nil {
					retErr = err
					return blackfriday.Terminate
				}
				evalCommandCode = newEvalCode
			}
		case blackfriday.CodeBlock, blackfriday.Code:
			if !evaling {
				break
			}
			blockCode := evalCommandCode.copy()
			lines := strings.Split(string(n.Literal), "\n")
			if len(n.CodeBlockData.Info) > 0 {
				blockCode.language = strings.TrimSpace(string(n.CodeBlockData.Info))
				blockCode.contents = strings.Join(lines, "\n")
			} else if n.Type == blackfriday.Code {
				if len(lines) > 0 && lines[0] != "" {
					blockCode.language = lines[0]
				}
				blockCode.contents = strings.Join(lines[1:], "\n")
			}
			result = append(result, blockCode)
		default:
			log15.Debug("ignoring block we don't care about", "blocktype", n.Type)
		}

		return blackfriday.GoToNext
	})

	return result, retErr
}
