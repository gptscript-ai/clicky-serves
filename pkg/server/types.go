package server

import (
	"github.com/gptscript-ai/go-gptscript"
)

type toolRequest struct {
	gptscript.Opts `json:",inline"`
	gptscript.Tool `json:",inline"`
}

type fileRequest struct {
	gptscript.Opts `json:",inline"`
	File           string `json:"file"`
	Input          string `json:"input"`
}
