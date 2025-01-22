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

import "unsafe"

// IsSameSliceSource determine whether two tool slices come from the same source.
// Only use it when you're sure that the slice content will not be modified.
// It is useful in specific conditions, like when determine whether tools come from Options.
func IsSameSliceSource[T any](a, b []*T) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	return unsafe.Pointer(&a[0]) == unsafe.Pointer(&b[0])
}
