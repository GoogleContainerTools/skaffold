package opts

import (
	"os"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestSecretOptionsSimple(t *testing.T) {
	var opt SecretOpt

	testCase := "app-secret"
	assert.NilError(t, opt.Set(testCase))

	reqs := opt.Value()
	assert.Assert(t, is.Len(reqs, 1))
	req := reqs[0]
	assert.Check(t, is.Equal("app-secret", req.SecretName))
	assert.Check(t, is.Equal("app-secret", req.File.Name))
	assert.Check(t, is.Equal("0", req.File.UID))
	assert.Check(t, is.Equal("0", req.File.GID))
}

func TestSecretOptionsSourceTarget(t *testing.T) {
	var opt SecretOpt

	testCase := "source=foo,target=testing"
	assert.NilError(t, opt.Set(testCase))

	reqs := opt.Value()
	assert.Assert(t, is.Len(reqs, 1))
	req := reqs[0]
	assert.Check(t, is.Equal("foo", req.SecretName))
	assert.Check(t, is.Equal("testing", req.File.Name))
}

func TestSecretOptionsShorthand(t *testing.T) {
	var opt SecretOpt

	testCase := "src=foo,target=testing"
	assert.NilError(t, opt.Set(testCase))

	reqs := opt.Value()
	assert.Assert(t, is.Len(reqs, 1))
	req := reqs[0]
	assert.Check(t, is.Equal("foo", req.SecretName))
}

func TestSecretOptionsCustomUidGid(t *testing.T) {
	var opt SecretOpt

	testCase := "source=foo,target=testing,uid=1000,gid=1001"
	assert.NilError(t, opt.Set(testCase))

	reqs := opt.Value()
	assert.Assert(t, is.Len(reqs, 1))
	req := reqs[0]
	assert.Check(t, is.Equal("foo", req.SecretName))
	assert.Check(t, is.Equal("testing", req.File.Name))
	assert.Check(t, is.Equal("1000", req.File.UID))
	assert.Check(t, is.Equal("1001", req.File.GID))
}

func TestSecretOptionsCustomMode(t *testing.T) {
	var opt SecretOpt

	testCase := "source=foo,target=testing,uid=1000,gid=1001,mode=0444"
	assert.NilError(t, opt.Set(testCase))

	reqs := opt.Value()
	assert.Assert(t, is.Len(reqs, 1))
	req := reqs[0]
	assert.Check(t, is.Equal("foo", req.SecretName))
	assert.Check(t, is.Equal("testing", req.File.Name))
	assert.Check(t, is.Equal("1000", req.File.UID))
	assert.Check(t, is.Equal("1001", req.File.GID))
	assert.Check(t, is.Equal(os.FileMode(0444), req.File.Mode))
}
