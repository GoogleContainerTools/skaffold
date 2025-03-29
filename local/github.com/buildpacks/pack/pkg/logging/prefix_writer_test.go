package logging_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestPrefixWriter(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "PrefixWriter", testPrefixWriter, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testPrefixWriter(t *testing.T, when spec.G, it spec.S) {
	var (
		assert = h.NewAssertionManager(t)
	)

	when("#Write", func() {
		it("prepends prefix to string", func() {
			var w bytes.Buffer

			writer := logging.NewPrefixWriter(&w, "prefix")
			_, err := writer.Write([]byte("test"))
			assert.Nil(err)
			err = writer.Close()
			assert.Nil(err)

			h.AssertEq(t, w.String(), "[prefix] test\n")
		})

		it("prepends prefix to multi-line string", func() {
			var w bytes.Buffer

			writer := logging.NewPrefixWriter(&w, "prefix")
			_, err := writer.Write([]byte("line 1\nline 2\nline 3"))
			assert.Nil(err)
			err = writer.Close()
			assert.Nil(err)

			h.AssertEq(t, w.String(), "[prefix] line 1\n[prefix] line 2\n[prefix] line 3\n")
		})

		it("buffers mid-line calls", func() {
			var buf bytes.Buffer

			writer := logging.NewPrefixWriter(&buf, "prefix")
			_, err := writer.Write([]byte("word 1, "))
			assert.Nil(err)
			_, err = writer.Write([]byte("word 2, "))
			assert.Nil(err)
			_, err = writer.Write([]byte("word 3."))
			assert.Nil(err)
			err = writer.Close()
			assert.Nil(err)

			h.AssertEq(t, buf.String(), "[prefix] word 1, word 2, word 3.\n")
		})

		it("handles empty lines", func() {
			var buf bytes.Buffer

			writer := logging.NewPrefixWriter(&buf, "prefix")
			_, err := writer.Write([]byte("\n"))
			assert.Nil(err)
			err = writer.Close()
			assert.Nil(err)

			h.AssertEq(t, buf.String(), "[prefix] \n")
		})

		it("handles empty input", func() {
			var buf bytes.Buffer

			writer := logging.NewPrefixWriter(&buf, "prefix")
			_, err := writer.Write([]byte(""))
			assert.Nil(err)
			err = writer.Close()
			assert.Nil(err)

			assert.Equal(buf.String(), "")
		})

		it("propagates reader errors", func() {
			var buf bytes.Buffer

			factory := &boobyTrapReaderFactory{failAtCallNumber: 2}
			writer := logging.NewPrefixWriter(&buf, "prefix", logging.WithReaderFactory(factory.NewReader))
			_, err := writer.Write([]byte("word 1,"))
			assert.Nil(err)
			_, err = writer.Write([]byte("word 2."))
			assert.ErrorContains(err, "some error")
		})

		it("handles requests to clear line", func() {
			var buf bytes.Buffer

			writer := logging.NewPrefixWriter(&buf, "prefix")
			_, err := writer.Write([]byte("progress 1\rprogress 2\rprogress 3\rcomplete!"))
			assert.Nil(err)
			err = writer.Close()
			assert.Nil(err)

			h.AssertEq(t, buf.String(), "[prefix] complete!\n")
		})

		it("handles requests clear line (amidst content)", func() {
			var buf bytes.Buffer

			writer := logging.NewPrefixWriter(&buf, "prefix")
			_, err := writer.Write([]byte("downloading\rcompleted!      \r\nall done!\nnevermind\r"))
			assert.Nil(err)
			err = writer.Close()
			assert.Nil(err)

			h.AssertEq(t, buf.String(), "[prefix] completed!      \n[prefix] all done!\n[prefix] \n")
		})
	})
}

type boobyTrapReaderFactory struct {
	numberOfCalls    int
	failAtCallNumber int
}

func (b *boobyTrapReaderFactory) NewReader(data []byte) io.Reader {
	b.numberOfCalls++
	if b.numberOfCalls >= b.failAtCallNumber {
		return &faultyReader{}
	}

	return bytes.NewReader(data)
}

type faultyReader struct {
}

func (f faultyReader) Read(b []byte) (n int, err error) {
	return 0, errors.New("some error")
}
