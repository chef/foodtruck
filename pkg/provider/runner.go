package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

type Runner interface {
	Run(ctx context.Context, providerName string, spec json.RawMessage) error
}

type ExecRunner struct {
}

func NewExecRunner() *ExecRunner {
	return &ExecRunner{}
}

func (p *ExecRunner) Run(ctx context.Context, providerName string, spec json.RawMessage) error {
	if err := validateProviderName(providerName); err != nil {
		return err
	}

	execPath, err := exec.LookPath(fmt.Sprintf("foodtruck-provider-%s", providerName))
	if err != nil {
		return err
	}

	cmd := exec.Command(execPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		io.Copy(stdin, bytes.NewReader(spec))
	}()

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func validateProviderName(providerName string) error {
	return nil
}
