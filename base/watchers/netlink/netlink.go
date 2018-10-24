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
package netlink // import "barista.run/base/watchers/netlink"

import (
	"net"
	"sort"
	"strings"
	"sync"

	"barista.run/base/value"
	l "barista.run/logging"
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
	names := []string{link.Name}
	oldLink, ok := links[index]
	if ok {
		if link.Name != oldLink.Name {
			changed = true
			names = append(names, oldLink.Name)
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
	notifyChanged(names...)
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
	notifyChanged(link.Name)
}

func ipPriority(ip net.IP) int {
	priorities := []func(net.IP) bool{
		net.IP.IsGlobalUnicast,
		net.IP.IsLinkLocalUnicast,
		net.IP.IsLinkLocalMulticast,
		net.IP.IsInterfaceLocalMulticast,
		net.IP.IsMulticast,
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
	delete(links, index)
	notifyChanged(link.Name)
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
	notifyChanged(link.Name)
}

func nlInit() {
	initialData, err := getInitialData()
	if err != nil {
		l.Log("Failed to populate initial data: %s", err)
		return
	}
	linksMu.Lock()
	links = initialData
	sorted := sortedLinks()
	linksMu.Unlock()
	msub.Set(sorted)
	go nlListen()
}

var (
	subs   []*Subscription
	msub   value.Value // of []Link
	subsMu sync.RWMutex
)

// Subscription represents a potentially filtered subscription to netlink, which
// returns the best link that matches the filter conditions specified.
type Subscription struct {
	C       <-chan struct{}
	name    string
	prefix  string
	value   value.Value // of Link
	doneSub func()
}

func (s *Subscription) matches(name string) bool {
	switch {
	case s.name != "":
		return s.name == name
	case s.prefix != "":
		return strings.HasPrefix(name, s.prefix)
	default:
		return true
	}
}

func (s *Subscription) notify(links []Link) {
	for _, link := range links {
		if s.matches(link.Name) {
			s.value.Set(link)
			return
		}
	}
	s.value.Set(Link{State: Gone})
}

func sortedLinks() []Link {
	allLinks := []Link{}
	for _, link := range links {
		allLinks = append(allLinks, link)
	}
	sort.Slice(allLinks, func(ai, bi int) bool {
		a, b := allLinks[ai], allLinks[bi]
		switch {
		case a.State > b.State:
			return true
		case a.State < b.State:
			return false
		default:
			return a.Name < b.Name
		}
	})
	return allLinks
}

func notifyChanged(names ...string) {
	allLinks := sortedLinks()
	subsMu.RLock()
	defer subsMu.RUnlock()
	for _, s := range subs {
		for _, n := range names {
			if s.matches(n) {
				s.notify(allLinks)
				break
			}
		}
	}
	msub.Set(allLinks)
}

func subscribe(s *Subscription) *Subscription {
	once.Do(nlInit)
	linksMu.RLock()
	sorted := sortedLinks()
	linksMu.RUnlock()
	subsMu.Lock()
	subs = append(subs, s)
	subsMu.Unlock()
	s.notify(sorted)
	s.C, s.doneSub = s.value.Subscribe()
	return s
}

// Unsubscribe stops further notifications and closes the channel.
func (s *Subscription) Unsubscribe() {
	s.doneSub()
	subsMu.Lock()
	defer subsMu.Unlock()
	for i, sub := range subs {
		if s == sub {
			subs = append(subs[:i], subs[i+1:]...)
			return
		}
	}
}

// ByName creates a netlink watcher for the named interface.
// Any updates to the named interface will cause the current
// information about that link to be sent on the returned channel.
func ByName(name string) *Subscription {
	return subscribe(&Subscription{name: name})
}

// WithPrefix creates a netlink watcher that returns the 'best'
// link beginning with the given prefix (e.g. 'wl' for wireless,
// or 'e' for ethernet). See #Any() for details on link priority.
func WithPrefix(prefix string) *Subscription {
	return subscribe(&Subscription{prefix: prefix})
}

// Any creates a netlink watcher that returns the 'best' link.
// Links are preferred in order of their status, and then by name.
// The status order is Up > Dormant > Testing > LowerLayerDown
// > Down > NotPresent > Unknown. (A 'virtual' link with status
// Gone may be returned if no links are available).
// If multiple links have the same status, they are ordered
// alphabetically by their name.
func Any() *Subscription {
	return subscribe(new(Subscription))
}

// Get returns the most recent Link that matches the subscription conditions.
func (s *Subscription) Get() Link {
	return s.value.Get().(Link)
}

// Next returns a channel that will be closed on the next update.
func (s *Subscription) Next() <-chan struct{} {
	return s.value.Next()
}

// MultiSubscription represents a subscription over all links.
type MultiSubscription struct{}

// All creates a netlink watcher for all links.
func All() MultiSubscription {
	once.Do(nlInit)
	return MultiSubscription{}
}

// Get returns the most recent Link that matches the subscription conditions.
func (s MultiSubscription) Get() []Link {
	if links, ok := msub.Get().([]Link); ok {
		return links
	}
	return nil
}

// Next returns a channel that will be closed on the next update.
func (s MultiSubscription) Next() <-chan struct{} {
	return msub.Next()
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
	oldLink := links[index]
	if link.Name == "" {
		link.Name = oldLink.Name
	}
	if len(link.HardwareAddr) == 0 {
		link.HardwareAddr = oldLink.HardwareAddr
	}
	if link.State == 0 {
		link.State = oldLink.State
	}
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
	msub = value.Value{}
	subsMu.Unlock()
	return &tester{}
}
