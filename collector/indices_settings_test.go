package collector

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-kit/kit/log"
)

func TestIndicesSettings(t *testing.T) {
	// Testcases created using:
	// curl http://localhost:9200/_all/_settings

	tcs := map[string]string{
		//"6.5.4": `{"viber":{"settings":{"index":{"creation_date":"1548066996192","number_of_shards":"5","number_of_replicas":"1","uuid":"kt2cGV-yQRaloESpqj2zsg","version":{"created":"6050499"},"provided_name":"viber"}}},"facebook":{"settings":{"index":{"creation_date":"1548066984670","number_of_shards":"5","number_of_replicas":"1","uuid":"jrU8OWQZQD--9v5eg0tjbg","version":{"created":"6050499"},"provided_name":"facebook"}}},"twitter":{"settings":{"index":{"number_of_shards":"5","blocks":{"read_only_allow_delete":"true"},"provided_name":"twitter","creation_date":"1548066697559","number_of_replicas":"1","uuid":"-sqtc4fVRrS2jHJCZ2hQ9Q","version":{"created":"6050499"}}}},"instagram":{"settings":{"index":{"number_of_shards":"5","blocks":{"read_only_allow_delete":"true"},"provided_name":"instagram","creation_date":"1548066991932","number_of_replicas":"1","uuid":"WeGWaxa_S3KrgE5SZHolTw","version":{"created":"6050499"}}}}}`,
		"7.5.4": `{"indices":{"mrf_fastly-20191025-000328":{"failed_step":"shrink","step_info":{"reason":"index mrf_fastly-20191025-000328 must have all shards allocated on the same node to shrink index"}},"mrf_fastly-20191026-000343":{"failed_step":"shrink","step_info":{"reason":"index mrf_fastly-20191026-000343 must have all shards allocated on the same node to shrink index"}},"mrf_fastly-20191028-000364":{"failed_step":"shrink","step_info":{"reason":"index mrf_fastly-20191028-000364 must have all shards allocated on the same node to shrink index"}}}}`,
	}
	for ver, out := range tcs {
		for hn, handler := range map[string]http.Handler{
			"plain": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, out)
			}),
		} {
			ts := httptest.NewServer(handler)
			defer ts.Close()

			u, err := url.Parse(ts.URL)
			if err != nil {
				t.Fatalf("Failed to parse URL: %s", err)
			}
			c := NewIndicesSettings(log.NewNopLogger(), http.DefaultClient, u)
			nsr, err := c.fetchAndDecodeIndicesSettings()
			if err != nil {
				t.Fatalf("Failed to fetch or decode indices settings: %s", err)
			}
			t.Logf("[%s/%s] All Indices Settings Response: %+v", hn, ver, nsr)
			// if nsr.Cluster.Routing.Allocation.Enabled != "ALL" {
			// 	t.Errorf("Wrong setting for cluster routing allocation enabled")
			// }
			var counter int
			for key, value := range nsr.Indices {
				counter++
				if key == "" {
					t.Errorf("No index detected")
				}

				if value.FailedStep != "shrink" {
					t.Errorf("Error detecting failed step")
				}

				if value.StepInfo.Reason != "" && !strings.Contains(value.StepInfo.Reason, "all shards allocated on the same node to shrink index") {
					t.Errorf("Error detecting step failing reason")
				}
				/*t.Errorf(key)
				t.Errorf(value.FailedStep)
				t.Errorf(value.StepInfo.Reason)*/
			}
			if counter != 3 {
				t.Errorf("Wrong number of indexes")
			}
		}
	}
}
