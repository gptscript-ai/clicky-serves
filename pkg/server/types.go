package server

import "github.com/gptscript-ai/go-gptscript"

type toolRequest struct {
	gptscript.Opts    `json:",inline"`
	gptscript.ToolDef `json:",inline"`
	content           `json:",inline"`
}

type content struct {
	Content string `json:"content"`
}

func (c *content) String() string {
	return c.Content
}

type fileRequest struct {
	gptscript.Opts `json:",inline"`
	File           string `json:"file"`
}

type parseRequest struct {
	gptscript.Opts `json:",inline"`
	content        `json:",inline"`
	File           string `json:"file"`
}
