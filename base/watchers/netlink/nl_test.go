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
	"errors"
	"net"
	"syscall"

	"github.com/vishvananda/netlink/nl"
	"golang.org/x/sys/unix"
)

type testNlSubscriber struct {
	msgChan   <-chan syscall.NetlinkMessage
	errorChan <-chan error
}

func (t *testNlSubscriber) Receive() ([]syscall.NetlinkMessage, *unix.SockaddrNetlink, error) {
	select {
	case m := <-t.msgChan:
		return []syscall.NetlinkMessage{m}, nil, nil
	case e := <-t.errorChan:
		return nil, nil, e
	}
}

func returnTestSubscriber() (msgs chan<- syscall.NetlinkMessage, errs chan<- error) {
	m := make(chan syscall.NetlinkMessage)
	e := make(chan error)
	nlMu.Lock()
	defer nlMu.Unlock()
	nlSubscribe = func(int, ...uint) (nlReceiver, error) {
		return &testNlSubscriber{m, e}, nil
	}
	return m, e
}

func returnCustomSubscriber(subFn func(int, ...uint) (nlReceiver, error)) {
	nlMu.Lock()
	defer nlMu.Unlock()
	nlSubscribe = subFn
}

type testNlRequest struct {
	msgs []syscall.NetlinkMessage
	err  error
}

func (t testNlRequest) AddData(nl.NetlinkRequestData) {}

func (t testNlRequest) Execute(int, uint16) ([][]byte, error) {
	data := [][]byte{}
	for _, msg := range t.msgs {
		data = append(data, msg.Data)
	}
	return data, t.err
}

func setInitialData(getLinks, getAddrs testNlRequest) {
	nlMu.Lock()
	defer nlMu.Unlock()
	newNlRequest = func(proto, flags int) nlRequest {
		switch proto {
		case unix.RTM_GETLINK:
			return getLinks
		case unix.RTM_GETADDR:
			return getAddrs
		default:
			return testNlRequest{nil, errors.New("unexpected request")}
		}
	}
}

func makeNetlinkMessage(
	headerType uint16,
	data nl.NetlinkRequestData,
	attrs ...*nl.RtAttr,
) syscall.NetlinkMessage {
	m := syscall.NetlinkMessage{}
	m.Header.Type = headerType
	m.Data = data.Serialize()
	for _, attr := range attrs {
		m.Data = append(m.Data, attr.Serialize()...)
	}
	return m
}

func msgNewLink(linkIdx int, l Link) syscall.NetlinkMessage {
	data := nl.NewIfInfomsg(unix.AF_UNSPEC)
	data.Index = int32(linkIdx)
	return makeNetlinkMessage(
		unix.RTM_NEWLINK,
		data,
		nl.NewRtAttr(unix.IFLA_IFNAME, append([]byte(l.Name), 0)),
		nl.NewRtAttr(unix.IFLA_ADDRESS, l.HardwareAddr),
		nl.NewRtAttr(unix.IFLA_OPERSTATE, []byte{byte(l.State)}),
	)
}

func msgNewAddrs(linkIdx int, addr, localAddr net.IP) syscall.NetlinkMessage {
	attrs := []*nl.RtAttr{nl.NewRtAttr(unix.IFA_ADDRESS, addr)}
	if localAddr != nil {
		attrs = append(attrs, nl.NewRtAttr(unix.IFA_LOCAL, localAddr))
	}
	data := nl.NewIfAddrmsg(nl.GetIPFamily(addr))
	data.Index = uint32(linkIdx)
	return makeNetlinkMessage(
		unix.RTM_NEWADDR,
		data,
		attrs...,
	)
}

func msgDelLink(linkIdx int, l Link) syscall.NetlinkMessage {
	m := msgNewLink(linkIdx, l)
	m.Header.Type = unix.RTM_DELLINK
	return m
}

func msgDelAddrs(linkIdx int, addr, localAddr net.IP) syscall.NetlinkMessage {
	m := msgNewAddrs(linkIdx, addr, localAddr)
	m.Header.Type = unix.RTM_DELADDR
	return m
}
