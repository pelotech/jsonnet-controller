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

package v1

import "fmt"

func (k *Konfiguration) newArgs(cmd string) []string {
	args := []string{cmd, "--cache-dir", "/cache", "--namespace", k.GetNamespace()}

	// Add any global arguments provided by the user.
	if globalArgs := k.GetKubecfgArgs(); len(globalArgs) != 0 {
		args = append(args, globalArgs...)
	}

	return args
}

// ToUpdateArgs converts this Konfiguration schema into kubecfg update
// arguments.
func (k *Konfiguration) ToUpdateArgs(paths []string, dryRun bool) []string {
	args := k.newArgs("update")

	// Check if we are adding garbage collection flags.
	if k.GCEnabled() {
		gcTag := fmt.Sprintf("%s_%s", k.GetNamespace(), k.GetName())
		args = append(args, []string{"--gc-tag", gcTag}...)
	}

	// Check if disabling validation.
	if !k.ValidateEnabled() {
		args = append(args, "--validate=false")
	}

	// Check if defining external or top-level arguments.
	if vars := k.GetVariables(); vars != nil {
		args = vars.AppendToArgs(args)
	}

	if dryRun {
		args = append(args, "--dry-run")
	}

	// Finally add the paths
	args = append(args, paths...)

	return args
}

// ToDiffArgs converts this Konfiguration schema into kubecfg diff arguments.
func (k *Konfiguration) ToDiffArgs(paths []string) []string {
	args := k.newArgs("diff")
	// Check if defining external or top-level arguments.
	if vars := k.GetVariables(); vars != nil {
		args = vars.AppendToArgs(args)
	}
	// Append the diff strategy
	args = append(args, []string{"--diff-strategy", k.GetDiffStrategy()}...)
	// Finally add the paths
	args = append(args, paths...)
	return args
}
