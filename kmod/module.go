// Copyright 2017 Tristan Claverie. All rights reserved.
// Use of this source code is governed by an Apache
// license that can be found in the LICENSE file.
package kmod

/*
#cgo LDFLAGS: -lkmod
#include <libkmod.h>
#include <string.h>
#include <stdio.h>
*/
import "C"

import (
	"fmt"
	"runtime"
)

// Version structure for an individual entry in __versions.
type Version struct {
	// An exported symbol of the module
	symbol string
	// Crc for version consistency
	crc uint64
}

// Symbol returns the symbol for this instance of a Version.
func (ver *Version) Symbol() string {
	return ver.symbol
}

// Crc returns the crc for this instance of a Version.
func (ver *Version) Crc() uint64 {
	return ver.crc
}

// ModuleList wraps a kmod_list structure.
type ModuleList struct {
	kmodList *C.struct_kmod_list
	modules  []*Module
}

// Create a new module list and populates it with modules.
//
// When garbage collected, a call to kmod_module_unref_list if performed, which fill free the list and unref all its modules.
func newModuleList(list *C.struct_kmod_list) *ModuleList {
	var modCurr *Module
	var listCurr *C.struct_kmod_list

	modList := &ModuleList{list, make([]*Module, 0)}

	for listCurr = list; listCurr != nil; listCurr = C.kmod_list_next(list, listCurr) {
		modCurr = newModule(C.kmod_module_get_module(listCurr))
		// References to modules are accessible through the Golang slice, hence their ref count needs to be incremented
		C.kmod_module_ref(modCurr.mod)
		modList.modules = append(modList.modules, modCurr)
	}

	runtime.SetFinalizer(modList, (*ModuleList).cleanup)
	return modList
}

// Unref the list and all its modules (done by libkmod).
func (modList *ModuleList) cleanup() {
	if modList.kmodList != nil {
		C.kmod_module_unref_list(modList.kmodList)
	}
}

// Module wraps a module from kmod.
type Module struct {
	mod *C.struct_kmod_module
}

// Creates a new module from the kmod module.
func newModule(mod *C.struct_kmod_module) *Module {
	module := &Module{mod}
	runtime.SetFinalizer(module, (*Module).cleanup)
	return module
}

// Unref the module.
func (mod *Module) cleanup() {
	if mod.mod != nil {
		C.kmod_module_unref(mod.mod)
	}
}

// RefCnt returns the reference count for this module.
func (mod *Module) RefCnt() int32 {
	return int32(C.kmod_module_get_refcnt(mod.mod))
}

// Size returns the size of the module in bytes.
func (mod *Module) Size() uint64 {
	return uint64(C.kmod_module_get_size(mod.mod))
}

// Name returns the name of the module.
func (mod *Module) Name() string {
	return C.GoString(C.kmod_module_get_name(mod.mod))
}

// Path returns the path at which the module is stored.
func (mod *Module) Path() string {
	return C.GoString(C.kmod_module_get_path(mod.mod))
}

// Options returns the options given to the module.
func (mod *Module) Options() string {
	return C.GoString(C.kmod_module_get_options(mod.mod))
}

// InstallCommands returns the install commands of the module.
func (mod *Module) InstallCommands() string {
	return C.GoString(C.kmod_module_get_install_commands(mod.mod))
}

//RemoveCommands returns the remove commands of the module.
func (mod *Module) RemoveCommands() string {
	return C.GoString(C.kmod_module_get_remove_commands(mod.mod))
}

// Info returns the informations about a module (author, description etc ...).
//
// It is susceptible to panic if informations about the module couldn't be read
func (mod *Module) Info() map[string]string {
	var list, listCurr *C.struct_kmod_list
	var key, value string
	info := make(map[string]string)
	err := C.kmod_module_get_info(mod.mod, &list)

	if err < 0 {
		panic(fmt.Sprintf("Kmod : unable to get the module information: %s", goStrerror(-err)))
	}

	for listCurr = list; listCurr != nil; listCurr = C.kmod_list_next(list, listCurr) {
		key = C.GoString(C.kmod_module_info_get_key(listCurr))
		value = C.GoString(C.kmod_module_info_get_value(listCurr))
		info[key] = value
	}

	C.kmod_module_info_free_list(list)
	return info
}

// Versions returns the list of exported symbols for a module.
//
// This method can panic if they could not be retrieved.
func (mod *Module) Versions() []*Version {
	var list, listCurr *C.struct_kmod_list
	var symbol string
	var crc uint64
	versions := make([]*Version, 0)
	err := C.kmod_module_get_versions(mod.mod, &list)

	if err < 0 {
		panic(fmt.Sprintf("Kmod : unable to get the module versions: %s", goStrerror(-err)))
	}

	for listCurr = list; listCurr != nil; listCurr = C.kmod_list_next(list, listCurr) {
		symbol = C.GoString(C.kmod_module_version_get_symbol(listCurr))
		crc = uint64(C.kmod_module_version_get_crc(listCurr))
		versions = append(versions, &Version{symbol: symbol, crc: crc})
	}

	C.kmod_module_versions_free_list(list)
	return versions
}
