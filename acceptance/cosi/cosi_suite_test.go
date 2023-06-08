package cosi_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCosi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cosi Suite")
}
