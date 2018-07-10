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

// Package netlink uses the netlink library to watch for changes in link states.
package netlink

import (
	"net"
	"sort"
	"strings"
	"sync"

	l "github.com/soumya92/barista/logging"
)

// OperState represents the operating state of a link.
type OperState int

// Operating states, from the IF_OPER_* constants in linux.
const (
	Gone           OperState = -1 // Special value sent when a link is deleted.
	Unknown        OperState = 0
	NotPresent     OperState = 1
	Down           OperState = 2
	LowerLayerDown OperState = 3
	Testing        OperState = 4
	Dormant        OperState = 5
	Up             OperState = 6
)

// LinkIndex is an opaque identifier for a link as used by netlink.
type LinkIndex int

// Link represents a network link.
type Link struct {
	Name         string
	State        OperState
	HardwareAddr net.HardwareAddr
	IPs          []net.IP
}

var (
	once    sync.Once
	links   = map[LinkIndex]Link{}
	linksMu sync.RWMutex
)

func addLink(index LinkIndex, link Link) {
	linksMu.Lock()
	defer linksMu.Unlock()
	changed := false
	oldLink, ok := links[index]
	if ok {
		if link.Name != oldLink.Name {
			changed = true
			notifyChanged(Link{Name: oldLink.Name, State: Gone})
		}
		if link.State != oldLink.State {
			changed = true
		}
		if link.HardwareAddr.String() != oldLink.HardwareAddr.String() {
			changed = true
		}
		if !changed {
			l.Fine("Link %s@%d unchaged, skipping update",
				link.Name, index)
			return
		}
		l.Fine("Updating link %s@%d", link.Name, index)
		// addLink does not have address information
		link.IPs = oldLink.IPs
	} else {
		l.Fine("Adding link %s@%d", link.Name, index)
	}
	links[index] = link
	notifyChanged(link)
}

func addIP(index LinkIndex, addr net.IP) {
	linksMu.Lock()
	defer linksMu.Unlock()
	link, ok := links[index]
	if !ok {
		l.Log("Skipping add IP for unknown link %d", index)
		return
	}
	for _, oldAddr := range link.IPs {
		if oldAddr.Equal(addr) {
			l.Fine("IP %s for %s@%d already present, skipping add",
				addr, link.Name, index)
			return
		}
	}
	l.Fine("Adding IP %s for %s@%d", addr, link.Name, index)
	link.IPs = append(link.IPs, addr)
	// Sort the IPs in a deterministic fashion, prioritising global unicast
	// IPs over link-local, all the way down to loopback and unspecified.
	// (see ipPriority for the complete ordering)
	// There are two reasons for doing this:
	//
	// - This package only exists to support barista modules, which are
	//   more likely to care about displaying "best" IP than about the order
	//   of addition.
	//
	// - We cannot consistently order this list by when IPs were added
	//   because the initial data returns the IPs in an unspecified order
	//   (likely family, v4 before v6).
	sort.Slice(link.IPs, func(ai, bi int) bool {
		a, b := link.IPs[ai], link.IPs[bi]
		priA, priB := ipPriority(a), ipPriority(b)
		switch {
		case priA < priB:
			return true
		case priA > priB:
			return false
		default:
			return a.String() < b.String()
		}
	})
	links[index] = link
	notifyChanged(link)
}

func ipPriority(ip net.IP) int {
	priorities := []func(net.IP) bool{
		net.IP.IsGlobalUnicast,
		net.IP.IsMulticast,
		net.IP.IsInterfaceLocalMulticast,
		net.IP.IsLinkLocalMulticast,
		net.IP.IsLinkLocalUnicast,
		net.IP.IsLoopback,
	}
	for pri, fn := range priorities {
		if fn(ip) {
			return pri
		}
	}
	return len(priorities)
}

func delLink(index LinkIndex) {
	linksMu.Lock()
	defer linksMu.Unlock()
	link, ok := links[index]
	if !ok {
		l.Fine("Skipping delete of unknown link %d", index)
		return
	}
	l.Fine("Deleting link %s@%d", link.Name, index)
	notifyChanged(Link{Name: link.Name, State: Gone})
	delete(links, index)
}

func delIP(index LinkIndex, addr net.IP) {
	linksMu.Lock()
	defer linksMu.Unlock()
	link, ok := links[index]
	if !ok {
		l.Log("Skipping delete IP for unknown link %d", index)
		return
	}
	exists := false
	for idx, oldAddr := range link.IPs {
		if oldAddr.Equal(addr) {
			exists = true
			link.IPs = append(link.IPs[:idx], link.IPs[idx+1:]...)
			break
		}
	}
	if !exists {
		l.Fine("IP %s for %s@%d not present, skipping delete",
			addr, link.Name, index)
		return
	}
	l.Fine("Deleting IP %s for %s@%d", addr, link.Name, index)
	links[index] = link
	notifyChanged(link)
}

func nlInit() {
	initialData, err := getInitialData()
	if err != nil {
		l.Log("Failed to populate initial data: %s", err)
		return
	}
	for _, link := range initialData {
		notifyChanged(link)
	}
	linksMu.Lock()
	links = initialData
	linksMu.Unlock()
	go nlListen()
}

var (
	subs   []subscription
	subsMu sync.RWMutex
)

type subscription struct {
	name       string
	prefix     string
	notifyChan chan<- Link
}

func (s subscription) matches(iface string) bool {
	switch {
	case s.name != "":
		return s.name == iface
	case s.prefix != "":
		return strings.HasPrefix(iface, s.prefix)
	default:
		return true
	}
}

func notifyChanged(l Link) {
	subsMu.RLock()
	defer subsMu.RUnlock()
	for _, s := range subs {
		if s.matches(l.Name) {
			s.notifyChan <- l
		}
	}
}

func subscribe(s subscription) <-chan Link {
	once.Do(nlInit)
	linksMu.RLock()
	defer linksMu.RUnlock()
	subsMu.Lock()
	defer subsMu.Unlock()

	ch := make(chan Link, len(links)+5)
	s.notifyChan = ch
	subs = append(subs, s)
	for _, link := range links {
		if s.matches(link.Name) {
			s.notifyChan <- link
		}
	}
	return ch
}

// ByName creates a netlink watcher for the named interface.
// Any updates to the named interface will cause the current
// information about that link to be sent on the returned channel.
func ByName(name string) <-chan Link {
	return subscribe(subscription{name: name})
}

// WithPrefix creates a netlink watcher that aggregates all links
// the begin with the given prefix (e.g. 'wl' for wireless,
// or 'e' for ethernet).
func WithPrefix(prefix string) <-chan Link {
	return subscribe(subscription{prefix: prefix})
}

// All creates a netlink watcher for all links.
func All() <-chan Link {
	return subscribe(subscription{})
}

// Tester provides methods to simulate netlink messages
// for testing.
type Tester interface {
	AddLink(Link) LinkIndex
	UpdateLink(LinkIndex, Link)
	RemoveLink(LinkIndex)
	AddIP(LinkIndex, net.IP)
	RemoveIP(LinkIndex, net.IP)
}

type tester struct{ lastIdx LinkIndex }

func (t *tester) AddLink(link Link) LinkIndex {
	t.lastIdx++
	addLink(t.lastIdx, link)
	return t.lastIdx
}

func (t *tester) UpdateLink(index LinkIndex, link Link) {
	addLink(index, link)
}

func (t *tester) RemoveLink(index LinkIndex) {
	delLink(index)
}

func (t *tester) AddIP(index LinkIndex, addr net.IP) {
	addIP(index, addr)
}

func (t *tester) RemoveIP(index LinkIndex, addr net.IP) {
	delIP(index, addr)
}

// TestMode puts the netlink watcher in test mode, and resets the
// link and subscriber states.
func TestMode() Tester {
	once.Do(func() {}) // Prevent real subscription.
	linksMu.Lock()
	links = map[LinkIndex]Link{}
	linksMu.Unlock()
	subsMu.Lock()
	subs = nil
	subsMu.Unlock()
	return &tester{}
}
