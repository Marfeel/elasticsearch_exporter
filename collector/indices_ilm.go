package collector

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// IndicesILM information struct
type IndicesILM struct {
	logger log.Logger
	client *http.Client
	url    *url.URL

	up                              prometheus.Gauge
	totalScrapes, jsonParseFailures prometheus.Counter
	indicesMetrics                  []*Indices
}

// NewIndicesILM defines Indices ILM Prometheus metrics
func NewIndicesILM(logger log.Logger, client *http.Client, url *url.URL) *IndicesILM {
	return &IndicesILM{
		logger: logger,
		client: client,
		url:    url,

		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "indices_ilm_errors", "up"),
			Help: "Was the last scrape of the ElasticSearch Indices ILM endpoint successful.",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: prometheus.BuildFQName(namespace, "indices_ilm_errors", "total_scrapes"),
			Help: "Current total ElasticSearch Indices ILM scrapes.",
		}),
		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: prometheus.BuildFQName(namespace, "indices_ilm_errors", "json_parse_failures"),
			Help: "Number of errors while parsing JSON.",
		}),

		indicesMetrics: []*Indices{
			{
				Opts: prometheus.GaugeOpts{
					Namespace:   namespace,
					Subsystem:   "indices_ilm_errors",
					Name:        "shards_docs",
					ConstLabels: nil,
					Help:        "Count of documents on this shard",
				},
				Value: func(data IndexStatsIndexShardsDetailResponse) float64 {
					return float64(data.Docs.Count)
				},
				Labels: []string{"index", "shard", "node"},
				LabelValues: func(indexName string, shardName string, data IndexStatsIndexShardsDetailResponse) prometheus.Labels {
					return prometheus.Labels{"index": indexName, "shard": shardName, "node": data.Routing.Node}
				},
			},
		},
	}
}

// Describe add Snapshots metrics descriptions
func (cs *IndicesILM) Describe(ch chan<- *prometheus.Desc) {
	ch <- cs.up.Desc()
	ch <- cs.totalScrapes.Desc()
	ch <- cs.jsonParseFailures.Desc()
}

func (cs *IndicesILM) getAndParseURL(u *url.URL, data interface{}) error {
	res, err := cs.client.Get(u.String())
	if err != nil {
		return fmt.Errorf("failed to get from %s://%s:%s%s: %s",
			u.Scheme, u.Hostname(), u.Port(), u.Path, err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			_ = level.Warn(cs.logger).Log(
				"msg", "failed to close http.Client",
				"err", err,
			)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	if err := json.NewDecoder(res.Body).Decode(data); err != nil {
		cs.jsonParseFailures.Inc()
		return err
	}
	return nil
}

func (cs *IndicesILM) fetchAndDecodeIndicesILM() (IndicesILMResponse, error) {

	u := *cs.url
	u.Path = path.Join(u.Path, "/_all/_ilm/explain?filter_path=indices.*.failed_step,indices.*.step_info.reason")
	var asr IndicesILMResponse
	err := cs.getAndParseURL(&u, &asr)
	if err != nil {
		return asr, err
	}

	return asr, err
}

// Collect gets all indices ILM metric values
func (cs *IndicesILM) Collect(ch chan<- prometheus.Metric) {

	cs.totalScrapes.Inc()
	defer func() {
		ch <- cs.up
		ch <- cs.totalScrapes
		ch <- cs.jsonParseFailures
	}()

	asr, err := cs.fetchAndDecodeIndicesILM()
	if err != nil {
		cs.up.Set(0)
		_ = level.Warn(cs.logger).Log(
			"msg", "failed to fetch and decode index ILM stats",
			"err", err,
		)
		return
	}
	cs.up.Set(1)

	// Index stats
	for indexName, indexStats := range asr.Indices {
		ch <- prometheus.MustNewConstMetric(
			indexStats.Desc,
			indexStats.Type,
			indexStats.Value(indexStats),
			indexStats.Labels(indexName)...,
		)
	}
	prometheus.NewGaugeVec(metric.Opts, metric.Labels).Collect(ch)
}
