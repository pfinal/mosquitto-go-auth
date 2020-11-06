package backends

import (
	"errors"
	"io/ioutil"
	"time"

	"github.com/robertkrimen/otto"
)

type JsRunner struct {
	stackDepthLimit int
	msMaxDuration   int64
}

var Halt = errors.New("exceeded max execution time")

func NewJsRunner(stackDepthLimit int, msMaxDuration int64) *JsRunner {
	return &JsRunner{
		stackDepthLimit: stackDepthLimit,
		msMaxDuration:   msMaxDuration,
	}
}

func loadScript(path string) (string, error) {
	script, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(script), nil
}

func (o *JsRunner) runScript(script string, params map[string]interface{}) (granted bool, err error) {
	// The VM is not thread-safe, so we need to create a new VM on every run.
	// TODO: This could be enhanced by having a pool of VMs.
	vm := otto.New()
	vm.SetStackDepthLimit(o.stackDepthLimit)
	vm.Interrupt = make(chan func(), 1)

	defer func() {
		if caught := recover(); caught != nil {
			if caught == Halt {
				granted = false
				err = Halt
				return
			}
			panic(caught)
		}
	}()

	go func() {
		time.Sleep(time.Duration(o.msMaxDuration) * time.Millisecond)
		vm.Interrupt <- func() {
			panic(Halt)
		}
	}()

	for k, v := range params {
		vm.Set(k, v)
	}

	val, err := vm.Run(script)
	if err != nil {
		return false, err
	}

	granted, err = val.ToBoolean()
	if err != nil {
		return false, err
	}

	return
}
