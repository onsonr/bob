package tasktest

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	dir string
)

var _ = BeforeSuite(func() {
	testDir, err := ioutil.TempDir("", "bob-test-task-*")
	Expect(err).NotTo(HaveOccurred())
	dir = testDir
})

var _ = AfterSuite(func() {
	err := os.RemoveAll(dir)
	Expect(err).NotTo(HaveOccurred())
})

func TestTask(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "task suite")
}
