package fs

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

var files = map[string]string{
	"file":              "content",
	".all/user1":        "test1",
	".all/user2":        "test2",
	"subdir/other-file": "other-content",
}

// The root populates the tree in its OnAdd method
var _ = (fs.NodeOnAdder)((*fsRoot)(nil))

var _ = (fs.InodeEmbedder)((*fsNode)(nil))
var _ = (fs.NodeOpener)((*fsNode)(nil))
var _ = (fs.NodeReader)((*fsNode)(nil))
var _ = (fs.NodeMkdirer)((*fsNode)(nil))
var _ = (fs.NodeGetattrer)((*fsNode)(nil))
var _ = (fs.NodeCreater)((*fsNode)(nil))
var _ = (fs.NodeSetattrer)((*fsNode)(nil))

type fsNode struct {
	fs.Inode
	Data  []byte
	mu    sync.Mutex
	mtime time.Time
}

// fsRoot is the root of the filesystem. Its only function is to populate the filesystem.
type fsRoot struct {
	fs.Inode
}

var root *fsRoot

func (root *fsRoot) OnAdd(ctx context.Context) {
	// p := &root.Inode

	// add .all directory to root
	// allDir := p.NewPersistentInode(ctx, &fsNode{}, fs.StableAttr{Mode: syscall.S_IFDIR})
	// p.AddChild(".all", allDir, true)

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
				ch = p.NewPersistentInode(ctx, &fsNode{}, fs.StableAttr{Mode: syscall.S_IFDIR})
				// Add it
				p.AddChild(component, ch, true)
				log.Printf("Added directory %s", component)
			}

			p = ch
		}

		// Make a file out of the content bytes.
		embedder := &fsNode{
			Data: []byte(content),
		}

		// Create the file.  The inode must be persistent because its lifetime is not under
		// control of the kernel.
		child := p.NewPersistentInode(ctx, embedder, fs.StableAttr{})
		p.AddChild(base, child, true)
		log.Printf("Added file %s", base)
	}
}

func (fn *fsNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (node *fs.Inode, fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	allDir := (&root.Inode).GetChild(".all")
	if peer := allDir.GetChild(name); peer == nil {
		return nil, nil, 0, syscall.EACCES
	}

	// Check that a child node with this name does not exist.
	p := &fn.Inode
	ch := p.GetChild(name)

	if ch == nil {
		embedder := &fsNode{
			mtime: time.Now(),
		}
		ch = p.NewPersistentInode(ctx, embedder, fs.StableAttr{Mode: syscall.S_IFREG})

		// Add it
		p.AddChild(name, ch, true)
	}

	return ch, nil, 0, 0
}

func (fn *fsNode) Open(ctx context.Context, openFlags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	return nil, 0, 0
}

func (fn *fsNode) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	fn.mu.Lock()
	defer fn.mu.Unlock()
	fn.getattr(out)
	return 0
}

func (fn *fsNode) Setattr(ctx context.Context, f fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	fn.mu.Lock()
	defer fn.mu.Unlock()
	fn.getattr(out)
	return 0
}

func (fn *fsNode) getattr(out *fuse.AttrOut) {
	out.Size = uint64(len(fn.Data))
	out.SetTimes(nil, &fn.mtime, nil)
}

func (fn *fsNode) Read(ctx context.Context, fh fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	return fuse.ReadResultData(fn.Data), fs.OK
}

func (fn *fsNode) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	// Check that a child node with this name does not exist.
	p := &fn.Inode
	ch := p.GetChild(name)

	if ch == nil {
		// Create a directory
		ch = p.NewPersistentInode(ctx, &fs.Inode{}, fs.StableAttr{Mode: syscall.S_IFDIR})
		// Add it
		p.AddChild(name, ch, true)
	}

	return ch, 0
}

// CreateFileSystem creates a new FUSE server
func CreateFileSystem() (*fuse.Server, error) {
	mntDir, err := ioutil.TempDir("", "")

	if err != nil {
		return nil, err
	}

	root = &fsRoot{}
	opts := &fs.Options{MountOptions: fuse.MountOptions{Debug: true}}
	server, err := fs.Mount(mntDir, root, opts)

	if err != nil {
		return nil, fmt.Errorf("Mount fail: %v", err)
	}

	log.Printf("Mounted on %s", mntDir)

	return server, nil
}
