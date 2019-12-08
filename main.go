package main

import (
	"context"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

var files = map[string]string{
	"file":              "content",
	"subdir/other-file": "other-content",
}

// FS is the filesystem root
type FS struct {
	fs.Inode
}

// Ensure that we implement NodeOnAdder
var _ = (fs.NodeOnAdder)((*FS)(nil))

// OnAdd is called when mounting the file system. Use it to populate the file system tree.
func (root *FS) OnAdd(ctx context.Context) {
	for name, content := range files {
		dir, base := filepath.Split(name)

		p := &root.Inode

		// Add directories leading up to the file.
		for _, component := range strings.Split(dir, "/") {
			if len(component) == 0 {
				continue
			}

			// Check that a child node with this name does not exist.
			ch := p.GetChild(component)
			if ch == nil {
				// Create a directory
				ch = p.NewPersistentInode(ctx, &fs.Inode{}, fs.StableAttr{Mode: syscall.S_IFDIR})
				// Add it
				p.AddChild(component, ch, true)
				log.Printf("Added directory %s", component)
			}

			p = ch
		}

		// Make a file out of the content bytes.
		embedder := &fs.MemRegularFile{
			Data: []byte(content),
		}

		// Create the file.  The inode must be persistent because its lifetime is not under
		// control of the kernel.
		child := p.NewPersistentInode(ctx, embedder, fs.StableAttr{})
		p.AddChild(base, child, true)
		log.Printf("Added file %s", base)
	}
}

func main() {
	mntDir, err := ioutil.TempDir("", "")

	if err != nil {
		log.Panic(err)
	}

	root := &FS{}
	opts := &fs.Options{MountOptions: fuse.MountOptions{Debug: true}}
	server, err := fs.Mount(mntDir, root, opts)

	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
	}

	log.Printf("Mounted on %s", mntDir)
	log.Printf("Unmount by calling 'fusermount -u %s'", mntDir)

	// Wait until unmount before exiting
	server.Wait()
}
