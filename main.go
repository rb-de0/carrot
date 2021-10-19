package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

func compile(file string) error {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	input := string(bytes)
	token := Tokenize(input)
	parserContext := ParserContext{currentToken: token}
	node := parserContext.Parse()
	ir := GenerateIR(node)
	tmpDir, err := ioutil.TempDir("", ".build")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(tmpDir+"/main.ll", []byte(ir), 0666)
	if err != nil {
		panic(err)
	}
	clangArgs := []string{
		"-Wno-override-module",
		tmpDir + "/main.ll",
		"-o", "output",
	}
	cmd := exec.Command("clang", clangArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	if len(output) > 0 {
		return errors.New("error")
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("no input file")
		os.Exit(-1)
	}
	file := os.Args[1]
	err := compile(file)
	if err != nil {
		fmt.Println(err)
	}
}
