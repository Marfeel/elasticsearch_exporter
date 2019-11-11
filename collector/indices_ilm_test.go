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

func TestIndicesILM(t *testing.T) {
	// Testcases created using:
	// curl http://localhost:9200/_all/_ilm/explain?filter_path=indices.*.failed_step,indices.*.step_info.reason

	tcs := map[string]string{
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
			c := NewIndicesILM(log.NewNopLogger(), http.DefaultClient, u)
			nsr, err := c.fetchAndDecodeIndicesILM()
			if err != nil {
				t.Fatalf("Failed to fetch or decode indices ILM: %s", err)
			}
			t.Logf("[%s/%s] All Indices ILM Response: %+v", hn, ver, nsr)
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
