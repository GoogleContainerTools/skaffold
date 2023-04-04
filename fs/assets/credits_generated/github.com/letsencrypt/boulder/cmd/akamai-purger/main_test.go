package notmain

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/letsencrypt/boulder/akamai/proto"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/test"
)

func TestThroughput_validate(t *testing.T) {
	type fields struct {
		QueueEntriesPerBatch int
		PurgeBatchInterval   time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"optimized defaults, should succeed",
			fields{
				QueueEntriesPerBatch: defaultEntriesPerBatch,
				PurgeBatchInterval:   defaultPurgeBatchInterval},
			false,
		},
		{"2ms faster than optimized defaults, should succeed",
			fields{
				QueueEntriesPerBatch: defaultEntriesPerBatch,
				PurgeBatchInterval:   defaultPurgeBatchInterval + 2*time.Millisecond},
			false,
		},
		{"exceeds URLs per second by 4 URLs",
			fields{
				QueueEntriesPerBatch: defaultEntriesPerBatch,
				PurgeBatchInterval:   29 * time.Millisecond},
			true,
		},
		{"exceeds bytes per second by 20 bytes",
			fields{
				QueueEntriesPerBatch: 125,
				PurgeBatchInterval:   1 * time.Second},
			true,
		},
		{"exceeds requests per second by 1 request",
			fields{
				QueueEntriesPerBatch: 1,
				PurgeBatchInterval:   19999 * time.Microsecond},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Throughput{
				QueueEntriesPerBatch: tt.fields.QueueEntriesPerBatch,
			}
			tr.PurgeBatchInterval.Duration = tt.fields.PurgeBatchInterval
			if err := tr.validate(); (err != nil) != tt.wantErr {
				t.Errorf("Throughput.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type mockCCU struct {
	proto.AkamaiPurgerClient
}

func (m *mockCCU) Purge(urls []string) error {
	return errors.New("Lol, I'm a mock")
}

func TestAkamaiPurgerQueue(t *testing.T) {
	ap := &akamaiPurger{
		maxStackSize:    250,
		entriesPerBatch: 2,
		client:          &mockCCU{},
		log:             blog.NewMock(),
	}

	// Add 250 entries to fill the stack.
	for i := 0; i < 250; i++ {
		req := proto.PurgeRequest{Urls: []string{fmt.Sprintf("http://test.com/%d", i)}}
		_, err := ap.Purge(context.Background(), &req)
		test.AssertNotError(t, err, fmt.Sprintf("Purge failed for entry %d.", i))
	}

	// Add another entry to the stack and using the Purge method.
	req := proto.PurgeRequest{Urls: []string{"http://test.com/250"}}
	_, err := ap.Purge(context.Background(), &req)
	test.AssertNotError(t, err, "Purge failed.")

	// Verify that the stack is still full.
	test.AssertEquals(t, len(ap.toPurge), 250)

	// Verify that the first entry in the stack is the entry we just added.
	test.AssertEquals(t, ap.toPurge[len(ap.toPurge)-1][0], "http://test.com/250")

	// Verify that the last entry in the stack is the second entry we added.
	test.AssertEquals(t, ap.toPurge[0][0], "http://test.com/1")

	expectedTopEntryAfterFailure := ap.toPurge[len(ap.toPurge)-(ap.entriesPerBatch+1)][0]

	// Fail to purge a batch of entries from the stack.
	batch := ap.takeBatch()
	test.AssertNotNil(t, batch, "Batch should not be nil.")

	err = ap.purgeBatch(batch)
	test.AssertError(t, err, "Mock should have failed to purge.")

	// Verify that the stack is no longer full.
	test.AssertEquals(t, len(ap.toPurge), 248)

	// The first entry of the next batch should be on the top after the failed
	// purge.
	test.AssertEquals(t, ap.toPurge[len(ap.toPurge)-1][0], expectedTopEntryAfterFailure)
}

func TestAkamaiPurgerQueueWithOneEntry(t *testing.T) {
	ap := &akamaiPurger{
		maxStackSize:    250,
		entriesPerBatch: 2,
		client:          &mockCCU{},
		log:             blog.NewMock(),
	}

	// Add one entry to the stack and using the Purge method.
	req := proto.PurgeRequest{Urls: []string{"http://test.com/0"}}
	_, err := ap.Purge(context.Background(), &req)
	test.AssertNotError(t, err, "Purge failed.")
	test.AssertEquals(t, len(ap.toPurge), 1)
	test.AssertEquals(t, ap.toPurge[len(ap.toPurge)-1][0], "http://test.com/0")

	// Fail to purge a batch of entries from the stack.
	batch := ap.takeBatch()
	test.AssertNotNil(t, batch, "Batch should not be nil.")

	err = ap.purgeBatch(batch)
	test.AssertError(t, err, "Mock should have failed to purge.")

	// Verify that the stack no longer contains our entry.
	test.AssertEquals(t, len(ap.toPurge), 0)
}
