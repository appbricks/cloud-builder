package helpers

import (
	"bytes"
	"io"
	"log"
	"os"
	"sync"

	"github.com/onsi/ginkgo"
)

// testing helper function to capture output
// written to stdout/stderr by the encapsulated
// function
func CaptureOutput(f func()) string {

	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	stdout := os.Stdout
	stderr := os.Stderr
	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
		log.SetOutput(os.Stderr)
	}()
	os.Stdout = writer
	os.Stderr = writer
	log.SetOutput(writer)

	out := make(chan string)

	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {

		var buf bytes.Buffer
		wg.Done()

		_, err := io.Copy(&buf, reader)
		if err != nil {
			ginkgo.Fail(err.Error())
		}
		out <- buf.String()
	}()

	wg.Wait()

	f()

	writer.Close()
	return <-out
}
