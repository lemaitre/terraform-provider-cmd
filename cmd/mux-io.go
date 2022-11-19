package cmd

import (
	"bytes"
	"io"
)

type MultiWriter struct {
	Writers []io.Writer
}

func (muxed MultiWriter) Write(p []byte) (int, error) {
	towrite := len(p)
	for _, writer := range muxed.Writers {
		written := 0
		for written < towrite {
			n, err := writer.Write(p[written:])
			if err != nil {
				return 0, err
			}
			written += n
		}
	}
	return towrite, nil
}

type CommandOutput struct {
	Stdout       bytes.Buffer
	Stderr       bytes.Buffer
	Combined     bytes.Buffer
	StdoutWriter MultiWriter
	StderrWriter MultiWriter
}

func NewCommandOutput() *CommandOutput {
	var out CommandOutput
	out.StdoutWriter.Writers = []io.Writer{&out.Stdout, &out.Combined}
	out.StderrWriter.Writers = []io.Writer{&out.Stderr, &out.Combined}
	return &out
}
