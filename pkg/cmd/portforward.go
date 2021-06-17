package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func forwardControllerPort(port string) (forwarder *portforward.PortForwarder, stopChan chan struct{}, err error) {
	var podList corev1.PodList
	if err = k8sClient.List(context.Background(), &podList, client.InNamespace("flux-system"), client.MatchingLabels{
		"app": "jsonnet-controller",
	}); err != nil {
		return
	}

	if len(podList.Items) == 0 {
		err = errors.New("could not locate the jsonnet-controller in the cluster")
		return
	}

	ctrlPod := podList.Items[0]

	roundTripper, upgrader, err := spdy.RoundTripperFor(restConfig)
	if err != nil {
		return
	}
	pfURL := &url.URL{
		Scheme: "https",
		Host:   strings.TrimLeft(restConfig.Host, "htps:/"),
		Path:   fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", ctrlPod.GetNamespace(), ctrlPod.GetName()),
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, pfURL)

	var readyChan chan struct{}
	stopChan, readyChan = make(chan struct{}, 1), make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)
	forwarder, err = portforward.New(dialer, []string{":" + port}, stopChan, readyChan, out, errOut)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range readyChan { // Block until the forward is ready
		}
		if len(errOut.String()) != 0 {
			err = errors.New(errOut.String())
		} else if len(out.String()) != 0 {
			// fmt.Println(out.String())
		}
	}()

	go func() {
		if serr := forwarder.ForwardPorts(); serr != nil {
			fmt.Fprintln(os.Stderr, serr.Error())
			os.Exit(2)
		}
	}()

	wg.Wait()
	return
}
