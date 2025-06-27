//go:build gomobile

package intra

import (
    "split"
    "protect"
)

var (
    _ split.RetryStats
    _ protect.Protector
)
