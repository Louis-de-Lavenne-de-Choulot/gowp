package main

import (
	"fmt"
	"os"
	"plugin"

	_ "github.com/Louis-de-Lavenne-de-Choulot/gowp"
)

func main() {
	// determine plugin to load

	mod := "./admin_plugin.so"

	// load module
	// 1. open the so file to load the symbols
	plug, err := plugin.Open(mod)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 2. look up a symbol (an exported function or variable)
	// in this case, variable Greeter
	symPluginInit, err := plug.Lookup("Init")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 3. Assert that loaded symbol is of a desired type
	// in this case interface type Greeter (defined above)
	pluginInit, ok := symPluginInit.(func() gowp.Route)
	if !ok {
		fmt.Println("unexpected type from module symbol")
		os.Exit(1)
	}

	// 4. use the module
	res := pluginInit()
	fmt.Println(res)

	// for r := range *res {
	// 	fmt.Println(r)
	// }
}
