// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package diff

import "github.com/pgavlin/text"

// This file exports some private declarations to tests.

func LineEdits[S text.String](src S, edits []Edit[S]) ([]Edit[S], error) {
	return lineEdits(src, edits)
}
