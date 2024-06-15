// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package opentelemetry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"code.gitea.io/gitea/modules/log"

	"github.com/go-logr/logr/funcr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func Setup(ctx context.Context) (shutdown func(context.Context)) {
	// Redirect otel logger to write to common forgejo log at info
	logWrap := funcr.New(func(prefix, args string) {
		log.Info(fmt.Sprint(prefix, args))
	}, funcr.Options{})
	otel.SetLogger(logWrap)
	// Redirect error handling to forgejo log as well
	otel.SetErrorHandler(otelErrorHandler{})

	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
	}

	otel.SetTextMapPropagator(newPropagator())

	res, err := newResource(ctx)
	if err != nil {
		return shutdown
	}

	traceShutdown, err := setupTraceProvider(ctx, res)
	if err != nil {
		log.Warn("OpenTelemetry trace setup failed, shutting trace exporter down, err=%s", err)
		if err := traceShutdown(ctx); err != nil {
			log.Warn("OpenTelemetry trace exporter shutdown failed, err=%s", err)
		}
	}

	shutdownFuncs = append(shutdownFuncs, traceShutdown)
	return shutdown
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

type otelErrorHandler struct{}

func (o otelErrorHandler) Handle(err error) {
	log.Error("internal opentelemetry error was raised: %s", err)
}

func createCertPool(certBytes []byte) (*x509.CertPool, error) {
	cp := x509.NewCertPool()
	if ok := cp.AppendCertsFromPEM(certBytes); !ok {
		return nil, errors.New("failed to append certificate to the cert pool")
	}
	return cp, nil
}

func withCertPool(path string, fn func(*x509.CertPool)) {
	if path == "" {
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		log.Warn("Otel: reading ca cert failed path=%s, err=%s", path, err)
		return
	}
	c, err := createCertPool(b)
	if err != nil {
		log.Warn("Otel: create cert pool failed")
		return
	}
	fn(c)
}

func WithClientCert(nc, nk string, fn func(tls.Certificate)) {
	if nc == "" || nk == "" {
		return
	}

	cert, err := os.ReadFile(nc)
	if err != nil {
		log.Warn("Otel: read tls client cert path=%s, err=%s", nc, err)
		return
	}
	key, err := os.ReadFile(nk)
	if err != nil {
		log.Warn("Otel: read tls client key path=%s, err=%s", nk, err)
		return
	}
	crt, err := tls.X509KeyPair(cert, key)
	if err != nil {
		log.Warn("Otel: create tls client key pair failed")
		return
	}

	fn(crt)
}
