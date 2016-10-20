/*
 * Copyright 2016 DGraph Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 		http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package worker

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/dgraph/algo"
	"github.com/dgraph-io/dgraph/index"
	"github.com/dgraph-io/dgraph/posting"
	"github.com/dgraph-io/dgraph/schema"
	"github.com/dgraph-io/dgraph/store"
	"github.com/dgraph-io/dgraph/task"
	"github.com/dgraph-io/dgraph/x"
	flatbuffers "github.com/google/flatbuffers/go"
)

func init() {
	ParseGroupConfig("")
	StartRaftNodes(1, "localhost:12345", "1:localhost:12345", "")
	// Wait for the node to become leader for group 0.
	time.Sleep(5 * time.Second)
}

func addEdge(t *testing.T, edge x.DirectedEdge, l *posting.List) {
	require.NoError(t,
		l.AddMutationWithIndex(context.Background(), edge, posting.Set))
}

func delEdge(t *testing.T, edge x.DirectedEdge, l *posting.List) {
	require.NoError(t,
		l.AddMutationWithIndex(context.Background(), edge, posting.Del))
}

func getOrCreate(key []byte, ps *store.Store) *posting.List {
	l, _ := posting.GetOrCreate(key, ps)
	return l
}

func populateGraph(t *testing.T, ps *store.Store) {
	edge := x.DirectedEdge{
		ValueId:   23,
		Source:    "author0",
		Timestamp: time.Now(),
		Attribute: "friend",
	}
	edge.Entity = 10
	addEdge(t, edge, getOrCreate(posting.Key(10, "friend"), ps))

	edge.Entity = 11
	addEdge(t, edge, getOrCreate(posting.Key(11, "friend"), ps))

	edge.Entity = 12
	addEdge(t, edge, getOrCreate(posting.Key(12, "friend"), ps))

	edge.ValueId = 25
	addEdge(t, edge, getOrCreate(posting.Key(12, "friend"), ps))

	edge.ValueId = 26
	addEdge(t, edge, getOrCreate(posting.Key(12, "friend"), ps))

	edge.Entity = 10
	edge.ValueId = 31
	addEdge(t, edge, getOrCreate(posting.Key(10, "friend"), ps))

	edge.Entity = 12
	addEdge(t, edge, getOrCreate(posting.Key(12, "friend"), ps))

	edge.Entity = 12
	edge.Value = []byte("photon")
	addEdge(t, edge, getOrCreate(posting.Key(12, "friend"), ps))

	edge.Entity = 10
	addEdge(t, edge, getOrCreate(posting.Key(10, "friend"), ps))
}

func taskValues(t *testing.T, v *task.ValueList) []string {
	out := make([]string, v.ValuesLength())
	for i := 0; i < v.ValuesLength(); i++ {
		var tv task.Value
		require.True(t, v.Values(&tv, i))
		out[i] = string(tv.ValBytes())
	}
	return out
}

func initTest(t *testing.T, schemaStr string) (string, *store.Store) {
	schema.ParseBytes([]byte(schemaStr))

	dir, err := ioutil.TempDir("", "storetest_")
	require.NoError(t, err)

	ps, err := store.NewStore(dir)
	require.NoError(t, err)

	posting.Init()
	SetState(ps)

	posting.InitIndex(ps)
	index.InitIndex(ps)
	InitIndex()

	populateGraph(t, ps)
	time.Sleep(200 * time.Millisecond) // Let the index process jobs from channel.

	return dir, ps
}

func TestProcessTask(t *testing.T) {
	dir, ps := initTest(t, `scalar friend:string @index`)
	defer os.RemoveAll(dir)
	defer ps.Close()

	query := newQuery("friend", []uint64{10, 11, 12}, nil)
	result, err := processTask(query)
	require.NoError(t, err)

	r := task.GetRootAsResult(result, 0)
	require.EqualValues(t,
		[][]uint64{{23, 31}, {23}, {23, 25, 26, 31}},
		algo.ToUintsListForTest(algo.FromTaskResult(r)))
}

// newQuery creates a Query flatbuffer table, serializes and returns it.
func newQuery(attr string, uids []uint64, tokens []string) []byte {
	b := flatbuffers.NewBuilder(0)

	x.Assert(uids == nil || tokens == nil)

	var vend flatbuffers.UOffsetT
	if uids != nil {
		task.QueryStartUidsVector(b, len(uids))
		for i := len(uids) - 1; i >= 0; i-- {
			b.PrependUint64(uids[i])
		}
		vend = b.EndVector(len(uids))
	} else {
		offsets := make([]flatbuffers.UOffsetT, 0, len(tokens))
		for _, term := range tokens {
			uo := b.CreateString(term)
			offsets = append(offsets, uo)
		}
		task.QueryStartTokensVector(b, len(tokens))
		for i := len(tokens) - 1; i >= 0; i-- {
			b.PrependUOffsetT(offsets[i])
		}
		vend = b.EndVector(len(tokens))
	}

	ao := b.CreateString(attr)
	task.QueryStart(b)
	task.QueryAddAttr(b, ao)
	if uids != nil {
		task.QueryAddUids(b, vend)
	} else {
		task.QueryAddTokens(b, vend)
	}
	qend := task.QueryEnd(b)
	b.Finish(qend)
	return b.Bytes[b.Head():]
}

// Index-related test. Similar to TestProcessTaskIndex but we call MergeLists only
// at the end. In other words, everything is happening only in mutation layers,
// and not committed to RocksDB until near the end.
func TestProcessTaskIndexMLayer(t *testing.T) {
	dir, ps := initTest(t, `scalar friend:string @index`)
	defer os.RemoveAll(dir)
	defer ps.Close()

	query := newQuery("friend", nil, []string{"hey", "photon"})
	result, err := processTask(query)
	require.NoError(t, err)

	r := task.GetRootAsResult(result, 0)
	require.EqualValues(t, [][]uint64{{}, {10, 12}},
		algo.ToUintsListForTest(algo.FromTaskResult(r)))

	// Now try changing 12's friend value from "photon" to "notphoton_extra" to
	// "notphoton".
	edge := x.DirectedEdge{
		Value:     []byte("notphoton_extra"),
		Source:    "author0",
		Timestamp: time.Now(),
		Attribute: "friend",
		Entity:    12,
	}
	addEdge(t, edge, getOrCreate(posting.Key(12, "friend"), ps))
	edge.Value = []byte("notphoton")
	addEdge(t, edge, getOrCreate(posting.Key(12, "friend"), ps))
	time.Sleep(200 * time.Millisecond) // Let the index process jobs from channel.

	// Issue a similar query.
	query = newQuery("friend", nil, []string{"hey", "photon", "notphoton", "notphoton_extra"})
	result, err = processTask(query)
	require.NoError(t, err)

	r = task.GetRootAsResult(result, 0)
	require.EqualValues(t, [][]uint64{{}, {10}, {12}, {}},
		algo.ToUintsListForTest(algo.FromTaskResult(r)))

	// Try deleting.
	edge = x.DirectedEdge{
		Value:     []byte("photon"),
		Source:    "author0",
		Timestamp: time.Now(),
		Attribute: "friend",
		Entity:    10,
	}
	// Redundant deletes.
	delEdge(t, edge, getOrCreate(posting.Key(10, "friend"), ps))
	delEdge(t, edge, getOrCreate(posting.Key(10, "friend"), ps))

	// Delete followed by set.
	edge.Entity = 12
	edge.Value = []byte("notphoton")
	delEdge(t, edge, getOrCreate(posting.Key(12, "friend"), ps))
	edge.Value = []byte("ignored")
	addEdge(t, edge, getOrCreate(posting.Key(12, "friend"), ps))
	time.Sleep(200 * time.Millisecond) // Let the index process jobs from channel.

	// Issue a similar query.
	query = newQuery("friend", nil, []string{"photon", "notphoton", "ignored"})
	result, err = processTask(query)
	require.NoError(t, err)

	r = task.GetRootAsResult(result, 0)
	require.EqualValues(t, [][]uint64{{}, {}, {12}},
		algo.ToUintsListForTest(algo.FromTaskResult(r)))

	// Final touch: Merge everything to RocksDB.
	posting.MergeLists(10)
	time.Sleep(200 * time.Millisecond) // Let the index process jobs from channel.

	query = newQuery("friend", nil, []string{"photon", "notphoton", "ignored"})
	result, err = processTask(query)
	require.NoError(t, err)

	r = task.GetRootAsResult(result, 0)
	require.EqualValues(t, [][]uint64{{}, {}, {12}},
		algo.ToUintsListForTest(algo.FromTaskResult(r)))

	require.EqualValues(t,
		[]string{"extra", "ignored", "notphoton", "photon"},
		index.KeysForTest("friend"))
}

// Index-related test. Similar to TestProcessTaskIndeMLayer except we call
// MergeLists in between a lot of updates.
func TestProcessTaskIndex(t *testing.T) {
	dir, ps := initTest(t, `scalar friend:string @index`)
	defer os.RemoveAll(dir)
	defer ps.Close()

	query := newQuery("friend", nil, []string{"hey", "photon"})
	result, err := processTask(query)
	require.NoError(t, err)

	r := task.GetRootAsResult(result, 0)
	require.EqualValues(t, [][]uint64{{}, {10, 12}},
		algo.ToUintsListForTest(algo.FromTaskResult(r)))

	posting.MergeLists(10)
	time.Sleep(200 * time.Millisecond) // Let the index process jobs from channel.

	// Now try changing 12's friend value from "photon" to "notphoton_extra" to
	// "notphoton".
	edge := x.DirectedEdge{
		Value:     []byte("notphoton_extra"),
		Source:    "author0",
		Timestamp: time.Now(),
		Attribute: "friend",
		Entity:    12,
	}
	addEdge(t, edge, getOrCreate(posting.Key(12, "friend"), ps))
	edge.Value = []byte("notphoton")
	addEdge(t, edge, getOrCreate(posting.Key(12, "friend"), ps))
	time.Sleep(200 * time.Millisecond) // Let the index process jobs from channel.

	// Issue a similar query.
	query = newQuery("friend", nil, []string{"hey", "photon", "notphoton", "notphoton_extra"})
	result, err = processTask(query)
	require.NoError(t, err)

	r = task.GetRootAsResult(result, 0)
	require.EqualValues(t, [][]uint64{{}, {10}, {12}, {}},
		algo.ToUintsListForTest(algo.FromTaskResult(r)))

	posting.MergeLists(10)
	time.Sleep(200 * time.Millisecond) // Let the index process jobs from channel.

	// Try deleting.
	edge = x.DirectedEdge{
		Value:     []byte("photon"),
		Source:    "author0",
		Timestamp: time.Now(),
		Attribute: "friend",
		Entity:    10,
	}
	// Redundant deletes.
	delEdge(t, edge, getOrCreate(posting.Key(10, "friend"), ps))
	delEdge(t, edge, getOrCreate(posting.Key(10, "friend"), ps))

	// Delete followed by set.
	edge.Entity = 12
	edge.Value = []byte("notphoton")
	delEdge(t, edge, getOrCreate(posting.Key(12, "friend"), ps))
	edge.Value = []byte("ignored")
	addEdge(t, edge, getOrCreate(posting.Key(12, "friend"), ps))
	time.Sleep(200 * time.Millisecond) // Let the index process jobs from channel.

	// Issue a similar query.
	query = newQuery("friend", nil, []string{"photon", "notphoton", "ignored"})
	result, err = processTask(query)
	require.NoError(t, err)

	r = task.GetRootAsResult(result, 0)
	require.EqualValues(t, [][]uint64{{}, {}, {12}},
		algo.ToUintsListForTest(algo.FromTaskResult(r)))

	require.EqualValues(t,
		[]string{"extra", "ignored", "notphoton", "photon"},
		index.KeysForTest("friend"))
}

func populateGraphForSort(t *testing.T, ps *store.Store) {
	edge := x.DirectedEdge{
		Source:    "author1",
		Timestamp: time.Now(),
		Attribute: "dob",
	}

	dobs := []string{
		"1980-05-05", // 10 (1980)
		"1980-04-05", // 11
		"1979-05-05", // 12 (1979)
		"1979-02-05", // 13
		"1979-03-05", // 14
		"1965-05-05", // 15 (1965)
		"1965-04-05", // 16
		"1965-03-05", // 17
		"1970-05-05", // 18 (1970)
		"1970-04-05", // 19
		"1970-01-05", // 20
		"1970-02-05", // 21
	}

	for i, dob := range dobs {
		edge.Entity = uint64(i + 10)
		edge.Value = []byte(dob)
		addEdge(t, edge,
			getOrCreate(posting.Key(edge.Entity, edge.Attribute), ps))
	}
}

// newCoarseSort creates a task.Sort for course sorting.
func newCoarseSort(uids [][]uint64, offset, count int) []byte {
	x.Assert(uids != nil)

	b := flatbuffers.NewBuilder(0)
	ao := b.CreateString("dob") // Only consider sort by dob for tests.

	// Add UID matrix.
	uidOffsets := make([]flatbuffers.UOffsetT, 0, len(uids))
	for _, ul := range uids {
		var l algo.UIDList
		l.FromUints(ul)
		uidOffsets = append(uidOffsets, l.AddTo(b))
	}

	task.CoarseSortStartUidmatrixVector(b, len(uidOffsets))
	for i := len(uidOffsets) - 1; i >= 0; i-- {
		b.PrependUOffsetT(uidOffsets[i])
	}
	uend := b.EndVector(len(uidOffsets))

	task.CoarseSortStart(b)
	task.CoarseSortAddUidmatrix(b, uend)
	task.CoarseSortAddOffset(b, int32(offset))
	task.CoarseSortAddCount(b, int32(count))
	csEnd := task.CoarseSortEnd(b)

	task.SortStart(b)
	task.SortAddAttr(b, ao)
	task.SortAddCoarseSort(b, csEnd)
	task.SortAddCoarse(b, byte(1))
	b.Finish(task.SortEnd(b))
	return b.FinishedBytes()
}

func TestProcessCoarseSort(t *testing.T) {
	dir, ps := initTest(t, `scalar dob:date @index`)
	defer os.RemoveAll(dir)
	defer ps.Close()
	populateGraphForSort(t, ps)
	time.Sleep(200 * time.Millisecond) // Let the index process jobs from channel.

	sort := newCoarseSort([][]uint64{
		{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21},
		{10, 11, 12, 13, 14, 21},
		{16, 17, 18, 19, 20, 21},
	}, 0, 0)
	result, err := processSort(sort)
	require.NoError(t, err)

	r := task.GetRootAsSortResult(result, 0)
	require.NotNil(t, r)
	require.EqualValues(t, [][]uint64{
		{15, 16, 17, 18, 19, 20, 21, 12, 13, 14, 10, 11},
		{21, 12, 13, 14, 10, 11},
		{16, 17, 18, 19, 20, 21}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))
}

func TestProcessCoarseSortOffset(t *testing.T) {
	dir, ps := initTest(t, `scalar dob:date @index`)
	defer os.RemoveAll(dir)
	defer ps.Close()
	populateGraphForSort(t, ps)
	time.Sleep(200 * time.Millisecond) // Let the index process jobs from channel.

	input := [][]uint64{
		{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21},
		{10, 11, 12, 13, 14, 21},
		{16, 17, 18, 19, 20, 21}}

	// Offset 1.
	sort := newCoarseSort(input, 1, 0)
	result, err := processSort(sort)
	require.NoError(t, err)
	r := task.GetRootAsSortResult(result, 0)
	require.EqualValues(t, [][]uint64{
		{15, 16, 17, 18, 19, 20, 21, 12, 13, 14, 10, 11},
		{12, 13, 14, 10, 11},
		{16, 17, 18, 19, 20, 21}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))

	// Offset 2.
	sort = newCoarseSort(input, 2, 0)
	result, err = processSort(sort)
	require.NoError(t, err)
	r = task.GetRootAsSortResult(result, 0)
	require.EqualValues(t, [][]uint64{
		{15, 16, 17, 18, 19, 20, 21, 12, 13, 14, 10, 11},
		{12, 13, 14, 10, 11},
		{18, 19, 20, 21}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))

	// Offset 5.
	sort = newCoarseSort(input, 5, 0)
	result, err = processSort(sort)
	require.NoError(t, err)
	r = task.GetRootAsSortResult(result, 0)
	require.EqualValues(t, [][]uint64{
		{18, 19, 20, 21, 12, 13, 14, 10, 11},
		{10, 11},
		{18, 19, 20, 21}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))

	// Offset 6.
	sort = newCoarseSort(input, 6, 0)
	result, err = processSort(sort)
	require.NoError(t, err)
	r = task.GetRootAsSortResult(result, 0)
	require.EqualValues(t, [][]uint64{
		{18, 19, 20, 21, 12, 13, 14, 10, 11},
		{},
		{}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))
}

func TestProcessCoarseSortCount(t *testing.T) {
	dir, ps := initTest(t, `scalar dob:date @index`)
	defer os.RemoveAll(dir)
	defer ps.Close()
	populateGraphForSort(t, ps)
	time.Sleep(200 * time.Millisecond) // Let the index process jobs from channel.

	input := [][]uint64{
		{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21},
		{10, 11, 12, 13, 14, 21},
		{16, 17, 18, 19, 20, 21}}

	// Count 1.
	sort := newCoarseSort(input, 0, 1)
	result, err := processSort(sort)
	require.NoError(t, err)

	r := task.GetRootAsSortResult(result, 0)
	require.NotNil(t, r)
	require.EqualValues(t, [][]uint64{
		{15, 16, 17},
		{21},
		{16, 17}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))

	// Count 2.
	sort = newCoarseSort(input, 0, 2)
	result, err = processSort(sort)
	require.NoError(t, err)

	r = task.GetRootAsSortResult(result, 0)
	require.NotNil(t, r)
	require.EqualValues(t, [][]uint64{
		{15, 16, 17},
		{21, 12, 13, 14},
		{16, 17}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))

	// Count 5.
	sort = newCoarseSort(input, 0, 5)
	result, err = processSort(sort)
	require.NoError(t, err)

	r = task.GetRootAsSortResult(result, 0)
	require.NotNil(t, r)
	require.EqualValues(t, [][]uint64{
		{15, 16, 17, 18, 19, 20, 21},
		{21, 12, 13, 14, 10, 11},
		{16, 17, 18, 19, 20, 21}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))

	// Count 6.
	sort = newCoarseSort(input, 0, 6)
	result, err = processSort(sort)
	require.NoError(t, err)

	r = task.GetRootAsSortResult(result, 0)
	require.NotNil(t, r)
	require.EqualValues(t, [][]uint64{
		{15, 16, 17, 18, 19, 20, 21},
		{21, 12, 13, 14, 10, 11},
		{16, 17, 18, 19, 20, 21}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))
}

func TestProcessCoarseSortOffsetCount(t *testing.T) {
	dir, ps := initTest(t, `scalar dob:date @index`)
	defer os.RemoveAll(dir)
	defer ps.Close()
	populateGraphForSort(t, ps)
	time.Sleep(200 * time.Millisecond) // Let the index process jobs from channel.

	input := [][]uint64{
		{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21},
		{10, 11, 12, 13, 14, 21},
		{16, 17, 18, 19, 20, 21}}

	// Offset 1. Count 1.
	sort := newCoarseSort(input, 1, 1)
	result, err := processSort(sort)
	require.NoError(t, err)
	r := task.GetRootAsSortResult(result, 0)
	require.EqualValues(t, [][]uint64{
		{15, 16, 17},
		{12, 13, 14},
		{16, 17}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))

	// Offset 1. Count 2.
	sort = newCoarseSort(input, 1, 2)
	result, err = processSort(sort)
	require.NoError(t, err)
	r = task.GetRootAsSortResult(result, 0)
	require.EqualValues(t, [][]uint64{
		{15, 16, 17},
		{12, 13, 14},
		{16, 17, 18, 19, 20, 21}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))

	// Offset 1. Count 3.
	sort = newCoarseSort(input, 1, 3)
	result, err = processSort(sort)
	require.NoError(t, err)
	r = task.GetRootAsSortResult(result, 0)
	require.EqualValues(t, [][]uint64{
		{15, 16, 17, 18, 19, 20, 21},
		{12, 13, 14},
		{16, 17, 18, 19, 20, 21}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))

	// Offset 1. Count 1000.
	sort = newCoarseSort(input, 1, 1000)
	result, err = processSort(sort)
	require.NoError(t, err)
	r = task.GetRootAsSortResult(result, 0)
	require.EqualValues(t, [][]uint64{
		{15, 16, 17, 18, 19, 20, 21, 12, 13, 14, 10, 11},
		{12, 13, 14, 10, 11},
		{16, 17, 18, 19, 20, 21}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))

	// Offset 5. Count 1.
	sort = newCoarseSort(input, 5, 1)
	result, err = processSort(sort)
	require.NoError(t, err)
	r = task.GetRootAsSortResult(result, 0)
	require.EqualValues(t, [][]uint64{
		{18, 19, 20, 21},
		{10, 11},
		{18, 19, 20, 21}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))

	// Offset 5. Count 2.
	sort = newCoarseSort(input, 5, 2)
	result, err = processSort(sort)
	require.NoError(t, err)
	r = task.GetRootAsSortResult(result, 0)
	require.EqualValues(t, [][]uint64{
		{18, 19, 20, 21},
		{10, 11},
		{18, 19, 20, 21}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))

	// Offset 5. Count 3.
	sort = newCoarseSort(input, 5, 3)
	result, err = processSort(sort)
	require.NoError(t, err)
	r = task.GetRootAsSortResult(result, 0)
	require.EqualValues(t, [][]uint64{
		{18, 19, 20, 21, 12, 13, 14},
		{10, 11},
		{18, 19, 20, 21}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))

	// Offset 100. Count 100.
	sort = newCoarseSort(input, 100, 100)
	result, err = processSort(sort)
	require.NoError(t, err)
	r = task.GetRootAsSortResult(result, 0)
	require.EqualValues(t, [][]uint64{
		{},
		{},
		{}},
		algo.ToUintsListForTest(algo.FromSortResult(r)))
}

func newFineSort(uids []uint64) []byte {
	x.Assert(uids != nil)

	b := flatbuffers.NewBuilder(0)
	ao := b.CreateString("dob") // Only consider sort by dob for tests.

	var l algo.UIDList
	l.FromUints(uids)
	uo := l.AddTo(b)

	task.FineSortStart(b)
	task.FineSortAddUid(b, uo)
	fsEnd := task.FineSortEnd(b)

	task.SortStart(b)
	task.SortAddAttr(b, ao)
	task.SortAddFineSort(b, fsEnd)
	task.SortAddCoarse(b, byte(0))
	b.Finish(task.SortEnd(b))
	return b.FinishedBytes()
}

func TestProcessFineSort(t *testing.T) {
	dir, ps := initTest(t, `scalar dob:date @index`)
	defer os.RemoveAll(dir)
	defer ps.Close()
	populateGraphForSort(t, ps)
	time.Sleep(200 * time.Millisecond) // Let the index process jobs from channel.

	sort := newFineSort([]uint64{
		10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21})
	result, err := processSort(sort)
	require.NoError(t, err)
	require.NotNil(t, result)

	r := task.GetRootAsSortResult(result, 0)
	idx := make([]uint32, r.IdxLength())
	for i := 0; i < r.IdxLength(); i++ {
		idx[i] = r.Idx(i)
	}
	require.EqualValues(t,
		[]uint32{7, 6, 5, 10, 11, 9, 8, 3, 4, 2, 1, 0}, idx)
}

func TestMain(m *testing.M) {
	x.Init()
	os.Exit(m.Run())
}
