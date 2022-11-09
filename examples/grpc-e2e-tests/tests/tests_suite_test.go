package test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGrpcE2eTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GrpcE2eTests Suite")
}
