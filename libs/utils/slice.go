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

// Chunk splits a slice into length-n chunks and returns chunks by a newly allocated slice.
// EXAMPLE:
// Chunk([]int{0, 1, 2, 3, 4}, 2) -> [][]int{{0, 1}, {2, 3}, {4}}
func Chunk[T any](slice []T, size int) [][]T {
	if size <= 0 {
		return nil
	}

	var chunks [][]T
	for size < len(slice) {
		slice, chunks = slice[size:], append(chunks, slice[0:size:size])
	}

	if len(slice) > 0 {
		chunks = append(chunks, slice)
	}

	return chunks
}
