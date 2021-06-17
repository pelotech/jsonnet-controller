package cmd

import (
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
	Use:   "build <PATH>",
	Short: "Evaluate what a given Konfiguration manifest would produce",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
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
		localAddr = fmt.Sprintf("https://127.0.0.1:%d/dry-run", ports[0].Local)
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
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer f.Close()

		r, err := http.NewRequest(http.MethodGet, localAddr, f)
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
			fmt.Println(errMap["error"])
			os.Exit(3)
		}

		fmt.Print(string(body))
		return nil
	},
}
