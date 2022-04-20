package logging

import (
	"fmt"
)

type Logger struct{}

func (log *Logger) Error(msg string, args ...interface{}) {
	fmt.Print("[ERROR] ", msg)
	if len(args) > 0 {
		fmt.Print(": ")
		fmt.Println(args...)
	}
}

func (log *Logger) Errorf(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

func (log *Logger) Warn(msg string, args ...interface{}) {
	fmt.Print("[WARNING] ", msg)
	if len(args) > 0 {
		fmt.Print(": ")
		fmt.Println(args...)
	}
}

func (log *Logger) Warnf(format string, args ...interface{}) {
	fmt.Printf("[WARNING] "+format+"\n", args...)
}

func (log *Logger) Info(msg string, args ...interface{}) {
	fmt.Print("[INFO] ", msg)
	if len(args) > 0 {
		fmt.Print(": ")
		fmt.Println(args...)
	}
}

func (log *Logger) Infof(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}
