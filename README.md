### Golang-kmod

Kmod is a set of tools for manipulating Linux Kernel modules. It relies on libkmod, which can be found at:

https://git.kernel.org/pub/scm/utils/kernel/kmod/kmod.git

This project provides Golang bindings to the library libkmod. This way, it is possible to perform module manipulation operations straight from Golang.

### Example

The following example must be run as root, and libkmod must be installed (headers files included).

```go
package main

import (
    "fmt"
    "github.com/ElyKar/golang-kmod/kmod"
)

func main() {
    km, err := kmod.NewKmod()

    if err != nil {
        panic(err)
    }

    // List all loaded modules
    list, err := km.List()
    if err != nil {
        panic(err)
    }

    for _, module := range list {
        fmt.Printf("%s, %d\n", module.Name(), module.Size())
    }

    // Finds a specific module and display some informations about it
    if pcspkr, err := km.ModuleFromName("pcspkr"); err == nil {
        infoMod, err := pcspkr.Info()
        if err != nil {
            panic(err)
        }

        fmt.Printf("Author: %s\n", infoMod["author"])
        fmt.Printf("Description: %s\n", infoMod["description"])
        fmt.Printf("License: %s\n", infoMod["license"])
    }

    // Insert a module and its dependencies
    _ = km.Insert("rtl2832")

    // Remove a module
    _ = km.Remove("rtl2832")
}
```

### From there

This package is really super simple (intended). The complete documentation can be found on Godoc.

