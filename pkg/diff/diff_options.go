// https://github.com/argoproj/gitops-engine/blob/master/pkg/diff/diff_options.go
// Originally taken from argoproj gitops-engine (Copyright Apache 2.0)
// https://github.com/argoproj/gitops-engine
package diff

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Option func(*options)

// Holds diffing settings
type options struct {
	// If set to true then differences caused by aggregated roles in RBAC resources are ignored.
	ignoreAggregatedRoles bool
	normalizer            Normalizer
	log                   logr.Logger
}

func applyOptions(opts []Option) options {
	o := options{
		ignoreAggregatedRoles: false,
		normalizer:            GetNoopNormalizer(),
		log:                   zap.New(),
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func IgnoreAggregatedRoles(ignore bool) Option {
	return func(o *options) {
		o.ignoreAggregatedRoles = ignore
	}
}

func WithNormalizer(normalizer Normalizer) Option {
	return func(o *options) {
		o.normalizer = normalizer
	}
}

func WithLogr(log logr.Logger) Option {
	return func(o *options) {
		o.log = log
	}
}
