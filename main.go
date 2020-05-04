package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/cbertinato/messagefsv2/fs"
)

func run() error {
	var wg sync.WaitGroup

	// Start file system
	fsServer, err := fs.CreateFileSystem()
	if err != nil {
		return err
	}

	wg.Add(1)
	go func() {
		fsServer.Serve()
		defer fsServer.Unmount()
		defer wg.Done()
	}()

	if err := fsServer.WaitMount(); err != nil {
		return err
	}

	// start network
	// ...

	wg.Wait()

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
