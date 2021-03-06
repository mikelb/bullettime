// Copyright 2015  Ericsson AB
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

package events

import (
	"sync"

	"github.com/matrix-org/bullettime/core/types"
	matrixTypes "github.com/matrix-org/bullettime/matrix/types"
)

func NewStreamMux() (*streamMux, matrixTypes.Error) {
	return &streamMux{
		channels: map[types.UserId]userChannels{},
	}, nil
}

type streamMux struct {
	lock     sync.Mutex
	channels map[types.UserId]userChannels
}

type userChannels []chan types.IndexedEvent

func (chs *userChannels) send(event types.IndexedEvent) matrixTypes.Error {
	for _, ch := range *chs {
		ch <- event
		close(ch)
	}
	*chs = (*chs)[:0]
	return nil
}

func (chs *userChannels) close(ch chan types.IndexedEvent) {
	l := len(*chs)
	for i, channel := range *chs {
		if ch == channel {
			close(channel)
			(*chs)[i] = (*chs)[l-1]
			(*chs)[l-1] = nil
			*chs = (*chs)[:l-1]
			break
		}
	}
}

func (chs *userChannels) make() chan types.IndexedEvent {
	channel := make(chan types.IndexedEvent, 1)
	*chs = append(*chs, channel)
	return channel
}

func (s *streamMux) Listen(userId types.UserId, cancel chan struct{}) (chan types.IndexedEvent, matrixTypes.Error) {
	s.lock.Lock()
	chs := s.channels[userId]
	channel := chs.make()
	s.channels[userId] = chs
	s.lock.Unlock()
	go func() {
		<-cancel
		s.lock.Lock()
		if chs2, ok := s.channels[userId]; ok {
			chs2.close(channel)
			if len(chs2) == 0 {
				delete(s.channels, userId)
			} else {
				s.channels[userId] = chs2
			}
		}
		s.lock.Unlock()
	}()
	return channel, nil
}

func (s *streamMux) Send(userIds []types.UserId, event types.IndexedEvent) matrixTypes.Error {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, userId := range userIds {
		if chs, ok := s.channels[userId]; ok {
			if err := chs.send(event); err != nil {
				return err
			}
			delete(s.channels, userId)
		}
	}
	return nil
}
