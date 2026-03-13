package main

import (
	"flag"
	"fmt"
	"os"
)

type CLIArgs struct {
	PhoneNumber   string
	Update        bool
	ListSessions  bool
	DeleteSession string
	ResetSession  string
	Version       bool
	Help          bool
}

func printHelp() {
	fmt.Print(`Usage: mack [options...]
 -p, --phone-number <number>   Phone number (int'l format) to identify/pair a device
 -u, --update                  Pull latest source and rebuild binary
 -l, --list-sessions           List all paired sessions in the database
 -d, --delete-session <number> Permanently delete a session by phone
 -r, --reset-session <number>  Reset a session so it can be re-paired
 -v, --version                 Print version information and exit
 -h, --help                    Show this help screen

Examples:
 mack                             Start the bot (uses stored session)
 mack -p 2348000000000            Pair a new device
 mack -u                          Update to latest version
 mack -l                          Show all saved sessions
`)
	os.Exit(0)
}

func printVersion() {
	fmt.Printf("mack v%s\n  commit: %s\n  built:  %s\n", Version, Commit, BuildDate)
	os.Exit(0)
}

func parseFlags() CLIArgs {
	var args CLIArgs

	flag.Usage = printHelp

	flag.StringVar(&args.PhoneNumber, "p", "", "Phone number to identify or pair a device")
	flag.StringVar(&args.PhoneNumber, "phone-number", "", "Phone number to identify or pair a device")

	flag.BoolVar(&args.Update, "u", false, "Pull latest source and rebuild binary")
	flag.BoolVar(&args.Update, "update", false, "Pull latest source and rebuild binary")

	flag.BoolVar(&args.ListSessions, "l", false, "List all paired sessions in the database")
	flag.BoolVar(&args.ListSessions, "list-sessions", false, "List all paired sessions in the database")

	flag.StringVar(&args.DeleteSession, "d", "", "Permanently delete a session by phone")
	flag.StringVar(&args.DeleteSession, "delete-session", "", "Permanently delete a session by phone")

	flag.StringVar(&args.ResetSession, "r", "", "Reset a session so it can be re-paired")
	flag.StringVar(&args.ResetSession, "reset-session", "", "Reset a session so it can be re-paired")

	flag.BoolVar(&args.Version, "v", false, "Print version information and exit")
	flag.BoolVar(&args.Version, "version", false, "Print version information and exit")

	flag.BoolVar(&args.Help, "h", false, "Show this help screen")
	flag.BoolVar(&args.Help, "help", false, "Show this help screen")

	flag.Parse()

	return args
}
