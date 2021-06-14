/*
Copyright 2021 Pelotech.

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

	"sigs.k8s.io/controller-runtime/pkg/log"

	konfigurationv1 "github.com/pelotech/kubecfg-operator/api/v1"
)

func runKubecfgShow(ctx context.Context, konfig *konfigurationv1.Konfiguration, path string) ([]byte, error) {
	log := log.FromContext(ctx)

	cmdCtx, cancel := context.WithTimeout(ctx, konfig.GetTimeout())
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "/kubecfg", konfig.ToShowArgs(path)...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	log.Info("Running kubecfg show", "Command", cmd.String())
	err := cmd.Run()

	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return nil, err
		}
		log.Info(fmt.Sprintf("Process exited with a non-zero status of %d", exitErr.ProcessState.ExitCode()))
		log.Info("Error executing command", "Stdout", outBuf.String(), "Stderr", sanitizeStderr(&errBuf))
		return nil, fmt.Errorf("kubecfg show failed: %s", exitErr.Error())
	}

	return outBuf.Bytes(), nil
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
