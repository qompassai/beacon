package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/exp/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/qompassai/beacon/dane"
	"github.com/qompassai/beacon/dkim"
	"github.com/qompassai/beacon/dmarc"
	"github.com/qompassai/beacon/dns"
	"github.com/qompassai/beacon/dnsbl"
	"github.com/qompassai/beacon/iprev"
	"github.com/qompassai/beacon/metrics"
	"github.com/qompassai/beacon/mlog"
	"github.com/qompassai/beacon/mtasts"
	"github.com/qompassai/beacon/smtpclient"
	"github.com/qompassai/beacon/spf"
	"github.com/qompassai/beacon/subjectpass"
	"github.com/qompassai/beacon/tlsrpt"
	"github.com/qompassai/beacon/updates"
)

var metricHTTPClient = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "beacon_httpclient_request_duration_seconds",
		Help:    "HTTP requests lookups.",
		Buckets: []float64{0.01, 0.05, 0.100, 0.5, 1, 5, 10, 20, 30},
	},
	[]string{
		"pkg",
		"method",
		"code",
		"result",
	},
)

// httpClientObserve tracks the result of an HTTP transaction in a metric, and
// logs the result.
func httpClientObserve(ctx context.Context, elog *slog.Logger, pkg, method string, statusCode int, err error, start time.Time) {
	log := mlog.New("metrics", elog)
	var result string
	switch {
	case err == nil:
		switch statusCode / 100 {
		case 2:
			result = "ok"
		case 4:
			result = "usererror"
		case 5:
			result = "servererror"
		default:
			result = "other"
		}
	case errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, context.DeadlineExceeded):
		result = "timeout"
	case errors.Is(err, context.Canceled):
		result = "canceled"
	default:
		result = "error"
	}
	metricHTTPClient.WithLabelValues(pkg, method, result, fmt.Sprintf("%d", statusCode)).Observe(float64(time.Since(start)) / float64(time.Second))
	log.Debugx("httpclient result", err,
		slog.String("pkg", pkg),
		slog.String("method", method),
		slog.Int("code", statusCode),
		slog.Duration("duration", time.Since(start)))
}

func init() {
	dane.MetricVerify = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "beacon_dane_verify_total",
			Help: "Total number of DANE verification attempts, including beacon_dane_verify_errors_total.",
		},
	)
	dane.MetricVerifyErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "beacon_dane_verify_errors_total",
			Help: "Total number of DANE verification failures, causing connections to fail.",
		},
	)

	dkim.MetricSign = counterVec{promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "beacon_dkim_sign_total",
			Help: "DKIM messages signings, label key is the type of key, rsa or ed25519.",
		},
		[]string{
			"key",
		},
	)}
	dkim.MetricVerify = histogramVec{
		promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "beacon_dkim_verify_duration_seconds",
				Help:    "DKIM verify, including lookup, duration and result.",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.100, 0.5, 1, 5, 10, 20},
			},
			[]string{
				"algorithm",
				"status",
			},
		),
	}

	dmarc.MetricVerify = histogramVec{promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "beacon_dmarc_verify_duration_seconds",
			Help:    "DMARC verify, including lookup, duration and result.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.100, 0.5, 1, 5, 10, 20},
		},
		[]string{
			"status",
			"reject", // yes/no
			"use",    // yes/no, if policy is used after random selection
		},
	)}
	dns.MetricLookup = histogramVec{
		promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "beacon_dns_lookup_duration_seconds",
				Help:    "DNS lookups.",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.100, 0.5, 1, 5, 10, 20, 30},
			},
			[]string{
				"pkg",
				"type",   // Lower-case Resolver method name without leading Lookup.
				"result", // ok, nxdomain, temporary, timeout, canceled, error
			},
		),
	}

	dnsbl.MetricLookup = histogramVec{promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "beacon_dnsbl_lookup_duration_seconds",
			Help:    "DNSBL lookup",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.100, 0.5, 1, 5, 10, 20},
		},
		[]string{
			"zone",
			"status",
		},
	)}

	iprev.MetricIPRev = histogramVec{promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "beacon_iprev_lookup_total",
			Help:    "Number of iprev lookups.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.100, 0.5, 1, 5, 10, 20, 30},
		},
		[]string{"status"},
	)}

	mtasts.MetricGet = histogramVec{promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "beacon_mtasts_get_duration_seconds",
			Help:    "MTA-STS get of policy, including lookup, duration and result.",
			Buckets: []float64{0.01, 0.05, 0.100, 0.5, 1, 5, 10, 20},
		},
		[]string{
			"result", // ok, lookuperror, fetcherror
		},
	)}
	mtasts.HTTPClientObserve = httpClientObserve

	smtpclient.MetricCommands = histogramVec{promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "beacon_smtpclient_command_duration_seconds",
			Help:    "SMTP client command duration and result codes in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.100, 0.5, 1, 5, 10, 20, 30, 60, 120},
		},
		[]string{
			"cmd",
			"code",
			"secode",
		},
	)}
	smtpclient.MetricTLSRequiredNoIgnored = counterVec{promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "beacon_smtpclient_tlsrequiredno_ignored_total",
			Help: "Connection attempts with TLS policy findings ignored due to message with TLS-Required: No header. Does not cover case where TLS certificate cannot be PKIX-verified.",
		},
		[]string{
			"ignored", // daneverification (no matching tlsa record)
		},
	)}
	smtpclient.MetricPanicInc = func() {
		metrics.PanicInc(metrics.Smtpclient)
	}

	spf.MetricVerify = histogramVec{promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "beacon_spf_verify_duration_seconds",
			Help:    "SPF verify, including lookup, duration and result.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.100, 0.5, 1, 5, 10, 20},
		},
		[]string{
			"status",
		},
	)}

	subjectpass.MetricGenerate = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "beacon_subjectpass_generate_total",
			Help: "Number of generated subjectpass challenges.",
		},
	)
	subjectpass.MetricVerify = counterVec{promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "beacon_subjectpass_verify_total",
			Help: "Number of subjectpass verifications.",
		},
		[]string{
			"result", // ok, fail
		},
	)}

	tlsrpt.MetricLookup = histogramVec{promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "beacon_tlsrpt_lookup_duration_seconds",
			Help:    "TLSRPT lookups with result.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.100, 0.5, 1, 5, 10, 20, 30},
		},
		[]string{"result"},
	)}

	updates.MetricLookup = histogramVec{promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "beacon_updates_lookup_duration_seconds",
			Help:    "Updates lookup with result.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.100, 0.5, 1, 5, 10, 20, 30},
		},
		[]string{"result"},
	)}
	updates.MetricFetchChangelog = histogramVec{promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "beacon_updates_fetchchangelog_duration_seconds",
			Help:    "Fetch changelog with result.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.100, 0.5, 1, 5, 10, 20, 30},
		},
		[]string{"result"},
	)}
}

type counterVec struct {
	*prometheus.CounterVec
}

func (m counterVec) IncLabels(labels ...string) {
	m.CounterVec.WithLabelValues(labels...).Inc()
}

type histogramVec struct {
	*prometheus.HistogramVec
}

func (m histogramVec) ObserveLabels(v float64, labels ...string) {
	m.HistogramVec.WithLabelValues(labels...).Observe(v)
}
