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

// +build baristadebuglog

package logging

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// ident stores a type+address combination that uniquely identifies an object.
// In go, the address (uintptr) of the first element of a struct is the same
// as that of the struct itself, so the address alone is not sufficient, it must
// be accompanied with type information.
type ident struct {
	typeName string
	address  uintptr
}

// zero returns true if either the type or the address is unknown,
// which means this ident cannot meaningfully identify anything.
func (i ident) zero() bool {
	return i.address == 0 || i.typeName == ""
}

// String implements Stringer for ident.
func (i ident) String() string {
	typeStr := i.typeName
	if typeStr == "" {
		typeStr = "?"
	}
	if i.address == 0 {
		return fmt.Sprintf("%s@?", typeStr)
	}
	return fmt.Sprintf("%s@%x", typeStr, i.address)
}

// typeName returns the shortened name of a type. It also formats some
// built-in types. In particular, it makes no distinction between a send-only,
// receive-only, or bidirectional channel.
func typeName(typ reflect.Type) string {
	if typ == nil {
		return "nil"
	}
	name := typ.Name()
	if typ.PkgPath() != "" {
		path := shorten(typ.PkgPath())
		if strings.HasSuffix(path, name) {
			name = path
		} else {
			name = fmt.Sprintf("%s.%s", path, name)
		}
	}
	if name != "" {
		return name
	}
	switch typ.Kind() {
	case reflect.Chan:
		elemType := typeName(typ.Elem())
		if elemType == "{}" {
			return "chan"
		}
		return fmt.Sprintf("chan %s", elemType)
	case reflect.Slice, reflect.Array:
		return fmt.Sprintf("[]%s", typeName(typ.Elem()))
	case reflect.Map:
		return fmt.Sprintf("[%s]%s", typeName(typ.Key()), typeName(typ.Elem()))
	case reflect.Struct:
		name = typ.String()
		name = strings.Replace(name, "struct ", "", -1)
		name = strings.Replace(name, "{ ", "{", -1)
		name = strings.Replace(name, " }", "}", -1)
		return name
	}
	return typ.String()
}

// identify returns an ident for the given object. The ident may be zero
// if the object cannot be addressed, such as a local variable in a function.
// identify attempts to always identify concrete types by indirecting
// pointers and interfaces.
func identify(thing interface{}) (id ident) {
	var refVal reflect.Value
	if r, ok := thing.(reflect.Value); ok {
		refVal = r
	} else {
		refVal = reflect.ValueOf(thing)
	}
	if !refVal.IsValid() {
		id.typeName = "nil"
		return id
	}
	var interfaceAddr uintptr
	for refVal.Type().Kind() == reflect.Ptr || refVal.Type().Kind() == reflect.Interface {
		if refVal.Type().Kind() == reflect.Interface && refVal.CanAddr() {
			// Special case for addressable instances of primitive interfaces.
			// Golang does not permit addressing a primitive value backing an
			// interface, but in most cases the interface itself will be addressable
			// because it will be embedded in an addressable struct or something.
			interfaceAddr = refVal.UnsafeAddr()
			// Keep going so that at least the type name corresponds to the backing
			// concrete type, even if the address is of a pointer to it.
		}
		refVal = refVal.Elem()
	}
	switch refVal.Type().Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Slice:
		id.address = refVal.Pointer()
	}
	if id.address == 0 && refVal.CanAddr() {
		id.address = refVal.UnsafeAddr()
	}
	if id.address == 0 {
		id.address = interfaceAddr
	}
	id.typeName = typeName(refVal.Type())
	return id
}

var (
	// objectIds stores an ID for each object, based on the first time it's used
	// as a context. When looking at logs, it's a lot easier to deal with chan#1
	// than chan@8375f30, especially when trying to relate objects across logs.
	objectIDs = map[ident]string{}
	// labels keeps track of the current label for an object, to allow a subsequent
	// Label(...) call to remove the current label.
	labels = map[ident]string{}
	// instances keeps track of the number of instances of each type, used when
	// generated IDs for previously unseen objects.
	instances = map[string]int{}
)

var mu sync.Mutex

// Root can be used to attach global objects to when they have no
// obvious or useful parent. The name used for attaching becomes their
// entire name.
var Root = &struct{ thisIsTheRootObject int }{0}
var rootId = identify(Root)

// getName returns the current name for the given identifier. If none
// exists, a new name will be generated based on the typename using a
// sequential counter per type.
func getName(id ident) string {
	if id == rootId {
		return ""
	}
	if id.zero() {
		return id.String()
	}
	if objectID, ok := objectIDs[id]; ok {
		return objectID
	}
	thisInstance, _ := instances[id.typeName]
	instances[id.typeName] = thisInstance + 1
	objectID := fmt.Sprintf("%s#%d", id.typeName, thisInstance)
	objectIDs[id] = objectID
	return objectID
}

// nameAndId returns the current name and an ident for the given object.
func nameAndId(thing interface{}) (name string, id ident) {
	id = identify(thing)
	name = getName(id)
	return name, id
}

// ID returns a unique name for the given value of the form 'type'#'index'
// for addressable types. This provides log statements with additional
// context and separates logs from multiple instances of the same type.
func ID(thing interface{}) string {
	mu.Lock()
	defer mu.Unlock()
	return getName(identify(thing))
}

// node is used to store object hierarchies built by Attach or Register.
// This allows objects to be identified as components of larger objects,
// e.g. timing.Scheduler#5 is less useful than clock.Module#0.typeNameicker.
// Because clock.Module#0 might later be renamed as well, we need to store
// the hierarchies and refresh all associated objects.
// Storing the hierarchy also allows guarding against infinite loops
// caused by attaching an object to one of its descendants.
type node struct {
	name     string
	parent   ident
	children []ident
}

// nodes maintains a node for each identifier, storing its
// name (for recreating names when parent changes),
// parent (to guard against creating cycles),
// and children (to propagate name updates).
var nodes = map[ident]node{}

// hasLoop returns true if parent or any of its ancestors are the same
// identifier as child.
func hasLoop(parent, child ident) bool {
	for {
		if parent == child {
			return true
		}
		parent = nodes[parent].parent
		if parent.zero() {
			return false
		}
	}
}

// updateNode applies a function to the node at the given identifier.
func updateNode(id ident, fn func(*node)) {
	n := nodes[id]
	fn(&n)
	nodes[id] = n
}

// refreshNames stores the given identifier's name, and updates all
// attached descendants' names to reflect the new name.
func refreshNames(id ident, name string) {
	objectIDs[id] = name
	for _, child := range nodes[id].children {
		refreshNames(child, name+nodes[child].name)
	}
}

// Label adds an additional label to thing, incorporated as part of its
// identifier, to provide more useful information than just #0, #1, ...
// For example, a diskspace module might use:
//     logging.Label(m, "sda1")
// which would make its ID mod:diskspace.Module#0<sda1>, making it
// easier to track in logs.
func Label(thing interface{}, label string) {
	mu.Lock()
	defer mu.Unlock()
	thingName, id := nameAndId(thing)
	if id.zero() {
		Log("Cannot add identifier '%s' to %s", label, thingName)
		return
	}
	newName := thingName
	if existingLabel, ok := labels[id]; ok {
		newName = strings.TrimSuffix(newName,
			fmt.Sprintf("<%s>", existingLabel))
	}
	newName += fmt.Sprintf("<%s>", label)
	Fine("%s -> %s", thingName, newName)
	labels[id] = label
	refreshNames(id, newName)
}

// Labelf is Label with built-in formatting. Because all logging functions
// are no-ops without baristadebuglog, having the sprintf be part of the Labelf
// function means that it will only be executed if debug logging is on.
func Labelf(thing interface{}, format string, args ...interface{}) {
	Label(thing, fmt.Sprintf(format, args...))
}

// Attach attaches an object as a named member of a different object.
// This is useful when a generic type (e.g. chan) is used within a more
// specific type (e.g. Module). Typical usage would be:
//     logging.Attach(m, m.scheduler, "refresher")
// where m is a module, m.scheduler is a timing.Scheduler.
// This will make subsequent log statements that use that scheduler as a
// context (even from a different package, e.g. timing) print it as
// module#1.refresher instead of timing/Scheduler#45.
func Attach(parent, child interface{}, name string) {
	if parent == nil {
		parent = Root
	}
	mu.Lock()
	defer mu.Unlock()

	childName, childId := nameAndId(child)
	parentName, parentId := nameAndId(parent)

	if parentId.zero() || childId.zero() {
		Log("Cannot identify %s->%s", parentName, childName)
		return
	}
	if parentId := nodes[childId].parent; !parentId.zero() {
		Log("Cannot reparent %s, already attached to %s", childName, objectIDs[parentId])
		return
	}
	if hasLoop(parentId, childId) {
		Log("%s is a descendant of %s, cannot also be parent", parentName, childName)
		return
	}
	updateNode(childId, func(n *node) {
		n.name = name
		n.parent = parentId
	})
	updateNode(parentId, func(n *node) {
		n.children = append(n.children, childId)
	})
	Fine("%s -> %s%s", childName, parentName, name)
	refreshNames(childId, parentName+name)
}

// Attachf is Attach with built-in formatting.
func Attachf(parent, child interface{}, format string, args ...interface{}) {
	Attach(parent, child, fmt.Sprintf(format, args...))
}

// Register attaches the given fields of a given *struct as '.' + name.
// This is just a shortcut for Register(&thing, &thing.field, ".field")...
// for a set of fields.
func Register(thing interface{}, names ...string) {
	val := reflect.ValueOf(thing)
	for val.Type().Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}
	if val.Type().Kind() != reflect.Struct {
		Log("Ignoring Register(...) for non-struct %+#v", thing)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	thingName, thingId := nameAndId(thing)
	if thingId.zero() {
		Log("Ignoring unaddressable value %s", thingName)
		return
	}
	thingNode := nodes[thingId]
	for _, name := range names {
		child := val.FieldByName(name)
		if !child.IsValid() {
			Log("Could not find %s in %s", name, thingName)
			continue
		}
		oldName, childId := nameAndId(child)
		if hasLoop(thingId, childId) {
			Log("Skipping %s->%s, is an ancestor of %s!", thingName, name, thingName)
			continue
		}
		thingNode.children = append(thingNode.children, childId)
		Fine("%s -> %s.%s", oldName, thingName, name)
		updateNode(childId, func(n *node) {
			n.name = "." + name
			n.parent = thingId
		})
	}
	nodes[thingId] = thingNode
	refreshNames(thingId, thingName)
}
