// Copyright 2017 Tristan Claverie. All rights reserved.
// Use of this source code is governed by an Apache
// license that can be found in the LICENSE file.

/*Package kmod performs bindings over libkmod to manipulate kernel modules from Golang seemlessly.

libkmod is a well-known library to handle kernel modules and which is used in the kmod set of tools (modprobe, modinfo, depmod etc ...). This Golang wrapper exposes common operations: list installed modules, retrieve information on a module, insert or remove a module from the tree.

The following file shows those abilities in practice are available

	package main

	import (
		"fmt"
		"github.com/ElyKar/golang-kmod/kmod"
	)

	func main() {
		km := kmod.NewKmod()

		// List all loaded modules
		for _, module := range km.List() {
			fmt.Printf("%s, %d\n", module.Name(), module.Size())
		}

		// Finds a specific module and display some informations about it
		pcspkr := km.ModuleFromName("pcspkr")
		infoMod := pcspkr.Info()
		fmt.Printf("Author: %s\n", infoMod["author"])
		fmt.Printf("Description: %s\n", infoMod["description"])
		fmt.Printf("License: %s\n", infoMod["license"])

		// Insert a module and its dependencies
		km.Insert("rtl2832")

		// Remove a module and its dependencies if possible
		km.Remove("rtl2832")
	}
*/
package kmod

/*
#cgo LDFLAGS: -L/usr/lib -lkmod -I/usr/include
#include <libkmod.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

// Helper function to get the proper message from an error.
func goStrerror(err C.int) string {
	var msg *C.char
	msg = C.strerror(err)
	return C.GoString(msg)
}

// Wraps a kmod_context inside it. When garbage collected, all module handles will be freed by libkmod.
type Kmod struct {
	ctx *C.struct_kmod_ctx
}

// NewKmod creates a new context from default directories and configuration files. It will search for modules in /lib/modules/`uname -r` and configuration files in /run/modprobe.d, /etc/modprobe.d and /lib/modprobe.d.
//
// This function can panic in case the library encounters a problem for creating and populating the context.
//
// The returned *Kmod must not be discarded, as releasing it will free the underlying C structure and all the modules in the context.
func NewKmod() *Kmod {
	var ctx *C.struct_kmod_ctx

	ctx = C.kmod_new(nil, nil)
	if ctx == nil {
		panic("Kmod: unable to create the kmod_ctx, leaving now")
	}

	if err := C.kmod_load_resources(ctx); err < 0 {
		panic(fmt.Sprintf("Kmod: unable to prepare the kmod_ctx, leaving now - %s", goStrerror(-err)))
	}

	ret := &Kmod{ctx}

	runtime.SetFinalizer(ret, (*Kmod).cleanup)
	return ret
}

// Cleanup the kmod context.
func (kmod *Kmod) cleanup() {
	if kmod.ctx != nil {
		C.kmod_unload_resources(kmod.ctx)
		C.kmod_unref(kmod.ctx)
		kmod.ctx = nil
	}
}

// List returns a slice containing all loaded modules.
//
// The method can panic in case the list can't be retrieved.
func (kmod *Kmod) List() []*Module {
	var list *C.struct_kmod_list
	var err C.int
	err = C.kmod_module_new_from_loaded(kmod.ctx, &list)
	if err < 0 {
		panic(fmt.Sprintf("Kmod: couldn't get the list of modules: %s\n", goStrerror(-err)))
	}

	modList := newModuleList(list)
	return modList.modules
}

// Lookup returns a slice of all modules matching 'alias_name'.
//
// The method can panic in case the lookup fails
func (kmod *Kmod) Lookup(aliasName string) []*Module {
	var list *C.struct_kmod_list
	var err C.int

	cAliasName := C.CString(aliasName)

	err = C.kmod_module_new_from_lookup(kmod.ctx, cAliasName, &list)
	C.free(unsafe.Pointer(cAliasName))
	if err < 0 {
		panic(fmt.Sprintf("Kmod : Failed to lookup %s - %s", aliasName, goStrerror(-err)))
	}

	modList := newModuleList(list)
	return modList.modules
}

// ModuleFromName returns a module handle from its name.
//
// The method panics if the module could not be found.
func (kmod *Kmod) ModuleFromName(name string) *Module {
	var module *C.struct_kmod_module
	cName := C.CString(name)
	err := C.kmod_module_new_from_name(kmod.ctx, cName, &module)
	C.free(unsafe.Pointer(cName))
	if err < 0 {
		panic(fmt.Sprintf("Kmod : Could not get module %s - %s", name, goStrerror(-err)))
	}

	return newModule(module)
}

// Insert a module in the tree with its name.
//
// It panics if the module could not be found or if it could not be inserted.
//
// To insert a wanted module:
//
//     kmod := NewKmod()
//     kmod.Insert("pcspkr")
//
// If this module depends on others that are not yet loaded, depencies will be loaded.
func (kmod *Kmod) Insert(name string) {
	modules := kmod.Lookup(name)
	var err C.int

	for _, module := range modules {
		err = C.kmod_module_probe_insert_module(module.mod, 0, nil, nil, nil, nil)
		if err < 0 {
			panic(fmt.Sprintf("Could not insert module %s : %s", module.Name(), goStrerror(-err)))
		}
	}
}

// Remove a module from the current tree using its name.
//
// It can panic if the module could not be found or could not be removed.
//
// Provided the module pcspkr is loaded and not used:
//
//     kmod := NewKmod()
//     kmod.Remove("pcspkr")
func (kmod *Kmod) Remove(name string) {
	modules := kmod.Lookup(name)
	var err C.int

	for _, module := range modules {
		err = C.kmod_module_remove_module(module.mod, 0)
		if err < 0 {
			panic(fmt.Sprintf("Could not remove module %s : %s", module.Name(), goStrerror(-err)))
		}
	}
}
