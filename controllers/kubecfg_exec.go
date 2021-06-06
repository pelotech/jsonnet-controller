package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
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
		return true, nil
	}

	return false, fmt.Errorf("Diff exited with non-zero/non-ten status %d, stderr: %s", exitErr.ProcessState.ExitCode(), string(exitErr.Stderr))
}

func runKubecfgUpdate(ctx context.Context, log logr.Logger, konfig *appsv1.Konfiguration, dryRun bool) error {
	cmdCtx, cancel := context.WithTimeout(ctx, konfig.GetTimeout().Duration)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "/kubecfg", konfig.ToUpdateArgs(dryRun)...)

	// capture full stderr
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	defer errPipe.Close()

	log.Info("Runing kubecfg update", "DryRun", dryRun, "Command", cmd.String())

	out, err := cmd.Output()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return err
		}
		log.Info(fmt.Sprintf("Process exited with a non-zero status of %d", exitErr.ProcessState.ExitCode()))
		stderr, err := ioutil.ReadAll(errPipe)
		if err != nil {
			return err
		}
		log.Info("Error executing command", "Stdout", out, "Stderr", string(stderr))
		return exitErr
	}

	log.Info("Process completed successfully", "Stdout", out)
	return nil
}
