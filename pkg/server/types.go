package server

import "github.com/gptscript-ai/go-gptscript"

type toolRequest struct {
	gptscript.Opts       `json:",inline"`
	gptscript.SimpleTool `json:",inline"`
	gptscript.FreeForm   `json:",inline"`
}

type fileRequest struct {
	gptscript.Opts `json:",inline"`
	File           string `json:"file"`
	Input          string `json:"input"`
}

type documentRequest struct {
	gptscript.Opts     `json:",inline"`
	gptscript.Document `json:",inline"`
}
