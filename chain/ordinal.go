package chain

// adapted from https://github.com/martinusso/inflect/blob/master/ordinal.go#L38

// The MIT License (MIT)
//
// Copyright (c) 2016 Breno Martinusso
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"fmt"
	"math"
)

const (
	st = "st"
	nd = "nd"
	rd = "rd"
	th = "th"
)

// Ordinal returns the ordinal suffix that should be added to a number to denote the position in an ordered sequence such as 1st, 2nd, 3rd, 4th...
func ordinal(number int) string {
	switch abs(number) % 100 {
	case 11, 12, 13:
		return th
	default:
		switch abs(number) % 10 {
		case 1:
			return st
		case 2:
			return nd
		case 3:
			return rd
		}
	}
	return th
}

func abs(number int) int {
	return int(math.Abs(float64(number)))
}

// Ordinalize turns a number into an ordinal string
func ordinalize(number int) string {
	ordinal := ordinal(number)
	return fmt.Sprintf("%d%s", number, ordinal)
}
