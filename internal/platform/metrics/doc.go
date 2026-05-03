// Package metrics owns the Prometheus registry and the canonical counters /
// histograms exposed by every binary mode. Each metric is namespaced
// `supremacy_<mode>_<name>` so dashboards can group by mode without
// guesswork.
package metrics
