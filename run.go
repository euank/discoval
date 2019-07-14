package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const api = "https://eval2.esk.io"

type evalRequest struct {
	Key      string `json:"key"`
	Env      string `json:"env"`
	Contents string `json:"contents"`
}

type evalResp struct {
	Response *RunResponse `json:"response"`
}

type RunResponse struct {
	Stdout  string `json:"stdout"`
	Stderr  string `json:"stderr"`
	Timeout bool   `json:"timeout"`
}

func (e *evalSessions) runCode(lang, code string) (string, error) {
	body, _ := json.Marshal(&evalRequest{
		Key:      e.EvalKey,
		Env:      lang,
		Contents: code,
	})
	resp, err := http.Post(api, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("error making eval request: %v", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("error making eval request: StatusCode %d", resp.StatusCode)
	}
	var eresp evalResp
	if err := json.NewDecoder(resp.Body).Decode(&eresp); err != nil {
		return "", fmt.Errorf("invalid eval response")
	}

	return formatResp(eresp.Response), nil

}

func formatResp(r *RunResponse) string {
	parts := []string{}
	if r.Stdout != "" {
		parts = append(parts, "stdout", "```\n"+r.Stdout+"```")
	}
	if r.Stderr != "" {
		parts = append(parts, "stderr", "```\n"+r.Stderr+"```")
	}
	if r.Timeout {
		parts = append(parts, "request timed out")
	}
	if len(parts) == 0 {
		parts = append(parts, "no output")
	}
	return strings.Join(parts, "\n")
}
