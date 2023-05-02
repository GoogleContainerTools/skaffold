// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

//go:build windows
// +build windows

package notify

// eventmask uses ei to create a new event which contains internal flags used by
// notify package logic. If one of FileAction* masks is detected, this function
// adds corresponding FileNotifyChange* values. This allows non registered
// FileAction* events to be passed on.
func eventmask(ei EventInfo, extra Event) (e Event) {
	if e = ei.Event() | extra; e&fileActionAll != 0 {
		if ev, ok := ei.(*event); ok {
			switch ev.ftype {
			case fTypeFile:
				e |= FileNotifyChangeFileName
			case fTypeDirectory:
				e |= FileNotifyChangeDirName
			case fTypeUnknown:
				e |= fileNotifyChangeModified
			}
			return e &^ fileActionAll
		}
	}
	return
}

// matches reports a match only when:
//
//   - for user events, when event is present in the given set
//   - for internal events, when additionally both event and set have omit bit set
//
// Internal events must not be sent to user channels and vice versa.
func matches(set, event Event) bool {
	return (set&omit)^(event&omit) == 0 && (set&event == event || set&fileNotifyChangeModified&event != 0)
}
