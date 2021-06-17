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
