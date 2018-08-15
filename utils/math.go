// Copyright 2018. box.la authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package utils

import (
	"github.com/shopspring/decimal"
)

//比较float64
func CompareFloat64(a, b float64) int {
	return decimal.NewFromFloat(a).Cmp(decimal.NewFromFloat(b))
}

//a+b
func AddFloat64(a, b float64) float64 {
	value, _ := decimal.NewFromFloat(a).Add(decimal.NewFromFloat(b)).Float64()
	return value
}

//a-b
func SubFloat64(a, b float64) float64 {
	value, _ := decimal.NewFromFloat(a).Sub(decimal.NewFromFloat(b)).Float64()
	return value
}

//a*b
func MulFloat64(a, b float64) float64 {
	value, _ := decimal.NewFromFloat(a).Mul(decimal.NewFromFloat(b)).Float64()
	return value
}

//a/b
func DivFloat64(a, b float64) float64 {
	value, _ := decimal.NewFromFloat(a).Div(decimal.NewFromFloat(b)).Float64()
	return value
}

// if a<b
func LessThanFloat64(a, b float64) bool {
	return decimal.NewFromFloat(a).LessThan(decimal.NewFromFloat(b))
}

//if a>b
func GreaterThanFloat64(a, b float64) bool {
	return decimal.NewFromFloat(a).GreaterThan(decimal.NewFromFloat(b))
}

//if a=b
func EqualFloat64(a, b float64) bool {
	return decimal.NewFromFloat(a).Equal(decimal.NewFromFloat(b))
}
