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

package db

import (
	"sync"
	"time"

	"github.com/Rugvip/bullettime/interfaces"
	"github.com/Rugvip/bullettime/types"
	"github.com/Rugvip/bullettime/utils"
)

type roomDb struct { // always lock in the same order as below
	roomsLock  sync.RWMutex
	rooms      map[types.RoomId]*dbRoom
	eventsLock sync.RWMutex
	events     map[types.EventId]types.Event
}

func NewRoomDb() (interfaces.RoomStore, error) {
	return &roomDb{
		events: map[types.EventId]types.Event{},
		rooms:  map[types.RoomId]*dbRoom{},
	}, nil
}

type stateId struct {
	EventType string
	StateKey  string
}

type dbRoom struct { // always lock in the same order as below
	id         types.RoomId
	stateLock  sync.RWMutex
	states     map[stateId]*types.State
	eventsLock sync.RWMutex
	events     []types.Event
}

func (db *roomDb) CreateRoom(domain string) (types.RoomId, types.Error) {
	db.roomsLock.Lock()
	defer db.roomsLock.Unlock()
	id := types.NewRoomId("", domain)
	for {
		id.Id.Id = utils.RandomString(16)
		if db.rooms[id] == nil {
			break
		}
	}
	db.rooms[id] = &dbRoom{
		id:     id,
		states: map[stateId]*types.State{},
	}
	return id, nil
}

func (db *roomDb) RoomExists(id types.RoomId) (bool, types.Error) {
	db.roomsLock.RLock()
	defer db.roomsLock.RUnlock()
	if db.rooms[id] == nil {
		return false, nil
	}
	return true, nil
}

func (db *roomDb) AddRoomMessage(roomId types.RoomId, userId types.UserId, content types.TypedContent) (*types.Message, types.Error) {
	db.roomsLock.RLock()
	defer db.roomsLock.RUnlock()
	room := db.rooms[roomId]
	if room == nil {
		return nil, types.NotFoundError("room '" + roomId.String() + "' doesn't exist")
	}
	db.eventsLock.Lock()
	defer db.eventsLock.Unlock()
	var eventId = types.DeriveEventId("", userId.Id)
	for {
		eventId.Id.Id = utils.RandomString(16)
		if db.events[eventId] == nil {
			break
		}
	}
	event := new(types.Message)
	event.EventId = eventId
	event.RoomId = roomId
	event.UserId = userId
	event.EventType = content.GetEventType()
	event.Timestamp = types.Timestamp{time.Now()}
	event.Content = content

	db.events[eventId] = event
	room.eventsLock.Lock()
	defer room.eventsLock.Unlock()
	room.events = append(room.events, event)

	return event, nil
}

func (db *roomDb) SetRoomState(roomId types.RoomId, userId types.UserId, content types.TypedContent, stateKey string) (*types.State, types.Error) {
	db.roomsLock.RLock()
	defer db.roomsLock.RUnlock()
	room := db.rooms[roomId]
	if room == nil {
		return nil, types.NotFoundError("room '" + roomId.String() + "' doesn't exist")
	}
	db.eventsLock.Lock()
	defer db.eventsLock.Unlock()
	var eventId = types.DeriveEventId("", userId.Id)
	for {
		eventId.Id.Id = utils.RandomString(16)
		if db.events[eventId] == nil {
			break
		}
	}
	stateId := stateId{content.GetEventType(), stateKey}

	state := new(types.State)
	state.EventId = eventId
	state.RoomId = roomId
	state.UserId = userId
	state.EventType = content.GetEventType()
	state.StateKey = stateKey
	state.Timestamp = types.Timestamp{time.Now()}
	state.Content = content
	state.OldState = (*types.OldState)(room.states[stateId])

	db.events[eventId] = state
	room.eventsLock.Lock()
	defer room.eventsLock.Unlock()
	room.events = append(room.events, state)
	room.stateLock.Lock()
	defer room.stateLock.Unlock()
	room.states[stateId] = state

	return state, nil
}

func (db *roomDb) RoomState(roomId types.RoomId, eventType, stateKey string) (*types.State, types.Error) {
	db.roomsLock.RLock()
	defer db.roomsLock.RUnlock()
	room := db.rooms[roomId]
	if room == nil {
		return nil, types.NotFoundError("room '" + roomId.String() + "' doesn't exist")
	}
	room.stateLock.RLock()
	defer room.stateLock.RUnlock()
	state := room.states[stateId{eventType, stateKey}]
	return state, nil
}

func (db *roomDb) EntireRoomState(roomId types.RoomId) ([]*types.State, types.Error) {
	db.roomsLock.RLock()
	defer db.roomsLock.RUnlock()
	room := db.rooms[roomId]
	if room == nil {
		return nil, types.NotFoundError("room '" + roomId.String() + "' doesn't exist")
	}
	room.stateLock.RLock()
	defer room.stateLock.RUnlock()
	states := make([]*types.State, 0, len(room.states))
	for _, state := range room.states {
		states = append(states, state)
	}
	return states, nil
}