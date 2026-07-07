package terminal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type MenuOption[T any] struct {
	Number int
	Label  string
	Value  T
}

type Terminal interface {
	ReadLine() (string, error)
	ConfirmNoDefault(prompt string) (bool, error)
}

type stdTerminal struct{}

var stdinReader = bufio.NewReader(os.Stdin)

func Default() Terminal {
	return stdTerminal{}
}

func SetInput(reader io.Reader) func() {
	originalReader := stdinReader
	stdinReader = bufio.NewReader(reader)
	return func() {
		stdinReader = originalReader
	}
}

func Select[T any](title string, promptRange string, options []MenuOption[T]) (T, error) {
	var zero T
	if len(options) == 0 {
		return zero, fmt.Errorf("菜单选项不能为空")
	}

	actions := make(map[int]T, len(options))
	for _, option := range options {
		actions[option.Number] = option.Value
	}

	for {
		if strings.TrimSpace(title) != "" {
			fmt.Println(title)
		}
		for _, option := range options {
			fmt.Printf("  %d. %s\n", option.Number, option.Label)
		}
		fmt.Printf("请输入操作编号 %s: ", promptRange)
		line, err := ReadLine()
		if err != nil {
			return zero, fmt.Errorf("读取用户输入失败: %w", err)
		}

		value := strings.TrimSpace(line)
		if value == "" {
			fmt.Println("输入不能为空，请输入菜单编号。")
			fmt.Println()
			continue
		}

		number, err := strconv.Atoi(value)
		if err == nil {
			if action, ok := actions[number]; ok {
				return action, nil
			}
		}
		fmt.Println("输入无效，请重新输入。")
		fmt.Println()
	}
}

func ConfirmNoDefault(prompt string) (bool, error) {
	return Default().ConfirmNoDefault(prompt)
}

func (stdTerminal) ConfirmNoDefault(prompt string) (bool, error) {
	fmt.Print(prompt)
	line, err := ReadLine()
	if err != nil {
		return false, fmt.Errorf("读取用户输入失败: %w", err)
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func ReadLine() (string, error) {
	return Default().ReadLine()
}

func (stdTerminal) ReadLine() (string, error) {
	return stdinReader.ReadString('\n')
}
