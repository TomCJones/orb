/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"sync"

	"github.com/trustbloc/orb/pkg/activitypub/vocab"
)

// UndeliverableActivity holds an undeliverable activity along with the URL to which the
// activity could not be delivered.
type UndeliverableActivity struct {
	Activity *vocab.ActivityType
	ToURL    string
}

// UndeliverableHandler implements a mock undeliverable activity handler.
type UndeliverableHandler struct {
	mutex      sync.Mutex
	activities []*UndeliverableActivity
}

// NewUndeliverableHandler returns a mock undeliverable activity handler.
func NewUndeliverableHandler() *UndeliverableHandler {
	return &UndeliverableHandler{}
}

// HandleUndeliverableActivity adds the given undeliverable activity to a map that may be later queried by unit tests.
func (h *UndeliverableHandler) HandleUndeliverableActivity(activity *vocab.ActivityType, toURL string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.activities = append(h.activities, &UndeliverableActivity{
		Activity: activity,
		ToURL:    toURL,
	})
}

// Activities returns the undeliverable activities.
func (h *UndeliverableHandler) Activities() []*UndeliverableActivity {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	return h.activities
}
