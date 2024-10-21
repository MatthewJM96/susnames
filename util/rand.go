package util

import (
	"math/rand"
	"time"
)

var Rnd *rand.Rand = rand.New(rand.NewSource(42))

func RefreshRandSeed() {
	Rnd.Seed(time.Now().UTC().UnixNano())
}
