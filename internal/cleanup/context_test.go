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

package cleanup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBackground(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var k struct{}
	v := "incontext"
	ctx = context.WithValue(ctx, k, v) //nolint:staticcheck

	assert.Nil(t, contextError(ctx))
	assert.Equal(t, ctx.Value(k), v)

	cancel()
	assert.Error(t, contextError(ctx))
	assert.Equal(t, ctx.Value(k), v)

	// cleanup context should no longer be canceled
	ctx = Background(ctx)
	assert.Nil(t, contextError(ctx))
	assert.Equal(t, ctx.Value(k), v)

	// cleanup contexts can be rewrapped in cancel context
	ctx, cancel = context.WithCancel(ctx)
	cancel()
	assert.Error(t, contextError(ctx))
	assert.Equal(t, ctx.Value(k), v)
}

func contextError(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}