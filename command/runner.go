package command

import (
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/dty1er/kubecolor/color"
	"github.com/dty1er/kubecolor/kubectl"
	"github.com/dty1er/kubecolor/printer"
	"github.com/mattn/go-colorable"
)

var (
	Stdout = colorable.NewColorableStdout()
	Stderr = colorable.NewColorableStderr()
)

type Printers struct {
	FullColoredPrinter printer.Printer
	ErrorPrinter       printer.Printer
}

// This is defined here to be replaced in test
var getPrinters = func(subcommandInfo *kubectl.SubcommandInfo, darkBackground bool) *Printers {
	return &Printers{
		FullColoredPrinter: &printer.KubectlOutputColoredPrinter{
			SubcommandInfo: subcommandInfo,
			DarkBackground: darkBackground,
			Recursive:      subcommandInfo.Recursive,
		},
		ErrorPrinter: &printer.WithFuncPrinter{
			Fn: func(line string) color.Color {
				if strings.HasPrefix(strings.ToLower(line), "error") {
					return color.Red
				}

				return color.Yellow
			},
		},
	}
}

func Run(args []string) error {
	args, config := ResolveConfig(args)
	shouldColorize, subcommandInfo := ResolveSubcommand(args, config)

	cmd := exec.Command(config.KubectlCmd, args...)
	cmd.Stdin = os.Stdin

	// when should not colorize, just run command and return
	if !shouldColorize {
		cmd.Stdout = Stdout
		cmd.Stderr = Stderr
		if err := cmd.Start(); err != nil {
			return err
		}

		cmd.Wait()
		return nil
	}

	// when colorize, capture stdout and err then colorize it
	outReader, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	errReader, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	printers := getPrinters(subcommandInfo, config.DarkBackground)

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		printers.FullColoredPrinter.Print(outReader, Stdout)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		printers.ErrorPrinter.Print(errReader, Stderr)
		wg.Done()
	}()

	wg.Wait()
	cmd.Wait()

	return nil
}
