/* ipp-usb - HTTP reverse proxy, backed by IPP-over-USB connection to device
 *
 * Copyright (C) 2020 and up by Alexander Pevzner (pzz@apevzner.com)
 * See LICENSE for license terms and conditions
 *
 * DNS-SD publisher: system-independent stuff
 */

package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// DNSSdTxtItem represents a single TXT record item
type DNSSdTxtItem struct {
	Key, Value string // TXT entry: Key=Value
	URL        bool   // It's an URL, hostname must be adjusted
}

// DNSSdTxtRecord represents a TXT record
type DNSSdTxtRecord []DNSSdTxtItem

// Add adds regular (non-URL) item to DNSSdTxtRecord
func (txt *DNSSdTxtRecord) Add(key, value string) {
	*txt = append(*txt, DNSSdTxtItem{key, value, false})
}

// AddURL adds URL item to DNSSdTxtRecord
func (txt *DNSSdTxtRecord) AddURL(key, value string) {
	*txt = append(*txt, DNSSdTxtItem{key, value, true})
}

// AddPDL adds PDL list (list of supported Page Description Languages, i.e.,
// document formats) to the DNSSdTxtRecord.
//
// Sometimes the PDL list that comes from device, is too large to fit
// TXT record (key=value pair must not exceed 255 bytes). At this case
// we take only as much as possible leading entries of the device-supplied
// list in hope that firmware is smart enough to place most common PDLs
// to the beginning of the list, while more exotic entries goes to the end
func (txt *DNSSdTxtRecord) AddPDL(key, value string) {
	// How many space we have for value? Is it enough?
	max := 255 - len(key) - 1
	if max >= len(value) {
		txt.Add(key, value)
		return
	}

	// Safety check
	if max <= 0 {
		return
	}

	// Truncate the value to fit available space
	value = value[:max+1]
	i := strings.LastIndexByte(value, ',')
	if i < 0 {
		return
	}

	value = value[:i]
	txt.Add(key, value)
}

// IfNotEmpty adds item to DNSSdTxtRecord if its value is not empty
//
// It returns true if item was actually added, false otherwise
func (txt *DNSSdTxtRecord) IfNotEmpty(key, value string) bool {
	if value != "" {
		txt.Add(key, value)
		return true
	}
	return false
}

// URLIfNotEmpty works as IfNotEmpty, but for URLs
func (txt *DNSSdTxtRecord) URLIfNotEmpty(key, value string) bool {
	if value != "" {
		txt.AddURL(key, value)
		return true
	}
	return false
}

// export DNSSdTxtRecord into Avahi format
func (txt DNSSdTxtRecord) export() [][]byte {
	var exported [][]byte

	// Note, for a some strange reason, Avahi published
	// TXT record in reverse order, so compensate it here
	for i := len(txt) - 1; i >= 0; i-- {
		item := txt[i]
		exported = append(exported, []byte(item.Key+"="+item.Value))
	}

	return exported
}

// DNSSdSvcInfo represents a DNS-SD service information
type DNSSdSvcInfo struct {
	Instance string         // If not "", override common instance name
	Type     string         // Service type, i.e. "_ipp._tcp"
	SubTypes []string       // Service subtypes, if any
	Port     int            // TCP port
	Txt      DNSSdTxtRecord // TXT record
	Loopback bool           // Advertise only on loopback interface
}

// DNSSdServices represents a collection of DNS-SD services
type DNSSdServices []DNSSdSvcInfo

// Add DNSSdSvcInfo to DNSSdServices
func (services *DNSSdServices) Add(srv DNSSdSvcInfo) {
	*services = append(*services, srv)
}

// DNSSdPublisher represents a DNS-SD service publisher
// One publisher may publish multiple services unser the
// same Service Instance Name
type DNSSdPublisher struct {
	Log      *Logger        // Device's logger
	DevState *DevState      // Device persistent state
	Services DNSSdServices  // Registered services
	fin      chan struct{}  // Closed to terminate publisher goroutine
	finDone  sync.WaitGroup // To wait for goroutine termination
	sysdep   *dnssdSysdep   // System-dependent stuff
}

// DNSSdStatus represents DNS-SD publisher status
type DNSSdStatus int

const (
	// DNSSdNoStatus is used to indicate that status is
	// not known (yet)
	DNSSdNoStatus DNSSdStatus = iota

	// DNSSdCollision indicates instance name collision
	DNSSdCollision

	// DNSSdFailure indicates publisher failure with any
	// other reason that listed before
	DNSSdFailure

	// DNSSdSuccess indicates successful status
	DNSSdSuccess
)

// String returns human-readable representation of DNSSdStatus
func (status DNSSdStatus) String() string {
	switch status {
	case DNSSdNoStatus:
		return "DNSSdNoStatus"
	case DNSSdCollision:
		return "DNSSdCollision"
	case DNSSdFailure:
		return "DNSSdFailure"
	case DNSSdSuccess:
		return "DNSSdSuccess"
	}

	return fmt.Sprintf("Unknown DNSSdStatus %d", status)
}

// NewDNSSdPublisher creates new DNSSdPublisher
//
// Service instance name comes from the DevState, and if
// name changes as result of name collision resolution,
// DevState will be updated
func NewDNSSdPublisher(log *Logger,
	devstate *DevState, services DNSSdServices) *DNSSdPublisher {

	return &DNSSdPublisher{
		Log:      log,
		DevState: devstate,
		Services: services,
		fin:      make(chan struct{}),
	}
}

// Publish all services
func (publisher *DNSSdPublisher) Publish() error {
	instance := publisher.instance(0)
	publisher.sysdep = newDnssdSysdep(publisher.Log, instance,
		publisher.Services)

	publisher.Log.Info('+', "DNS-SD: %s: publishing requested", instance)

	publisher.finDone.Add(1)
	go publisher.goroutine()

	return nil
}

// Unpublish everything
func (publisher *DNSSdPublisher) Unpublish() {
	close(publisher.fin)
	publisher.finDone.Wait()

	publisher.sysdep.Halt()

	publisher.Log.Info('-', "DNS-SD: %s: removed", publisher.instance(0))
}

// Build service instance name with optional collision-resolution suffix
func (publisher *DNSSdPublisher) instance(suffix int) string {
	name := publisher.DevState.DNSSdName
	strSuffix := ""

	switch {
	// This happens when we try to resolve name conflict
	case suffix != 0:
		strSuffix = fmt.Sprintf(" (USB %d)", suffix)

	// This happens when we've just initialized or reset DNSSdOverride,
	// so append "(USB)" suffix
	case publisher.DevState.DNSSdName == publisher.DevState.DNSSdOverride:
		strSuffix = " (USB)"

	// Otherwise, DNSSdOverride contains saved conflict-resolved device name
	default:
		name = publisher.DevState.DNSSdOverride
	}

	const MaxDNSSDName = 63
	if len(name)+len(strSuffix) > MaxDNSSDName {
		name = name[:MaxDNSSDName-len(strSuffix)]
	}

	return name + strSuffix
}

// Event handling goroutine
func (publisher *DNSSdPublisher) goroutine() {
	// Catch panics to log
	defer func() {
		v := recover()
		if v != nil {
			Log.Panic(v)
		}
	}()

	defer publisher.finDone.Done()

	timer := time.NewTimer(time.Hour)
	timer.Stop()       // Not ticking now
	defer timer.Stop() // And cleanup at return

	var err error
	var suffix int

	instance := publisher.instance(0)
	for {
		fail := false

		select {
		case <-publisher.fin:
			return

		case status := <-publisher.sysdep.Chan():
			switch status {
			case DNSSdSuccess:
				publisher.Log.Info(' ', "DNS-SD: %s: published", instance)
				if instance != publisher.DevState.DNSSdOverride {
					publisher.DevState.DNSSdOverride = instance
					publisher.DevState.Save()
				}

			case DNSSdCollision:
				publisher.Log.Error(' ', "DNS-SD: %s: name collision",
					instance)
				suffix++
				fallthrough

			case DNSSdFailure:
				publisher.Log.Error(' ', "DNS-SD: %s: publishing failed",
					instance)

				fail = true
				publisher.sysdep.Halt()

			default:
				publisher.Log.Error(' ', "DNS-SD: %s: unknown event %s",
					instance, status)
			}

		case <-timer.C:
			instance = publisher.instance(suffix)
			publisher.sysdep = newDnssdSysdep(publisher.Log,
				instance, publisher.Services)

			if err != nil {
				publisher.Log.Error('!', "DNS-SD: %s: %s", instance, err)
				fail = true
			}
		}

		if fail {
			timer.Reset(DNSSdRetryInterval)
		}
	}
}
