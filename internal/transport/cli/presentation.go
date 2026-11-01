package cli

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

const soundcheckLogo = `       _____   ____  _    _ _   _ _____   _____ _    _ ______ _____ _  __
      / ____| / __ \| |  | | \ | |  __ \ / ____| |  | |  ____/ ____| |/ /
     | (___  | |  | | |  | |  \| | |  | | |    | |__| | |__ | |    | ' /
      \___ \ | |  | | |  | | . ` + "`" + ` | |  | | |    |  __  |  __|| |    |  <
      ____) || |__| | |__| | |\  | |__| | |____| |  | | |___| |____| . \
     |_____/  \____/ \____/|_| \_|_____/ \_____|_|  |_|______\_____|_|\_\`

func printTerminalLogo(w io.Writer) error {
	f, ok := w.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return nil
	}
	width, _, err := term.GetSize(int(f.Fd()))
	if err != nil || width < 74 {
		return nil
	}
	_, err = fmt.Fprint(w, soundcheckLogo+"\n\n")
	return err
}
