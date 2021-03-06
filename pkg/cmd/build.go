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

package cmd

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/portforward"
)

func init() {
	rootCmd.AddCommand(buildCmd)
}

var forwarder *portforward.PortForwarder
var stopChan chan struct{}
var localAddr string

var buildCmd = &cobra.Command{
	Use:   "build [PATH]",
	Short: "Evaluate what a given Konfiguration manifest would produce from the controller",
	Args:  cobra.MaximumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := checkClient(); err != nil {
			return err
		}
		var err error
		forwarder, stopChan, err = forwardControllerPort("9443")
		if err != nil {
			return err
		}
		ports, err := forwarder.GetPorts()
		if err != nil {
			stopChan <- struct{}{}
			return err
		}
		localAddr = fmt.Sprintf("https://127.0.0.1:%d/build", ports[0].Local)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		defer func() { stopChan <- struct{}{} }()
		httpClient := http.DefaultClient
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		if len(args) == 0 || args[0] == "-" {
			stat, err := os.Stdin.Stat()
			if err != nil {
				return err
			}
			if stat.Mode()&os.ModeNamedPipe == 0 {
				fmt.Fprintln(os.Stderr, "(reading from stdin)")
			}
			args = []string{os.Stdin.Name()}
		}

		data, err := ioutil.ReadFile(args[0])
		if err != nil {
			return err
		}

		r, err := http.NewRequest(http.MethodGet, localAddr, bytes.NewBuffer(data))
		if err != nil {
			return err
		}

		res, err := httpClient.Do(r)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		if res.StatusCode != http.StatusOK {
			var errMap map[string]string
			if err := json.Unmarshal(body, &errMap); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "ERROR: ", errMap["error"])
			os.Exit(3)
		}

		fmt.Print(string(body))
		return nil
	},
}
