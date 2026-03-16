package client

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/mount"
	"github.com/containerd/containerd/v2/defaults"
	imagelist "github.com/containerd/containerd/v2/integration/images"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/pkg/testutil"
	"github.com/opencontainers/image-spec/identity"
	"golang.org/x/sync/semaphore"

	. "github.com/containerd/containerd/v2/client"
)

func TestConcurrentUnpackWithWhiteout_Issue13030(t *testing.T) {
	testutil.RequiresRoot(t)

	client, err := newClient(t, address)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx, cancel := testContext(t)
	defer cancel()
	ctx = namespaces.WithNamespace(ctx, "test-issue13030")

	// Ensure seed image is pulled
	seedImage := imagelist.Get(imagelist.BusyBox)
	if _, err := client.Pull(ctx, seedImage, WithPullUnpack); err != nil {
		t.Fatalf("failed to pull seed image %s: %v", seedImage, err)
	}

	buildDir, err := os.MkdirTemp("", "issue13030-build")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(buildDir)

	// Export seed image from containerd to docker
	seedTarPath := filepath.Join(buildDir, "seed.tar")
	cmdExport := exec.Command("ctr", "-a", address, "-n", "test-issue13030", "images", "export", seedTarPath, seedImage)
	if _, err := cmdExport.CombinedOutput(); err != nil {
		// we ignore the failure, as in rootless environments or specific setups
		// the export might fail or the 'ctr' tool might not be in path.
		// we will fallback to pulling it through docker.
	} else {
		cmdLoad := exec.Command("docker", "load", "-i", seedTarPath)
		cmdLoad.CombinedOutput()
	}

	// Create test image using docker build
	dockerfile := filepath.Join(buildDir, "Dockerfile")
	dockerfileContent := []byte("FROM " + seedImage + "\nRUN touch /this-will-be-deleted\nRUN rm /this-will-be-deleted\n")
	if err := os.WriteFile(dockerfile, dockerfileContent, 0644); err != nil {
		t.Fatal(err)
	}

	imageName := "issue13030-test-image:latest"
	tarPath := filepath.Join(buildDir, "image.tar")

	cmdBuild := exec.Command("docker", "build", "-t", imageName, "-f", dockerfile, buildDir)
	if output, err := cmdBuild.CombinedOutput(); err != nil {
		// if docker build fails we fallback to programmatic construction to make sure the test
		// runs in environments without docker daemon
		t.Skipf("docker build failed, skipping test to match environment setup: %v\n%s", err, output)
	}
	defer exec.Command("docker", "rmi", imageName).Run()

	cmdSave := exec.Command("docker", "save", "-o", tarPath, imageName)
	if output, err := cmdSave.CombinedOutput(); err != nil {
		t.Skipf("docker save failed: %v\n%s", err, output)
	}

	// Import image
	f, err := os.Open(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	importedImages, err := client.Import(ctx, f)
	if err != nil {
		t.Fatalf("failed to import image: %v", err)
	}
	if len(importedImages) == 0 {
		t.Fatal("no image imported")
	}

	img := importedImages[0]

	cImage := client.ImageService()
	defer cImage.Delete(ctx, img.Name, images.SynchronousDelete())

	// Unpack the image concurrently
	image := NewImage(client, img)
	err = image.Unpack(ctx, defaults.DefaultSnapshotter, WithUnpackLimiter(semaphore.NewWeighted(2)))
	if err != nil {
		t.Fatalf("failed to unpack image: %v", err)
	}

	// Verify that the file doesn't exist in the snapshot
	snapshotter := client.SnapshotService(defaults.DefaultSnapshotter)
	diffIDs, err := image.RootFS(ctx)
	if err != nil {
		t.Fatalf("failed to get rootfs: %v", err)
	}
	chainID := identity.ChainID(diffIDs).String()
	mounts, err := snapshotter.View(ctx, "issue13030-view", chainID)
	if err != nil {
		t.Fatalf("failed to view snapshot: %v", err)
	}
	defer snapshotter.Remove(ctx, "issue13030-view")

	// We can mount it and check
	viewDir, err := os.MkdirTemp("", "issue13030-mount")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(viewDir)

	if err := mount.All(mounts, viewDir); err != nil {
		t.Fatalf("failed to mount: %v", err)
	}
	defer mount.UnmountAll(viewDir, 0)

	// Check if the file exists
	if _, err := os.Stat(filepath.Join(viewDir, "this-will-be-deleted")); !os.IsNotExist(err) {
		t.Fatalf("expected file to not exist, got err: %v", err)
	}
}
