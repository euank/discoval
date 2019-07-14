package main

import (
	"fmt"
	"os"

	"github.com/andersfylling/disgord"
	"github.com/andersfylling/disgord/std"
	"github.com/inconshreveable/log15"
)

func handleEvals(s disgord.Session, data *disgord.MessageCreate) {
	msg := data.Message
	log15.Info("info", "msg", fmt.Sprintf("%+v", msg))
}

func main() {
	client := disgord.New(&disgord.Config{
		BotToken: os.Getenv("DISGORD_TOKEN"),
		Logger:   disgord.DefaultLogger(true),
	})
	defer client.StayConnectedUntilInterrupted()

	evalKey := os.Getenv("EVAL_KEY")
	if evalKey == "" {
		log15.Error("EVAL_KEY must be set")
		os.Exit(1)
	}

	evalSessions := NewEvalSessions(client, evalKey)

	log, _ := std.NewLogFilter(client)
	filter, _ := std.NewMsgFilter(client)
	client.On(disgord.EvtMessageCreate, filter.NotByBot, log.LogMsg, evalSessions.OnMessage)
	client.On(disgord.EvtMessageUpdate, filter.NotByBot, log.LogMsg, evalSessions.OnUpdate)
}
