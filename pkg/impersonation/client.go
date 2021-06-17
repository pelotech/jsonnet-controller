package impersonation

import (
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client is an extension of the controller-runtime Client with the ability to retrieve
// a status poller using the same credentials.
type Client interface {
	client.Client

	// StatusPoller returns a polling.StatusPoller using the config from
	// this client instance.
	StatusPoller() *polling.StatusPoller
}

type clientWithPoller struct {
	client.Client
}

func (c *clientWithPoller) StatusPoller() *polling.StatusPoller {
	return polling.NewStatusPoller(c, c.RESTMapper())
}
