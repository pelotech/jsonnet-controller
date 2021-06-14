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

package v1

func (k *Konfiguration) newArgs(cmd string) []string {
	args := []string{cmd, "--cache-dir", "/cache", "--namespace", k.GetNamespace()}

	// Add any global arguments provided by the user.
	if globalArgs := k.GetKubecfgArgs(); len(globalArgs) != 0 {
		args = append(args, globalArgs...)
	}

	return args
}

func (k *Konfiguration) varsArgs(cmd string) []string {
	args := k.newArgs(cmd)
	if vars := k.GetVariables(); vars != nil {
		args = vars.AppendToArgs(args)
	}
	return args
}

// ToShowArgs convert this Konfiguration to show arguments.
func (k *Konfiguration) ToShowArgs(path string) []string {
	args := k.varsArgs("show")
	args = append(args, path)
	return args
}
