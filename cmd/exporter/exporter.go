package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"io"

	fcgiclient "github.com/tomasen/fcgi_client"
)

const (
	namespace = "opcache"
)

func newMetric(metricName, metricDesc string, fcgiURI string) *prometheus.Desc {
	labels := prometheus.Labels{"fcgi_uri": fcgiURI}
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "", metricName), metricDesc, nil, labels)
}

func boolMetric(value bool) float64 {
	return map[bool]float64{true: 1, false: 0}[value]
}

func intMetric(value int64) float64 {
	return float64(value)
}

// Exporter collects OPcache status from the given FastCGI URI and exports them using
// the prometheus metrics package.
type Exporter struct {
	mutex sync.RWMutex

	uri        *url.URL
	scriptPath string

	enabledDesc                            *prometheus.Desc
	cacheFullDesc                          *prometheus.Desc
	restartPendingDesc                     *prometheus.Desc
	restartInProgressDesc                  *prometheus.Desc
	memoryUsageUsedMemoryDesc              *prometheus.Desc
	memoryUsageFreeMemoryDesc              *prometheus.Desc
	memoryUsageWastedMemoryDesc            *prometheus.Desc
	memoryUsageCurrentWastedPercentageDesc *prometheus.Desc
	internedStringsUsageBufferSizeDesc     *prometheus.Desc
	internedStringsUsageUsedMemoryDesc     *prometheus.Desc
	internedStringsUsageUsedFreeMemory     *prometheus.Desc
	internedStringsUsageUsedNumerOfStrings *prometheus.Desc
	statisticsNumCachedScripts             *prometheus.Desc
	statisticsNumCachedKeys                *prometheus.Desc
	statisticsMaxCachedKeys                *prometheus.Desc
	statisticsHits                         *prometheus.Desc
	statisticsStartTime                    *prometheus.Desc
	statisticsLastRestartTime              *prometheus.Desc
	statisticsOOMRestarts                  *prometheus.Desc
	statisticsHashRestarts                 *prometheus.Desc
	statisticsManualRestarts               *prometheus.Desc
	statisticsMisses                       *prometheus.Desc
	statisticsBlacklistMisses              *prometheus.Desc
	statisticsBlacklistMissRatio           *prometheus.Desc
	statisticsHitRate                      *prometheus.Desc
}

// NewExporter returns an initialized Exporter.
func NewExporter(rawUri string, scriptPath string) (*Exporter, error) {
	// fallback for old default value
	if !strings.Contains(rawUri, "://") {
		rawUri = "tcp://" + rawUri
	}
	parsedUri, err := url.Parse(rawUri)

	exporter := &Exporter{
		uri:        parsedUri,
		scriptPath: scriptPath,

		enabledDesc:           newMetric("enabled", "Is OPcache enabled.", rawUri),
		cacheFullDesc:         newMetric("cache_full", "Is OPcache full.", rawUri),
		restartPendingDesc:    newMetric("restart_pending", "Is restart pending.", rawUri),
		restartInProgressDesc: newMetric("restart_in_progress", "Is restart in progress.", rawUri),

		memoryUsageUsedMemoryDesc:              newMetric("memory_usage_used_memory", "OPcache used memory.", rawUri),
		memoryUsageFreeMemoryDesc:              newMetric("memory_usage_free_memory", "OPcache free memory.", rawUri),
		memoryUsageWastedMemoryDesc:            newMetric("memory_usage_wasted_memory", "OPcache wasted memory.", rawUri),
		memoryUsageCurrentWastedPercentageDesc: newMetric("memory_usage_current_wasted_percentage", "OPcache current wasted percentage.", rawUri),

		internedStringsUsageBufferSizeDesc:     newMetric("interned_strings_usage_buffer_size", "OPcache interned string buffer size.", rawUri),
		internedStringsUsageUsedMemoryDesc:     newMetric("interned_strings_usage_used_memory", "OPcache interned string used memory.", rawUri),
		internedStringsUsageUsedFreeMemory:     newMetric("interned_strings_usage_free_memory", "OPcache interned string free memory.", rawUri),
		internedStringsUsageUsedNumerOfStrings: newMetric("interned_strings_usage_number_of_strings", "OPcache interned string number of strings.", rawUri),

		statisticsNumCachedScripts:   newMetric("statistics_num_cached_scripts", "OPcache statistics, number of cached scripts.", rawUri),
		statisticsNumCachedKeys:      newMetric("statistics_num_cached_keys", "OPcache statistics, number of cached keys.", rawUri),
		statisticsMaxCachedKeys:      newMetric("statistics_max_cached_keys", "OPcache statistics, max cached keys.", rawUri),
		statisticsHits:               newMetric("statistics_hits", "OPcache statistics, hits.", rawUri),
		statisticsStartTime:          newMetric("statistics_start_time", "OPcache statistics, start time.", rawUri),
		statisticsLastRestartTime:    newMetric("statistics_last_restart_time", "OPcache statistics, last restart time", rawUri),
		statisticsOOMRestarts:        newMetric("statistics_oom_restarts", "OPcache statistics, oom restarts", rawUri),
		statisticsHashRestarts:       newMetric("statistics_hash_restarts", "OPcache statistics, hash restarts", rawUri),
		statisticsManualRestarts:     newMetric("statistics_manual_restarts", "OPcache statistics, manual restarts", rawUri),
		statisticsMisses:             newMetric("statistics_misses", "OPcache statistics, misses", rawUri),
		statisticsBlacklistMisses:    newMetric("statistics_blacklist_misses", "OPcache statistics, blacklist misses", rawUri),
		statisticsBlacklistMissRatio: newMetric("statistics_blacklist_miss_ratio", "OPcache statistics, blacklist miss ratio", rawUri),
		statisticsHitRate:            newMetric("statistics_hit_rate", "OPcache statistics, opcache hit rate", rawUri),
	}

	return exporter, err
}

// Describe describes all the metrics ever exported by the OPcache exporter.
// Implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.enabledDesc
	ch <- e.cacheFullDesc
	ch <- e.restartPendingDesc
	ch <- e.restartInProgressDesc
	ch <- e.memoryUsageUsedMemoryDesc
	ch <- e.memoryUsageFreeMemoryDesc
	ch <- e.memoryUsageWastedMemoryDesc
	ch <- e.memoryUsageCurrentWastedPercentageDesc
	ch <- e.internedStringsUsageBufferSizeDesc
	ch <- e.internedStringsUsageUsedMemoryDesc
	ch <- e.internedStringsUsageUsedFreeMemory
	ch <- e.internedStringsUsageUsedNumerOfStrings
	ch <- e.statisticsNumCachedScripts
	ch <- e.statisticsNumCachedKeys
	ch <- e.statisticsMaxCachedKeys
	ch <- e.statisticsHits
	ch <- e.statisticsStartTime
	ch <- e.statisticsLastRestartTime
	ch <- e.statisticsOOMRestarts
	ch <- e.statisticsHashRestarts
	ch <- e.statisticsManualRestarts
	ch <- e.statisticsMisses
	ch <- e.statisticsBlacklistMisses
	ch <- e.statisticsBlacklistMissRatio
	ch <- e.statisticsHitRate
}

// Collect collects metrics of OPcache stats.
// Implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()

	status, err := e.getOpcacheStatus()
	if err != nil {
		log.Print(err)
		status = new(OPcacheStatus)
	}

	ch <- prometheus.MustNewConstMetric(e.enabledDesc, prometheus.GaugeValue, boolMetric(status.OPcacheEnabled))
	ch <- prometheus.MustNewConstMetric(e.cacheFullDesc, prometheus.GaugeValue, boolMetric(status.CacheFull))
	ch <- prometheus.MustNewConstMetric(e.restartPendingDesc, prometheus.GaugeValue, boolMetric(status.RestartPending))
	ch <- prometheus.MustNewConstMetric(e.restartInProgressDesc, prometheus.GaugeValue, boolMetric(status.RestartInProgress))
	ch <- prometheus.MustNewConstMetric(e.memoryUsageUsedMemoryDesc, prometheus.GaugeValue, intMetric(status.MemoryUsage.UsedMemory))
	ch <- prometheus.MustNewConstMetric(e.memoryUsageFreeMemoryDesc, prometheus.GaugeValue, intMetric(status.MemoryUsage.FreeMemory))
	ch <- prometheus.MustNewConstMetric(e.memoryUsageWastedMemoryDesc, prometheus.GaugeValue, intMetric(status.MemoryUsage.WastedMemory))
	ch <- prometheus.MustNewConstMetric(e.memoryUsageCurrentWastedPercentageDesc, prometheus.GaugeValue, status.MemoryUsage.CurrentWastedPercentage)
	ch <- prometheus.MustNewConstMetric(e.internedStringsUsageBufferSizeDesc, prometheus.GaugeValue, intMetric(status.InternedStringsUsage.BufferSize))
	ch <- prometheus.MustNewConstMetric(e.internedStringsUsageUsedMemoryDesc, prometheus.GaugeValue, intMetric(status.InternedStringsUsage.UsedMemory))
	ch <- prometheus.MustNewConstMetric(e.internedStringsUsageUsedFreeMemory, prometheus.GaugeValue, intMetric(status.InternedStringsUsage.FreeMemory))
	ch <- prometheus.MustNewConstMetric(e.statisticsNumCachedScripts, prometheus.GaugeValue, intMetric(status.OPcacheStatistics.NumCachedScripts))
	ch <- prometheus.MustNewConstMetric(e.statisticsNumCachedKeys, prometheus.GaugeValue, intMetric(status.OPcacheStatistics.NumCachedKeys))
	ch <- prometheus.MustNewConstMetric(e.statisticsMaxCachedKeys, prometheus.GaugeValue, intMetric(status.OPcacheStatistics.MaxCachedKeys))
	ch <- prometheus.MustNewConstMetric(e.statisticsHits, prometheus.GaugeValue, intMetric(status.OPcacheStatistics.Hits))
	ch <- prometheus.MustNewConstMetric(e.statisticsStartTime, prometheus.GaugeValue, intMetric(status.OPcacheStatistics.StartTime))
	ch <- prometheus.MustNewConstMetric(e.statisticsLastRestartTime, prometheus.GaugeValue, intMetric(status.OPcacheStatistics.LastRestartTime))
	ch <- prometheus.MustNewConstMetric(e.statisticsOOMRestarts, prometheus.GaugeValue, intMetric(status.OPcacheStatistics.OOMRestarts))
	ch <- prometheus.MustNewConstMetric(e.statisticsHashRestarts, prometheus.GaugeValue, intMetric(status.OPcacheStatistics.HashRestarts))
	ch <- prometheus.MustNewConstMetric(e.statisticsManualRestarts, prometheus.GaugeValue, intMetric(status.OPcacheStatistics.ManualRestarts))
	ch <- prometheus.MustNewConstMetric(e.statisticsMisses, prometheus.GaugeValue, intMetric(status.OPcacheStatistics.Misses))
	ch <- prometheus.MustNewConstMetric(e.statisticsBlacklistMisses, prometheus.GaugeValue, intMetric(status.OPcacheStatistics.BlacklistMisses))
	ch <- prometheus.MustNewConstMetric(e.statisticsBlacklistMissRatio, prometheus.GaugeValue, status.OPcacheStatistics.BlacklistMissRatio)
	ch <- prometheus.MustNewConstMetric(e.statisticsHitRate, prometheus.GaugeValue, status.OPcacheStatistics.OPcacheHitRate)
}

func (e *Exporter) getOpcacheStatus() (*OPcacheStatus, error) {
	host := e.uri.Host
	if e.uri.Scheme == "unix" {
		host = e.uri.Path
	}

	client, err := fcgiclient.Dial(e.uri.Scheme, host)
	if err != nil {
		return nil, err
	}

	env := map[string]string{
		"SCRIPT_FILENAME": e.scriptPath,
	}

	resp, err := client.Get(env)
	if err != nil {
		return nil, err
	}

	content, err := io.ReadAll(io.Reader(resp.Body))
	if err != nil {
		return nil, err
	}

	status := new(OPcacheStatus)
	err = json.Unmarshal(content, status)
	if err != nil {
		return nil, errors.New(string(content))
	}

	return status, nil
}
