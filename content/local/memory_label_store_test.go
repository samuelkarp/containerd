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

package local

import (
	"testing"

	digest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDigest digest.Digest = "digest"
)

func TestMemoryLabelStore_Get(t *testing.T) {
	labelStore := NewMemoryLabelStore()

	labels, err := labelStore.Get(testDigest)
	require.NoError(t, err, "get empty")
	assert.Empty(t, labels, "get empty")
}

func TestMemoryLabelStore_Set(t *testing.T) {
	labelStore := NewMemoryLabelStore()

	expectedLabels := map[string]string{
		"label1": "foo",
		"label2": "bar",
	}
	err := labelStore.Set(testDigest, expectedLabels)
	require.NoError(t, err, "set labels")

	labels, err := labelStore.Get(testDigest)
	require.NoError(t, err, "get labels")
	assert.EqualValues(t, expectedLabels, labels, "get labels")

	err = labelStore.Set(testDigest, map[string]string{})
	require.NoError(t, err, "set empty")

	labels, err = labelStore.Get(testDigest)
	require.NoError(t, err, "get empty")
	assert.Empty(t, labels, "get empty")
}

func TestMemoryLabelStore_Update(t *testing.T) {
	labelStore := NewMemoryLabelStore()

	initialLabels := map[string]string{
		"label1": "foo",
		"label2": "bar",
	}
	err := labelStore.Set(testDigest, initialLabels)
	require.NoError(t, err, "set labels")

	updateLabels := map[string]string{
		"label2": "",
		"label3": "baz",
	}
	expectedLabels := map[string]string{
		"label1": "foo",
		"label3": "baz",
	}
	updated, err := labelStore.Update(testDigest, updateLabels)
	require.NoError(t, err, "update labels")
	assert.EqualValues(t, expectedLabels, updated, "update labels")
}
