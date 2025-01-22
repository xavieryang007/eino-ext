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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDereferenceOrZero(t *testing.T) {
	assert.Equal(t, 123, DereferenceOrZero(ReferenceOf(123)))
	assert.Equal(t, "abc", DereferenceOrZero(ReferenceOf("abc")))
	assert.Zero(t, DereferenceOrZero[int](nil))
	assert.Nil(t, DereferenceOrZero[interface{}](nil))
	assert.Nil(t, DereferenceOrZero(ReferenceOf[fmt.Stringer](nil)))
}

func TestReferenceOf(t *testing.T) {
	{
		assert.Equal(t, 123, *ReferenceOf(123))
		assert.Equal(t, "abc", *ReferenceOf("abc"))
		assert.Equal(t, "abc", **ReferenceOf(ReferenceOf("abc")))
		assert.Equal(t, "abc", ***ReferenceOf(ReferenceOf(ReferenceOf("abc"))))
		assert.NotNil(t, ReferenceOf[*int](nil))
		assert.Nil(t, *ReferenceOf[*int](nil))
		assert.Nil(t, *ReferenceOf[*interface{}](nil))
	}

	{
		v := 1
		p := ReferenceOf(v)
		assert.False(t, p == &v)
		*p = 2
		assert.Equal(t, 1, v)
		assert.Equal(t, 2, *p)
	}
}
