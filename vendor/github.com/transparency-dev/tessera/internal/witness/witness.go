// Copyright 2025 The Tessera authors. All Rights Reserved.
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

// Package witness contains the implementation for sending out a checkpoint to witnesses
// and retrieving sufficient signatures to satisfy a policy.
package witness

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/transparency-dev/formats/log"
	"github.com/transparency-dev/tessera/client"
	"github.com/transparency-dev/tessera/internal/parse"
	"golang.org/x/mod/sumdb/note"
)

var ErrPolicyNotSatisfied = errors.New("witness policy was not satisfied")

// WitnessGroup defines a group of witnesses, and a threshold of
// signatures that must be met for this group to be satisfied.
// Witnesses within a group should be fungible, e.g. all of the Armored
// Witness devices form a logical group, and N should be picked to
// represent a threshold of the quorum. For some users this will be a
// simple majority, but other strategies are available.
// N must be <= len(WitnessKeys).
type WitnessGroup interface {
	// Satisfied returns true if the checkpoint provided is signed by this witness.
	// This will return false if there is no signature, and also if the
	// checkpoint cannot be read as a valid note. It is up to the caller to ensure
	// that the input value represents a valid note.
	Satisfied(cp []byte) bool

	// Endpoints returns the details required for updating a witness and checking the
	// response. The returned result is a map from the URL that should be used to update
	// the witness with a new checkpoint, to the value which is the verifier to check
	// the response is well formed.
	Endpoints() map[string]note.Verifier
}

// NewWitnessGateway returns a WitnessGateway that will send out new checkpoints to witnesses
// in the group, and will ensure that the policy is satisfied before returning. All outbound
// requests will be done using the given client. The tile fetcher is used for constructing
// consistency proofs for the witnesses.
func NewWitnessGateway(group WitnessGroup, client *http.Client, fetchTiles client.TileFetcherFunc) WitnessGateway {
	endpoints := group.Endpoints()
	witnesses := make([]*witness, 0, len(endpoints))
	for u, v := range endpoints {
		witnesses = append(witnesses, &witness{
			client:   client,
			url:      u,
			verifier: v,
			size:     0,
		})
	}
	return WitnessGateway{
		group:     group,
		witnesses: witnesses,
		fetchTile: fetchTiles,
	}
}

// WitnessGateway allows a log implementation to send out a checkpoint to witnesses.
type WitnessGateway struct {
	group     WitnessGroup
	witnesses []*witness
	fetchTile client.TileFetcherFunc
}

// Witness sends out a new checkpoint (which must be signed by the log), to all witnesses
// and returns the checkpoint as soon as the policy the WitnessGateway was constructed with
// is Satisfied.
func (wg *WitnessGateway) Witness(ctx context.Context, cp []byte) ([]byte, error) {
	if len(wg.witnesses) == 0 {
		return cp, nil
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var waitGroup sync.WaitGroup
	origin, size, hash, err := parse.CheckpointUnsafe(cp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse checkpoint from log: %v", err)
	}
	logCP := log.Checkpoint{
		Origin: origin,
		Size:   size,
		Hash:   hash,
	}
	pb, err := client.NewProofBuilder(ctx, logCP.Size, wg.fetchTile)
	if err != nil {
		return nil, fmt.Errorf("failed to build proof builder: %v", err)
	}
	pf := sharedConsistencyProofFetcher{
		pb:      pb,
		toSize:  size,
		results: make(map[uint64]consistencyFuture),
	}

	type sigOrErr struct {
		sig []byte
		err error
	}
	results := make(chan sigOrErr)

	// Kick off a goroutine for each witness and send result to results chan
	for _, w := range wg.witnesses {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			sig, err := w.update(ctx, cp, size, pf.ConsistencyProof)
			results <- sigOrErr{
				sig: sig,
				err: err,
			}
		}()
	}

	go func() {
		waitGroup.Wait()
		close(results)
	}()

	// Consume the results coming back from each witness
	var sigBlock bytes.Buffer
	sigBlock.Write(cp)
	for r := range results {
		if r.err != nil {
			err = errors.Join(err, r.err)
			continue
		}
		// Some basic validation, which can be extended if needed.
		if !bytes.HasSuffix(r.sig, []byte("\n")) {
			err = errors.Join(err, fmt.Errorf("invalid signature from witness: %q", r.sig))
			continue
		}
		// Add new signature to the new note we're building
		sigBlock.Write(r.sig)

		// See whether the group is satisfied now
		if newCp := sigBlock.Bytes(); wg.group.Satisfied(newCp) {
			return newCp, nil
		}
	}

	// We can only get here if all witnesses have returned and we're still not satisfied.
	return sigBlock.Bytes(), errors.Join(ErrPolicyNotSatisfied, err)
}

type consistencyFuture func() ([][]byte, error)

// sharedConsistencyProofFetcher is a thread-safe caching wrapper around a proof builder.
// This is an optimization for the common case where multiple witnesses are used, and all
// of the witnesses are of the same size, and thus require the same proof.
type sharedConsistencyProofFetcher struct {
	pb      *client.ProofBuilder
	toSize  uint64
	mu      sync.Mutex
	results map[uint64]consistencyFuture
}

// ConsistencyProof constructs a consistency proof, reusing any results from parallel requests.
func (pf *sharedConsistencyProofFetcher) ConsistencyProof(ctx context.Context, smaller, larger uint64) ([][]byte, error) {
	if larger != pf.toSize {
		return nil, fmt.Errorf("required larger size to be %d but was given %d", pf.toSize, larger)
	}
	var f consistencyFuture
	var ok bool
	pf.mu.Lock()
	if f, ok = pf.results[smaller]; !ok {
		f = sync.OnceValues(func() ([][]byte, error) {
			return pf.pb.ConsistencyProof(ctx, smaller, larger)
		})
		pf.results[smaller] = f
	}
	pf.mu.Unlock()
	return f()
}

// witness is the log's model of a witness's view of this log.
// It has a URL which is the address to which updates to this log's state can be posted to the witness,
// using the https://github.com/C2SP/C2SP/blob/main/tlog-witness.md spec.
// It also has the size of the checkpoint that the log thinks that the witness last signed.
// This is important for sending update proofs.
// This is defaulted to zero on startup and calibrated after the first request, which is expected by the spec:
// `If a client doesn't have information on the latest cosigned checkpoint, it MAY initially make a request with a old size of zero to obtain it`
type witness struct {
	client   *http.Client
	url      string
	verifier note.Verifier
	size     uint64
}

func (w *witness) update(ctx context.Context, cp []byte, size uint64, fetchProof func(ctx context.Context, from, to uint64) ([][]byte, error)) ([]byte, error) {
	var proof [][]byte
	if w.size > 0 {
		var err error
		proof, err = fetchProof(ctx, w.size, size)
		if err != nil {
			return nil, fmt.Errorf("fetchProof: %v", err)
		}
	}

	// The request body MUST be a sequence of
	// - a previous size line,
	// - zero or more consistency proof lines,
	// - and an empty line,
	// - followed by a [checkpoint][].
	body := fmt.Sprintf("old %d\n", w.size)
	for _, p := range proof {
		body += base64.StdEncoding.EncodeToString(p) + "\n"
	}
	body += "\n"
	body += string(cp)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to construct request to %q: %v", w.url, err)
	}
	httpResp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to post to witness at %q: %v", w.url, err)
	}
	rb, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body from witness at %q: %v", w.url, err)
	}
	_ = httpResp.Body.Close()

	switch httpResp.StatusCode {
	case http.StatusOK:
		// Concatenate the signature to the checkpoint passed in and verify it is valid.
		// append is tempting here but is dangerous because it can modify `cp` and race with other
		// witnesses, causing signatures to be swapped. cp must not be modified.
		signed := make([]byte, len(cp)+len(rb))
		copy(signed, cp)
		copy(signed[len(cp):], rb)
		if n, err := note.Open(signed, note.VerifierList(w.verifier)); err != nil {
			return nil, fmt.Errorf("witness %q at %q replied with invalid signature: %q\nconstructed note: %q\nerror: %v", w.verifier.Name(), w.url, rb, string(signed), err)
		} else {
			w.size = uint64(size)
			return fmt.Appendf(nil, "— %s %s\n", n.Sigs[0].Name, n.Sigs[0].Base64), nil
		}
	case http.StatusConflict:
		// Two cases here: the first is a situation we can recover from, the second isn't.

		// The witness MUST check that the old size matches the size of the latest checkpoint it cosigned
		// for the checkpoint's origin (or zero if it never cosigned a checkpoint for that origin).
		// If it doesn't match, the witness MUST respond with a "409 Conflict" HTTP status code.
		// The response body MUST consist of the tree size of the latest cosigned checkpoint in decimal,
		// followed by a newline (U+000A). The response MUST have a Content-Type of text/x.tlog.size
		ct := httpResp.Header["Content-Type"]
		if len(ct) == 1 && ct[0] == "text/x.tlog.size" {
			bodyStr := strings.TrimSpace(string(rb))
			newWitSize, err := strconv.ParseUint(bodyStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("witness at %q replied with x.tlog.size but body %q could not be parsed as decimal", w.url, bodyStr)
			}
			// These cases should not happen unless the witness is misbehaving.
			// w.size <= newWitSize <= size must always be true.
			if newWitSize > size {
				return nil, fmt.Errorf("witness at %q replied with x.tlog.size %d, larger than log size %d", w.url, newWitSize, size)
			}
			if newWitSize < w.size {
				return nil, fmt.Errorf("witness at %q replied with x.tlog.size %d, smaller than known size %d", w.url, newWitSize, w.size)
			}
			w.size = newWitSize
			// Witnesses could cause this recursion to go on for longer than expected if the value they kept returning
			// this case with slightly larger values. Consider putting a max recursion cap if context timeout isn't enough.
			return w.update(ctx, cp, size, fetchProof)
		}

		// If the old size matches the checkpoint size, the witness MUST check that the root hashes are also identical.
		// If they don't match, the witness MUST respond with a "409 Conflict" HTTP status code.
		return nil, fmt.Errorf("witness at %q says old root hash did not match previous for size %d: %d", w.url, w.size, httpResp.StatusCode)
	case http.StatusNotFound:
		// If the checkpoint origin is unknown, the witness MUST respond with a "404 Not Found" HTTP status code.
		return nil, fmt.Errorf("witness at %q says checkpoint origin is unknown: %d", w.url, httpResp.StatusCode)
	case http.StatusForbidden:
		// If none of the signatures verify against a trusted public key, the witness MUST respond with a "403 Forbidden" HTTP status code.
		return nil, fmt.Errorf("witness at %q says no signatures verify against trusted public key: %d", w.url, httpResp.StatusCode)
	case http.StatusBadRequest:
		// The old size MUST be equal to or lower than the checkpoint size.
		// Otherwise, the witness MUST respond with a "400 Bad Request" HTTP status code.
		return nil, fmt.Errorf("witness at %q says old checkpoint size of %d is too large: %d", w.url, w.size, httpResp.StatusCode)
	case http.StatusUnprocessableEntity:
		//  If the Merkle Consistency Proof doesn't verify, the witness MUST respond with a "422 Unprocessable Entity" HTTP status code.
		return nil, fmt.Errorf("witness at %q says that the consistency proof is bad: %d", w.url, httpResp.StatusCode)
	default:
		return nil, fmt.Errorf("got bad status code: %v", httpResp.StatusCode)
	}
}
