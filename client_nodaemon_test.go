/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/
package containerd

import (
	"io/ioutil"
	"testing"

	"github.com/containerd/containerd/content/local"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/sys"
)

// TestImagePullNoDaemon tests that images can be pulled to a local content store without a running containerd daemon
// This test runs in short mode and does not require root
func TestImagePullNoDaemon(t *testing.T) {
	root, err := ioutil.TempDir("", "image-pull-no-daemon")
	if err != nil {
		t.Fatal(err)
	}
	defer sys.ForceRemoveAll(root)

	store, err := local.NewLabeledStore(root, local.NewMemoryLabelStore())
	if err != nil {
		panic(err)
	}
	client, err := New("", WithServices(WithContentStore(store)))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx, cancel := testContext()
	defer cancel()
	_, err = client.Pull(ctx, testImage, WithPlatform(platforms.Default()), func(client *Client, remoteContext *RemoteContext) error {
		remoteContext.SkipImageBookkeeping = true
		remoteContext.SkipLease = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
