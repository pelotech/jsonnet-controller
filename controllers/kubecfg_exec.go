/*
Copyright 2021 Avi Zimmerman.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/go-logr/logr"
	appsv1 "github.com/pelotech/kubecfg-operator/api/v1"
)

func runKubecfgDiff(ctx context.Context, log logr.Logger, konfig *appsv1.Konfiguration, path string) (updateRequired bool, err error) {
	cmdCtx, cancel := context.WithTimeout(ctx, konfig.GetTimeout())
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "/kubecfg", konfig.ToDiffArgs(path)...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	log.Info("Running diff compare", "Command", cmd.String())
	err = cmd.Run()

	// no changes required
	if err == nil {
		log.Info("Diff compare exited zero. No changes necessary.")
		return false, nil
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return false, err
	}

	// 10 signifies clean diff with update required
	if exitErr.ProcessState.ExitCode() == 10 {
		log.Info("Diff compare exited 10 - Update required")
		return true, nil
	}

	return false, fmt.Errorf("Diff exited with non-zero/non-ten status %d, stdout: %s : stderr: %s", exitErr.ProcessState.ExitCode(), outBuf.String(), errBuf.String())
}

func runKubecfgUpdate(ctx context.Context, log logr.Logger, konfig *appsv1.Konfiguration, path string, dryRun bool) error {
	cmdCtx, cancel := context.WithTimeout(ctx, konfig.GetTimeout())
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "/kubecfg", konfig.ToUpdateArgs(path, dryRun)...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if dryRun {
		log.Info("Runing kubecfg dry-run update", "Command", cmd.String())
	} else {
		log.Info("Runing kubecfg update", "Command", cmd.String())
	}

	err := cmd.Run()

	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return err
		}
		log.Info(fmt.Sprintf("Process exited with a non-zero status of %d", exitErr.ProcessState.ExitCode()))
		log.Info("Error executing command", "Stdout", stdoutBuf.String(), "Stderr", sanitizeStderr(&stderrBuf))
		return exitErr
	}

	log.Info("Process completed successfully", "Stdout", stdoutBuf.String(), "Stderr", sanitizeStderr(&stderrBuf))
	return nil
}

func sanitizeStderr(buf *bytes.Buffer) string {
	scanner := bufio.NewScanner(buf)
	lines := make([]string, 0)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			continue
		}
		// warnings generated when pruning is enabled are too much...
		if strings.Contains(text, "warnings.go") {
			continue
		}
		lines = append(lines, text)
	}
	return strings.Join(lines, "\n")
}
