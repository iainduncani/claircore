package pgxpoolprometheus

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

type mockStater struct {
	stats pgxStat
}

func (m *mockStater) Stat() pgxStat {
	return m.stats
}

var (
	_ pgxStat = (*pgxStatMock)(nil)
)

type pgxStatMock struct {
	acquireCount         int64
	acquireDuration      time.Duration
	acquiredConns        int32
	canceledAcquireCount int64
	constructingConns    int32
	emptyAcquireCount    int64
	idleConns            int32
	maxConns             int32
	totalConns           int32
}

func (m *pgxStatMock) AcquireCount() int64 {
	return m.acquireCount
}
func (m *pgxStatMock) AcquireDuration() time.Duration {
	return m.acquireDuration
}
func (m *pgxStatMock) AcquiredConns() int32 {
	return m.acquiredConns
}
func (m *pgxStatMock) CanceledAcquireCount() int64 {
	return m.canceledAcquireCount
}
func (m *pgxStatMock) ConstructingConns() int32 {
	return m.constructingConns
}
func (m *pgxStatMock) EmptyAcquireCount() int64 {
	return m.emptyAcquireCount
}
func (m *pgxStatMock) IdleConns() int32 {
	return m.idleConns
}
func (m *pgxStatMock) MaxConns() int32 {
	return m.maxConns
}
func (m *pgxStatMock) TotalConns() int32 {
	return m.totalConns
}

func TestDescribeDescribesAllAvailableStats(t *testing.T) {
	labelName := "testLabel"
	labelValue := "testLabelValue"
	labels := map[string]string{labelName: labelValue}
	expectedDescriptorCount := 9
	timeout := time.After(time.Second * 5)
	stater := &mockStater{&pgxStatMock{}}
	statFn := func() pgxStat { return stater.Stat() }
	testObject := newCollector(statFn, labels)

	ch := make(chan *prometheus.Desc)
	go testObject.Describe(ch)

	expectedDescriptorCountRemaining := expectedDescriptorCount
	uniqueDescriptors := make(map[string]struct{})
	for {
		if expectedDescriptorCountRemaining == 0 {
			break
		}
		select {
		case desc := <-ch:
			if !strings.Contains(desc.String(), labelName) {
				t.Errorf("Expecting the description '%s' to contain the label name '%s' but it does not", desc.String(), labelName)
			}
			if !strings.Contains(desc.String(), labelValue) {
				t.Errorf("Expecting the description '%s' to contain the label name '%s' but it does not", desc.String(), labelValue)
			}
			uniqueDescriptors[desc.String()] = struct{}{}
			expectedDescriptorCountRemaining--
		case <-timeout:
			t.Fatalf("Test timed out while there were still %d descriptors expected", expectedDescriptorCountRemaining)
		}
	}
	if expectedDescriptorCountRemaining != 0 {
		t.Errorf("Expected all descriptors to be found but %d was not", expectedDescriptorCountRemaining)
	}
	if len(uniqueDescriptors) != expectedDescriptorCount {
		t.Errorf("Expected %d descriptors to be registered but there were %d", expectedDescriptorCount, len(uniqueDescriptors))
	}
}

func TestCollectCollectsAllAvailableStats(t *testing.T) {
	expectedMetricValues := map[string]float64{
		"pgxpool_acquire_count":          float64(1),
		"pgxpool_acquire_duration_ns":    float64(2e+09),
		"pgxpool_acquired_conns":         float64(3),
		"pgxpool_canceled_acquire_count": float64(4),
		"pgxpool_constructing_conns":     float64(5),
		"pgxpool_empty_acquire":          float64(6),
		"pgxpool_idle_conns":             float64(7),
		"pgxpool_max_conns":              float64(8),
		"pgxpool_total_conns":            float64(9),
	}

	mockStats := &pgxStatMock{
		acquireCount:         int64(1),
		acquireDuration:      time.Second * 2,
		acquiredConns:        int32(3),
		canceledAcquireCount: int64(4),
		constructingConns:    int32(5),
		emptyAcquireCount:    int64(6),
		idleConns:            int32(7),
		maxConns:             int32(8),
		totalConns:           int32(9),
	}
	expectedMetricCount := 9
	timeout := time.After(time.Second * 5)
	stater := &mockStater{mockStats}
	staterfn := func() pgxStat { return stater.Stat() }
	testObject := newCollector(staterfn, nil)

	ch := make(chan prometheus.Metric)
	go testObject.Collect(ch)

	expectedMetricCountRemaining := expectedMetricCount
	for {
		if expectedMetricCountRemaining == 0 {
			break
		}
		select {
		case metric := <-ch:
			pb := &dto.Metric{}
			metric.Write(pb)
			description := metric.Desc().String()
			metricExpected := false
			for expectedMetricName, expectedMetricValue := range expectedMetricValues {
				if strings.Contains(description, expectedMetricName) {
					var value float64
					if pb.GetCounter() != nil {
						value = *pb.GetCounter().Value
					}
					if pb.GetGauge() != nil {
						value = *pb.GetGauge().Value
					}
					if value != expectedMetricValue {
						t.Errorf("Expected the '%s' metric to be %g but was %g", expectedMetricName, expectedMetricValue, value)
					}
					metricExpected = true
					break
				}
			}
			if !metricExpected {
				t.Errorf("Unexpected description: %s", description)
			}
			expectedMetricCountRemaining--
		case <-timeout:
			t.Fatalf("Test timed out while there were still %d descriptors expected", expectedMetricCountRemaining)
		}
	}
	if expectedMetricCountRemaining != 0 {
		t.Errorf("Expected all metrics to be found but %d was not", expectedMetricCountRemaining)
	}
}
