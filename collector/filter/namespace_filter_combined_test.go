package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"

	conf "github.com/alibaba/MongoShake/v2/collector/configure"
	"github.com/alibaba/MongoShake/v2/oplog"
)

func TestCombinedNamespaceFilterDocumentsAndOplogs(t *testing.T) {
	white := []string{"serving", "ats", "indeed", "ups"}
	black := []string{"serving.gpt_result_dump"}
	filter := NewNamespaceFilter(white, black)

	for _, tc := range []struct {
		namespace string
		filtered  bool
	}{
		{"serving.jobs", false},
		{"serving.gpt_result_dump", true},
		{"serving.gpt_result_dump.archive", true},
		{"serving.gpt_result_dump_other", false},
		{"ats.jobs", false},
		{"linkedin.jobs", true},
	} {
		t.Run(tc.namespace, func(t *testing.T) {
			assert.Equal(t, tc.filtered, filter.FilterNs(tc.namespace), "full sync filter")
			log := &oplog.PartialLog{ParsedLog: oplog.ParsedLog{
				Namespace: tc.namespace,
				Operation: "i",
			}}
			assert.Equal(t, tc.filtered, filter.Filter(log), "incremental sync filter")
		})
	}
}

func TestCombinedNamespaceFilterUsedByFullSync(t *testing.T) {
	origin := conf.Options
	defer func() { conf.Options = origin }()
	conf.Options.FilterNamespaceWhite = []string{"serving"}
	conf.Options.FilterNamespaceBlack = []string{"serving.gpt_result_dump"}

	filters := NewDocFilterList()
	assert.False(t, filters.IterateFilter("serving.jobs"))
	assert.True(t, filters.IterateFilter("serving.gpt_result_dump"))
	assert.True(t, filters.IterateFilter("linkedin.jobs"))
}

func TestCombinedNamespaceFilterDDLAndApplyOps(t *testing.T) {
	filter := NewNamespaceFilter([]string{"serving"}, []string{"serving.gpt_result_dump"})

	for _, tc := range []struct {
		collection string
		filtered   bool
	}{
		{"gpt_result_dump", true},
		{"jobs", false},
	} {
		t.Run("drop "+tc.collection, func(t *testing.T) {
			log := &oplog.PartialLog{ParsedLog: oplog.ParsedLog{
				Namespace: "serving.$cmd",
				Operation: "c",
				Object:    bson.D{{Key: "drop", Value: tc.collection}},
			}}
			assert.Equal(t, tc.filtered, filter.Filter(log))
		})
	}

	log := &oplog.PartialLog{ParsedLog: oplog.ParsedLog{
		Namespace: "admin.$cmd",
		Operation: "c",
		Object: bson.D{{Key: "applyOps", Value: []bson.D{
			{{Key: "op", Value: "i"}, {Key: "ns", Value: "serving.gpt_result_dump"}, {Key: "o", Value: bson.D{{Key: "_id", Value: 1}}}},
			{{Key: "op", Value: "i"}, {Key: "ns", Value: "serving.jobs"}, {Key: "o", Value: bson.D{{Key: "_id", Value: 2}}}},
			{{Key: "op", Value: "i"}, {Key: "ns", Value: "linkedin.jobs"}, {Key: "o", Value: bson.D{{Key: "_id", Value: 3}}}},
		}}},
	}}
	require.False(t, filter.Filter(log), "the allowed operation should remain")
	ops, ok := log.Object[0].Value.(bson.A)
	require.True(t, ok)
	require.Len(t, ops, 1)
	assert.Equal(t, "serving.jobs", oplog.GetKey(ops[0].(bson.D), "ns"))
}
