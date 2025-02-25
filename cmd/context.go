// Copyright (c) 2021 Palantir Technologies. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"io"
	"os"

	"github.com/palantir/pkg/signals"
	"github.com/palantir/pkg/uuid"
	"github.com/palantir/witchcraft-go-logging/wlog"
	wlogtmpl "github.com/palantir/witchcraft-go-logging/wlog-tmpl"
	"github.com/palantir/witchcraft-go-logging/wlog/metriclog/metric1log"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
)

// contextWithDefaultLogger creates a context wired up with a multiwriter to write to  standard out
//
// The function returned by contextWithDefaultLogger must be called on application shutdown,
// this will cancel the context and close the file logger.
func contextWithDefaultLogger() (context.Context, func() error) {
	wlog.SetDefaultLoggerProvider(wlogtmpl.LoggerProvider(nil))
	ctx := context.Background()
	ctx = svc1log.WithLogger(ctx, svc1log.New(os.Stdout, wlog.InfoLevel,
		svc1log.OriginFromCallLineWithSkip(3),
		svc1log.SafeParam("runID", uuid.NewUUID())))
	ctx = metric1log.WithLogger(ctx, metric1log.New(io.Discard))
	withShutdown, cancel := signals.ContextWithShutdown(ctx)
	return withShutdown, func() error {
		cancel()
		return nil
	}
}
