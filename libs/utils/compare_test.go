/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSameSliceSource(t *testing.T) {
	{
		a := []*int{ReferenceOf(1), ReferenceOf(2)}
		b := a
		assert.True(t, IsSameSliceSource(a, b))
	}

	{
		a := []*int{ReferenceOf(1), ReferenceOf(2)}
		b := []*int{ReferenceOf(1), ReferenceOf(2)}
		assert.False(t, IsSameSliceSource(a, b))
	}

	{
		a := []*int{ReferenceOf(1), ReferenceOf(2)}
		b := []*int{a[0], a[1]}
		assert.False(t, IsSameSliceSource(a, b))
	}
}
