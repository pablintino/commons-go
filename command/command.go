package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type PostModifierTrimOption int

const (
	PostModifierTrimRight PostModifierTrimOption = iota
	PostModifierTrimLeft
	PostModifierTrimBoth
)

type RunnablePostModifier interface {
	process(content string) (string, error)
}

type trimPostModifier struct {
	trimOpt  PostModifierTrimOption
	trimChar string
}

func NewTrimPostModifier(option PostModifierTrimOption, char string) *trimPostModifier {
	return &trimPostModifier{trimOpt: option, trimChar: char}
}

func (t *trimPostModifier) process(content string) (string, error) {

	switch t.trimOpt {
	case PostModifierTrimRight:
		return strings.TrimRight(content, t.trimChar), nil
	case PostModifierTrimLeft:
		return strings.TrimLeft(content, t.trimChar), nil
	case PostModifierTrimBoth:
		return strings.Trim(content, t.trimChar), nil
	default:
		return "", fmt.Errorf("unknown trim option: %d", t.trimOpt)
	}
}

type Runnable interface {
	Run() error
	RunStdout() ([]byte, error)
	RunStdoutStr(modifiers ...RunnablePostModifier) (string, error)
	RunCombined() ([]byte, error)
	RunCombinedStr() (string, error)

	RunToWriter(stdout io.Writer, stderr io.Writer) error
}

type commandRequest struct {
	ctx  context.Context
	cmd  string
	args []string
}

type execCommand struct {
	commandRequest
}

func (e *execCommand) Run() error {
	return exec.CommandContext(e.ctx, e.cmd, e.args...).Run()
}

func (e *execCommand) RunStdout() ([]byte, error) {
	return exec.CommandContext(e.ctx, e.cmd, e.args...).Output()
}

func (e *execCommand) RunStdoutStr(modifiers ...RunnablePostModifier) (string, error) {
	bytes, err := e.RunStdout()
	if err != nil {
		return "", err
	}
	result := string(bytes)
	for _, modifier := range modifiers {
		procRes, procErr := modifier.process(result)
		if procErr != nil {
			return result, errors.Join(err, procErr)
		}
		result = procRes
	}
	return result, nil
}

func (e *execCommand) RunCombinedStr() (string, error) {
	bytes, err := e.RunCombined()
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (e *execCommand) RunCombined() ([]byte, error) {
	return exec.CommandContext(e.ctx, e.cmd, e.args...).CombinedOutput()
}

func (e *execCommand) RunToWriter(stdout io.Writer, stderr io.Writer) error {
	cmd := exec.CommandContext(e.ctx, e.cmd, e.args...)
	if stdout != nil {
		cmd.Stdout = stdout
	}
	if stderr != nil {
		cmd.Stderr = stderr
	}
	return cmd.Run()
}

type CommandFactory interface {
	Command(ctx context.Context, cmd string, args ...string) Runnable
}

type execCmdFactory struct{}

func NewExecCmdFactory() CommandFactory { return &execCmdFactory{} }
func (*execCmdFactory) Command(ctx context.Context, cmd string, args ...string) Runnable {
	return &execCommand{commandRequest: commandRequest{ctx, cmd, args}}
}
