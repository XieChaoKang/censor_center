package main

type Censor interface {
	CensorText(*CensorTextParams) error
	CensorImage(*CensorImageParams) error
}

type CensorTextParams struct {
	Text []string
}

type CensorImageParams struct {
	ImageUrl    []string
	ImageBase64 []string
}

var (
	DefaultCensor = map[string]Censor{}
)
