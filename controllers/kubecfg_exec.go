package controllers

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/go-logr/logr"
	appsv1 "github.com/tinyzimmer/kubecfg-operator/api/v1"
)

func runKubecfgDiff(ctx context.Context, log logr.Logger, konfig *appsv1.Konfiguration) (updateRequired bool, err error) {
	cmdCtx, cancel := context.WithTimeout(ctx, konfig.GetTimeout().Duration)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "/kubecfg", konfig.ToDiffArgs()...)

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

	return false, fmt.Errorf("Diff exited with non-zero/non-ten status %d, stderr: %s", exitErr.ProcessState.ExitCode(), string(exitErr.Stderr))
}

func runKubecfgUpdate(ctx context.Context, log logr.Logger, konfig *appsv1.Konfiguration, dryRun bool) error {
	cmdCtx, cancel := context.WithTimeout(ctx, konfig.GetTimeout().Duration)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "/kubecfg", konfig.ToUpdateArgs(dryRun)...)

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
		log.Info("Error executing command", "Stdout", stdoutBuf.String(), "Stderr", stderrBuf.String())
		return exitErr
	}

	log.Info("Process completed successfully", "Stdout", stdoutBuf.String())
	return nil
}
