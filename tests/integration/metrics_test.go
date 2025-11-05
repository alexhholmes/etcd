// Copyright 2017 The etcd Authors
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

package integration

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/storage"
	"go.etcd.io/etcd/tests/v3/framework/integration"
)

// TestMetricDbSizeBoot checks that the db size metric is set on boot.
func TestMetricDbSizeBoot(t *testing.T) {
	integration.BeforeTest(t)
	clus := integration.NewCluster(t, &integration.ClusterConfig{Size: 1})
	defer clus.Terminate(t)

	v, err := clus.Members[0].Metric("etcd_debugging_mvcc_db_total_size_in_bytes")
	require.NoError(t, err)

	require.NotEqualf(t, "0", v, "expected non-zero, got %q", v)
}

func TestMetricQuotaBackendBytes(t *testing.T) {
	integration.BeforeTest(t)
	clus := integration.NewCluster(t, &integration.ClusterConfig{Size: 1})
	defer clus.Terminate(t)

	qs, err := clus.Members[0].Metric("etcd_server_quota_backend_bytes")
	require.NoError(t, err)
	qv, err := strconv.ParseFloat(qs, 64)
	require.NoError(t, err)
	require.Equalf(t, storage.DefaultQuotaBytes, int64(qv), "expected %d, got %f", storage.DefaultQuotaBytes, qv)
}

func TestMetricsHealth(t *testing.T) {
	integration.BeforeTest(t)
	clus := integration.NewCluster(t, &integration.ClusterConfig{Size: 1})
	defer clus.Terminate(t)

	tr, err := transport.NewTransport(transport.TLSInfo{}, 5*time.Second)
	require.NoError(t, err)
	u := clus.Members[0].ClientURLs[0]
	u.Path = "/health"
	resp, err := tr.RoundTrip(&http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		URL:    &u,
	})
	resp.Body.Close()
	require.NoError(t, err)

	hv, err := clus.Members[0].Metric("etcd_server_health_failures")
	require.NoError(t, err)
	require.Equalf(t, "0", hv, "expected '0' from etcd_server_health_failures, got %q", hv)
}

func TestMetricsRangeDurationSeconds(t *testing.T) {
	integration.BeforeTest(t)
	clus := integration.NewCluster(t, &integration.ClusterConfig{Size: 1})
	defer clus.Terminate(t)

	client := clus.RandClient()

	keys := []string{
		"my-namespace/foobar", "my-namespace/foobar1", "namespace/foobar1",
	}
	for _, key := range keys {
		_, err := client.Put(t.Context(), key, "data")
		require.NoError(t, err)
	}

	_, err := client.Get(t.Context(), "", clientv3.WithFromKey())
	require.NoError(t, err)

	rangeDurationSeconds, err := clus.Members[0].Metric("etcd_server_range_duration_seconds")
	require.NoError(t, err)

	require.NotEmptyf(t, rangeDurationSeconds, "expected a number from etcd_server_range_duration_seconds")

	rangeDuration, err := strconv.ParseFloat(rangeDurationSeconds, 64)
	require.NoErrorf(t, err, "failed to parse duration: %s", rangeDurationSeconds)

	maxRangeDuration := 600.0
	require.GreaterOrEqualf(t, rangeDuration, 0.0, "expected etcd_server_range_duration_seconds to be between 0 and %f, got %f", maxRangeDuration, rangeDuration)
	require.LessOrEqualf(t, rangeDuration, maxRangeDuration, "expected etcd_server_range_duration_seconds to be between 0 and %f, got %f", maxRangeDuration, rangeDuration)
}
