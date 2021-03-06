/*
* Copyright 2016 DGraph Labs, Inc.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*         http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package worker

import (
	"bytes"
	"context"
	"io"
	"sort"

	"github.com/dgraph-io/dgraph/posting/types"
	"github.com/dgraph-io/dgraph/task"
	"github.com/dgraph-io/dgraph/x"
)

const (
	// MB represents a megabyte.
	MB = 1 << 20
)

// writeBatch performs a batch write of key value pairs to RocksDB.
func writeBatch(ctx context.Context, kv chan *task.KV, che chan error) {
	wb := pstore.NewWriteBatch()
	batchSize := 0
	batchWriteNum := 1
	for i := range kv {
		wb.Put(i.Key, i.Val)
		batchSize += len(i.Key) + len(i.Val)
		// We write in batches of size 32MB.
		if batchSize >= 32*MB {
			x.Trace(ctx, "Doing batch write %d.", batchWriteNum)
			if err := pstore.WriteBatch(wb); err != nil {
				che <- err
				return
			}

			batchWriteNum++
			// Resetting batch size after a batch write.
			batchSize = 0
			// Since we are writing data in batches, we need to clear up items enqueued
			// for batch write after every successful write.
			wb.Clear()
		}
	}
	// After channel is closed the above loop would exit, we write the data in
	// write batch here.
	if batchSize > 0 {
		x.Trace(ctx, "Doing batch write %d.", batchWriteNum)
		che <- pstore.WriteBatch(wb)
		return
	}
	che <- nil
}

func generateGroup(group uint64) (*task.GroupKeys, error) {
	it := pstore.NewIterator()
	defer it.Close()

	g := &task.GroupKeys{
		GroupId: group,
	}

	for it.SeekToFirst(); it.Valid(); it.Next() {
		// TODO: Check if this key belongs to the group.

		k, v := it.Key(), it.Value()
		var pl types.PostingList
		x.Check(pl.Unmarshal(v.Data()))

		key := &task.KC{
			Key:      k.Data(),
			Checksum: pl.Checksum,
		}
		g.Keys = append(g.Keys, key)
	}
	return g, it.Err()
}

// PopulateShard gets data for predicate pred from server with id serverId and
// writes it to RocksDB.
func populateShard(ctx context.Context, pl *pool, group uint64) (int, error) {
	gkeys, err := generateGroup(group)
	if err != nil {
		return 0, x.Wrapf(err, "While generating keys group")
	}

	conn, err := pl.Get()
	if err != nil {
		return 0, err
	}
	defer pl.Put(conn)
	c := NewWorkerClient(conn)

	stream, err := c.PredicateData(context.Background(), gkeys)
	if err != nil {
		return 0, err
	}
	x.Trace(ctx, "Streaming data for group: %v", group)

	kvs := make(chan *task.KV, 1000)
	che := make(chan error)
	go writeBatch(ctx, kvs, che)

	// We can use count to check the number of posting lists returned in tests.
	count := 0
	for {
		kv, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			close(kvs)
			return count, err
		}
		count++

		// We check for errors, if there are no errors we send value to channel.
		select {
		case kvs <- kv:
			// OK
		case <-ctx.Done():
			x.TraceError(ctx, x.Errorf("Context timed out while streaming group: %v", group))
			close(kvs)
			return count, ctx.Err()
		case err := <-che:
			x.TraceError(ctx, x.Errorf("Error while doing a batch write for group: %v", group))
			close(kvs)
			return count, err
		}
	}
	close(kvs)

	if err := <-che; err != nil {
		x.TraceError(ctx, x.Errorf("Error while doing a batch write for group: %v", group))
		return count, err
	}
	x.Trace(ctx, "Streaming complete for group: %v", group)
	return count, nil
}

// PredicateData can be used to return data corresponding to a predicate over
// a stream.
func (w *grpcWorker) PredicateData(group *task.GroupKeys, stream Worker_PredicateDataServer) error {
	// TODO: Use group id.

	// TODO(pawan) - Shift to CheckPoints once we figure out how to add them to the
	// RocksDB library we are using.
	// http://rocksdb.org/blog/2609/use-checkpoints-for-efficient-snapshots/
	it := pstore.NewIterator()
	defer it.Close()

	for it.SeekToFirst(); it.Valid(); it.Next() {
		k, v := it.Key(), it.Value()
		var pl types.PostingList
		x.Check(pl.Unmarshal(v.Data()))

		// TODO: Check that key is part of the specified group id.
		idx := sort.Search(len(group.Keys), func(i int) bool {
			t := group.Keys[i]
			return bytes.Compare(k.Data(), t.Key) <= 0
		})

		if idx < len(group.Keys) {
			// Found a match.
			t := group.Keys[idx]
			// Different keys would have the same prefix. So, check Checksum first,
			// it would be cheaper when there's no match.
			if bytes.Equal(pl.Checksum, t.Checksum) && bytes.Equal(k.Data(), t.Key) {
				// No need to send this.
				continue
			}
		}

		kv := &task.KV{
			Key: k.Data(),
			Val: v.Data(),
		}

		if err := stream.Send(kv); err != nil {
			return err
		}
		k.Free()
		v.Free()
	}
	if err := it.Err(); err != nil {
		return err
	}
	return nil
}
