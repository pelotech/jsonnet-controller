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

// Package healthcheck contains an interface for assessing the health of
// resources configured in a CR.
package healthcheck

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fluxcd/pkg/apis/meta"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/aggregator"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/collector"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HealthChecker interface {
	client.Object

	GetTimeout() time.Duration
	GetHealthChecks() []meta.NamespacedObjectKindReference
}

// HealthCheck is able to assess whether the configured health checks
// for a CR are all ready.
type HealthCheck struct {
	parent       HealthChecker
	statusPoller *polling.StatusPoller
}

// NewHealthCheck returns a new HealthCheck able to assess the given CR.
func NewHealthCheck(parent HealthChecker, statusPoller *polling.StatusPoller) *HealthCheck {
	return &HealthCheck{
		parent:       parent,
		statusPoller: statusPoller,
	}
}

// Assess will check if the configured health checks are all ready. It will poll
// at the given pollInterval.
func (hc *HealthCheck) Assess(pollInterval time.Duration) error {
	objMetadata, err := hc.toObjMetadata(hc.parent.GetHealthChecks())
	if err != nil {
		return err
	}

	timeout := hc.parent.GetTimeout() + (time.Second * 1)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	opts := polling.Options{PollInterval: pollInterval, UseCache: true}
	eventsChan := hc.statusPoller.Poll(ctx, objMetadata, opts)
	coll := collector.NewResourceStatusCollector(objMetadata)
	done := coll.ListenWithObserver(eventsChan, collector.ObserverFunc(
		func(statusCollector *collector.ResourceStatusCollector, e event.Event) {
			var rss []*event.ResourceStatus
			for _, rs := range statusCollector.ResourceStatuses {
				rss = append(rss, rs)
			}
			desired := status.CurrentStatus
			aggStatus := aggregator.AggregateStatus(rss, desired)
			if aggStatus == desired {
				cancel()
				return
			}
		}),
	)

	<-done

	if coll.Error != nil {
		return coll.Error
	}

	if ctx.Err() == context.DeadlineExceeded {
		ids := []string{}
		for _, rs := range coll.ResourceStatuses {
			if rs.Status != status.CurrentStatus {
				id := hc.objMetadataToString(rs.Identifier)
				ids = append(ids, id)
			}
		}
		return fmt.Errorf("health check timed out for [%v]", strings.Join(ids, ", "))
	}

	return nil
}

func (hc *HealthCheck) toObjMetadata(cr []meta.NamespacedObjectKindReference) ([]object.ObjMetadata, error) {
	oo := []object.ObjMetadata{}
	for _, c := range cr {
		// For backwards compatibility
		if c.APIVersion == "" {
			c.APIVersion = "apps/v1"
		}

		gv, err := schema.ParseGroupVersion(c.APIVersion)
		if err != nil {
			return []object.ObjMetadata{}, err
		}

		gk := schema.GroupKind{Group: gv.Group, Kind: c.Kind}
		o, err := object.CreateObjMetadata(c.Namespace, c.Name, gk)
		if err != nil {
			return []object.ObjMetadata{}, err
		}

		oo = append(oo, o)
	}
	return oo, nil
}

func (hc *HealthCheck) objMetadataToString(om object.ObjMetadata) string {
	return fmt.Sprintf("%s '%s/%s'", om.GroupKind.Kind, om.Namespace, om.Name)
}
