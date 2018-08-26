// Copyright 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package outputs

import (
	"testing"

	"github.com/martinlindhe/unit"
	"github.com/stretchr/testify/require"
)

func TestDataSizeFormats(t *testing.T) {
	require := require.New(t)
	require.Equal("10 KiB", IBytesize(10*unit.Kibibyte))
	require.Equal("10 kB", Bytesize(10*unit.Kilobyte))
	require.Equal("9.8 KiB", IBytesize(10*unit.Kilobyte))
}

func TestDataRateFormats(t *testing.T) {
	require := require.New(t)
	require.Equal("10 KiB/s", IByterate(10*unit.KibibytePerSecond))
	require.Equal("10 kB/s", Byterate(10*1000*8*unit.BitPerSecond))
	require.Equal("9.8 KiB/s", IByterate(10*1000*8*unit.BitPerSecond))
}
