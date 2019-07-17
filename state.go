package main

import (
	"fmt"
	"strings"

	"github.com/andersfylling/disgord"
	"github.com/inconshreveable/log15"
	"github.com/russross/blackfriday/v2"
)

type evalSessions struct {
	EvalKey string
}

func NewEvalSessions(client *disgord.Client, evalKey string) *evalSessions {
	return &evalSessions{
		EvalKey: evalKey,
	}
}

func (e *evalSessions) OnMessage(s disgord.Session, data *disgord.MessageCreate) {
	msg := data.Message
	codes, err := parseForBot(msg.Content)
	if err != nil {
		msg.Reply(s, err.Error())
		return
	}

	replyParts := []string{}

	for _, code := range codes {
		resp, err := e.runCode(code.language, code.contents)
		if err != nil {
			replyParts = append(replyParts, fmt.Sprintf("error running code: %v", err.Error()))
			continue
		}
		replyParts = append(replyParts, resp)
	}

	msg.Reply(s, strings.Join(replyParts, "\n\n"))
}

func (e *evalSessions) OnUpdate(s disgord.Session, data *disgord.MessageUpdate) {
	msg := data.Message
	_ = msg
	// TODO
}

type evalCode struct {
	language string
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
		case strings.HasPrefix(cmd, "lang="):
			ret.language = cmd[len("lang="):]
		default:
			return nil, fmt.Errorf("unrecognized command: %v", cmd)
		}
	}
	return ret, nil
}

func (e *evalCode) merge(rhs *evalCode) {
	if rhs.language != "" {
		e.language = rhs.language
	}
	if rhs.contents != "" {
		e.contents = rhs.contents
	}
}

func (e *evalCode) copy() *evalCode {
	c := *e
	return &c
}

func parseForBot(msg string) ([]*evalCode, error) {
	parsed := blackfriday.New(blackfriday.WithExtensions(blackfriday.CommonExtensions)).Parse([]byte(msg))

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
				evaling = true
				newEvalCode, err := evalCodeFromCommand(strings.TrimSpace(strings.TrimPrefix(string(n.Literal), "!eval")))
				if err != nil {
					retErr = err
					return blackfriday.Terminate
				}
				evalCommandCode = newEvalCode
			}
		case blackfriday.CodeBlock:
			if !evaling {
				break
			}
			blockCode := evalCommandCode.copy()
			blockCode.language = string(n.CodeBlockData.Info)
			blockCode.contents = string(n.Literal)
			result = append(result, blockCode)
		default:
			log15.Debug("ignoring block we don't care about", "blocktype", n.Type)
		}

		return blackfriday.GoToNext
	})

	return result, retErr
}
