/*
Copyright 2021 The Skaffold Authors
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

package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func IntMin(a *big.Int, b *big.Int) (*big.Int, bool) {
	cmp := a.Cmp(b)

	switch cmp {
	case 1:
		return b, false
	case -1:
		return a, false
	case 0:
		return big.NewInt(0), true
	default:
		break
	}

	return nil, false
}

func main() {
	x, _ := rand.Int(rand.Reader, big.NewInt(100))
	y, _ := rand.Int(rand.Reader, big.NewInt(100))

	min, ok := IntMin(x, y)
	if ok {
		fmt.Println(x, " and ", y, " are equal")
	} else if min != nil {
		fmt.Println("Min of ", x, " and ", y, " is: ", min)
	}
}
