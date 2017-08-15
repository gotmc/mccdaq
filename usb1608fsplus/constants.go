// Copyright (c) 2016-2017 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package usb1608fsplus

// Since each binary encoded value is 16-bits (2 bytes), the converter value is
// 0x8000, which is 32768.
const (
	maxFrequency     = 500000
	defaultFrequency = 10000
	bytesPerWord     = 2
	converter        = 32768
)
