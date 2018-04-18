package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/robertkrimen/otto"
)

type JavascriptEngine struct {
	vm *otto.Otto // Our javascript engine (Google V8)
}

func (je *JavascriptEngine) Load(sourceFile string) error {
	je.vm = otto.New()
	script, err := je.vm.Compile(sourceFile, mustReadFile(sourceFile))
	if err != nil {
		log.Fatal(fmt.Sprintf("error code : %s %s", sourceFile, err))
	}
	je.vm.Run(script)

	return err
}

func (je *JavascriptEngine) Call(code string, argumentList ...interface{}) (otto.Value, error) {
	value, error := je.vm.Call(code, nil, argumentList)

	return value, error
}

func mustReadFile(source string) string {
	buf, err := ioutil.ReadFile(source)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to read file: %s", source))
		return ""
	}
	return string(buf)
}
