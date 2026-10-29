package log

import ct "github.com/daviddengcn/go-colortext"

type ConsoleColor struct {
	colorBase ct.Color
	bright    bool
}

var (
	None        = ConsoleColor{}
	Black       = ConsoleColor{colorBase: ct.Black}
	DarkRed     = ConsoleColor{colorBase: ct.Red}
	DarkGreen   = ConsoleColor{colorBase: ct.Green}
	DarkYellow  = ConsoleColor{colorBase: ct.Yellow}
	DarkBlue    = ConsoleColor{colorBase: ct.Blue}
	DarkMagenta = ConsoleColor{colorBase: ct.Magenta}
	DarkCyan    = ConsoleColor{colorBase: ct.Cyan}
	Gray        = ConsoleColor{colorBase: ct.White}
	DarkGray    = ConsoleColor{colorBase: ct.Black, bright: true}
	Red         = ConsoleColor{colorBase: ct.Red, bright: true}
	Green       = ConsoleColor{colorBase: ct.Green, bright: true}
	Yellow      = ConsoleColor{colorBase: ct.Yellow, bright: true}
	Blue        = ConsoleColor{colorBase: ct.Blue, bright: true}
	Magenta     = ConsoleColor{colorBase: ct.Magenta, bright: true}
	Cyan        = ConsoleColor{colorBase: ct.Cyan, bright: true}
	White       = ConsoleColor{colorBase: ct.White, bright: true}
)

func SetColorFg(fg ConsoleColor) {
	ct.Foreground(fg.colorBase, fg.bright)
}

func SetColorBg(bg ConsoleColor) {
	ct.Background(bg.colorBase, bg.bright)
}

func SetColor(fg, bg ConsoleColor) {
	ct.ChangeColor(fg.colorBase, fg.bright, bg.colorBase, bg.bright)
}

func ResetColor() {
	ct.ResetColor()
}
