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

package netlink

import (
	"net"
	"sync"
	"syscall"

	l "barista.run/logging"

	"github.com/vishvananda/netlink/nl"
	"golang.org/x/sys/unix"
)

var native = nl.NativeEndian()

func linkFromMsg(msg []byte) (LinkIndex, Link) {
	ifmsg := nl.DeserializeIfInfomsg(msg)
	linkIndex := LinkIndex(ifmsg.Index)
	linksMu.RLock()
	link := links[linkIndex]
	linksMu.RUnlock()
	attrs, _ := nl.ParseRouteAttr(msg[ifmsg.Len():])
	for _, attr := range attrs {
		switch attr.Attr.Type {
		case unix.IFLA_IFNAME:
			link.Name = string(attr.Value[:len(attr.Value)-1 /* for '\0' */])
		case unix.IFLA_ADDRESS:
			link.HardwareAddr = net.HardwareAddr(attr.Value)
		case unix.IFLA_OPERSTATE:
			link.State = OperState(native.Uint32(attr.Value[0:4]))
		}
	}
	return linkIndex, link
}

func addrFromMsg(msg []byte) (LinkIndex, net.IP) {
	ifmsg := nl.DeserializeIfAddrmsg(msg)
	linkIndex := LinkIndex(ifmsg.Index)
	attrs, _ := nl.ParseRouteAttr(msg[ifmsg.Len():])
	var addr net.IP
	for _, attr := range attrs {
		switch attr.Attr.Type {
		case unix.IFA_LOCAL:
			// Prefer IFA_LOCAL, but fall back to IFA_ADDRESS.
			return linkIndex, net.IP(attr.Value)
		case unix.IFA_ADDRESS:
			addr = net.IP(attr.Value)
		}
	}
	return linkIndex, addr
}

// for tests.
type nlRequest interface {
	AddData(nl.NetlinkRequestData)
	Execute(int, uint16) ([][]byte, error)
}

var newNlRequest = func(proto, flags int) nlRequest {
	return nl.NewNetlinkRequest(proto, flags)
}

type nlReceiver interface {
	Receive() ([]syscall.NetlinkMessage, *unix.SockaddrNetlink, error)
}

var nlSubscribe = func(protocol int, groups ...uint) (nlReceiver, error) {
	return nl.Subscribe(protocol, groups...)
}

var nlMu sync.RWMutex

func getInitialData() (map[LinkIndex]Link, error) {
	links := map[LinkIndex]Link{}
	nlMu.RLock()
	defer nlMu.RUnlock()

	req := newNlRequest(unix.RTM_GETLINK, unix.NLM_F_DUMP)
	req.AddData(nl.NewIfInfomsg(unix.AF_UNSPEC))
	msgs, err := req.Execute(unix.NETLINK_ROUTE, unix.RTM_NEWLINK)
	if err != nil {
		return nil, err
	}
	for _, msg := range msgs {
		idx, link := linkFromMsg(msg)
		l.Fine("Found link %s@%d", link.Name, idx)
		links[idx] = link
	}

	req = newNlRequest(unix.RTM_GETADDR, unix.NLM_F_DUMP)
	req.AddData(nl.NewIfInfomsg(unix.AF_UNSPEC))
	msgs, err = req.Execute(unix.NETLINK_ROUTE, unix.RTM_NEWADDR)
	if err != nil {
		return nil, err
	}
	for _, msg := range msgs {
		idx, addr := addrFromMsg(msg)
		link, ok := links[idx]
		if !ok {
			l.Log("Got address for unknown link %d", idx)
			continue
		}
		l.Fine("Got address %s for %s@%d", addr, link.Name, idx)
		link.IPs = append(link.IPs, addr)
		links[idx] = link
	}

	return links, nil
}

func nlListen() {
	nlMu.RLock()
	s, err := nlSubscribe(
		unix.NETLINK_ROUTE,
		unix.RTNLGRP_LINK,
		unix.RTNLGRP_IPV4_IFADDR,
		unix.RTNLGRP_IPV6_IFADDR,
	)
	nlMu.RUnlock()
	if err != nil {
		l.Log("nl.Subscribe failed: %s", err)
		return
	}
	for {
		msgs, _, err := s.Receive()
		if err != nil {
			l.Log("nl Receive failed: %s", err)
			continue
		}
		for _, msg := range msgs {
			switch msg.Header.Type {
			case unix.RTM_NEWLINK:
				addLink(linkFromMsg(msg.Data))
			case unix.RTM_DELLINK:
				idx, _ := linkFromMsg(msg.Data)
				delLink(idx)
			case unix.RTM_NEWADDR:
				addIP(addrFromMsg(msg.Data))
			case unix.RTM_DELADDR:
				delIP(addrFromMsg(msg.Data))
			}
		}
	}
}
