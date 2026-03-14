package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/zkyrnx11/mack/src/plugins"
	"github.com/zkyrnx11/mack/src/store"
	"github.com/zkyrnx11/mack/src/store/sqlstore"

	"go.mau.fi/whatsmeow"
	waCompanionReg "go.mau.fi/whatsmeow/proto/waCompanionReg"
	waWa6 "go.mau.fi/whatsmeow/proto/waWa6"
	waStore "go.mau.fi/whatsmeow/store"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	_ "modernc.org/sqlite"
)

var sourceDir string

func dataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	dir := filepath.Join(home, "Documents", "Mack-Bot")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "warn: could not create data dir %s: %v\n", dir, err)
		return "."
	}
	return dir
}

func dbConfig() (string, string) {
	path := filepath.Join(dataDir(), "database.db")
	addr := "file:" + path +
		"?_pragma=foreign_keys(1)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_pragma=busy_timeout(10000)" +
		"&_pragma=cache_size(-2000)" +
		"&_pragma=mmap_size(0)" +
		"&_pragma=temp_store(MEMORY)"
	return "sqlite", addr
}

func getDevice(ctx context.Context, container *sqlstore.Container, phone string) (*store.Device, error) {
	if phone == "" {
		return container.GetFirstDevice(ctx)
	}

	devices, err := container.GetAllDevices(ctx)
	if err != nil {
		return nil, err
	}
	for _, dev := range devices {
		if dev.ID == nil {
			continue
		}
		userPhone := strings.SplitN(dev.ID.User, ".", 2)[0]
		if userPhone == phone {
			return dev, nil
		}
	}
	return container.NewDevice(), nil
}

func main() {
	args := parseFlags()

	if args.Help {
		printHelp()
	}
	if args.Version {
		printVersion()
	}

	ctx := context.Background()

	if args.Update {
		runUpdate()
		return
	}

	dialect, dbAddr := dbConfig()

	if args.ListSessions {
		runListSessions(ctx, dialect, dbAddr)
		return
	}
	if args.DeleteSession != "" {
		runDeleteSession(ctx, dialect, dbAddr, args.DeleteSession, false)
		return
	}
	if args.ResetSession != "" {
		runDeleteSession(ctx, dialect, dbAddr, args.ResetSession, true)
		return
	}

	dbLog := waLog.Stdout("Database", "ERROR", true)

	container, err := sqlstore.New(ctx, dialect, dbAddr, dbLog)
	if err != nil {
		panic(err)
	}

	if err := plugins.InitDB(container.DB()); err != nil {
		panic(fmt.Errorf("settings db init: %w", err))
	}

	plugins.InitLIDStore(container.LIDMap, "")

	deviceStore, err := getDevice(ctx, container, args.PhoneNumber)
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	waStore.DeviceProps.Os = proto.String("WhatsApp")
	waStore.DeviceProps.PlatformType = waCompanionReg.DeviceProps_ANDROID_PHONE.Enum()
	waStore.BaseClientPayload.UserAgent.Platform = waWa6.ClientPayload_UserAgent_ANDROID.Enum()
	waStore.BaseClientPayload.UserAgent.OsVersion = proto.String("")
	waStore.BaseClientPayload.UserAgent.OsBuildNumber = proto.String("")
	waStore.BaseClientPayload.WebInfo = nil
	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(plugins.NewHandler(client))

	if err := container.LIDMap.FillCache(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "warn: FillCache: %v\n", err)
	}

	plugins.InitSourceDir(sourceDir)
	plugins.SetRestartFunc(func() {
		client.Disconnect()
		exe, _ := os.Executable()
		exe, _ = filepath.EvalSymlinks(exe)
		cmd := exec.Command(exe, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Start()
		os.Exit(0)
	})

	fmt.Println("Connecting to WhatsApp...")
	err = client.Connect()
	if err != nil {
		fmt.Println("Connection failed.")
		panic(err)
	}
	fmt.Println("Connected.")

	// Aggressive memory management
	plugins.SetAggressiveGC()

	if client.Store.ID == nil {
		if args.PhoneNumber == "" {
			fmt.Println("No session found. Please provide a phone number using -p or --phone-number")
			return
		}

		fmt.Println("Waiting 10 seconds before generating pairing code...")
		time.Sleep(10 * time.Second)

		code, err := client.PairPhone(ctx, args.PhoneNumber, true, whatsmeow.PairClientChrome, "Chrome (Android)")
		if err != nil {
			panic(err)
		}
		fmt.Printf("Your pairing code is: %s\n", code)
	} else {
		ownerPhone := strings.SplitN(client.Store.ID.User, ".", 2)[0]
		plugins.InitLIDStore(container.LIDMap, ownerPhone)
		if err := plugins.InitSettings(ownerPhone); err != nil {
			panic(fmt.Errorf("settings load: %w", err))
		}
		plugins.BootstrapOwnerSudoers()
		fmt.Println("Already logged in.")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
	time.Sleep(800 * time.Millisecond)
}

func candidateSourceDirs() []string {
	candidates := []string{"/opt/mack-bot/src"}
	if pd := os.Getenv("ProgramData"); pd != "" {
		candidates = append([]string{filepath.Join(pd, "Mack-Bot", "src")}, candidates...)
	}
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		candidates = append(candidates, filepath.Join(pf, "Mack-Bot", "src"))
	}
	return candidates
}

func resolveSourceDir() string {
	if sourceDir != "" {
		return sourceDir
	}
	for _, dir := range candidateSourceDirs() {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
	}
	return ""
}

func runUpdate() {
	src := resolveSourceDir()
	if src == "" {
		fmt.Fprintln(os.Stderr, "error: could not locate the mack-bot source directory.")
		fmt.Fprintln(os.Stderr, "Please reinstall using the install script.")
		os.Exit(1)
	}
	sourceDir = src

	fmt.Println("Fetching latest changes...")
	fetch := exec.Command("git", "-C", sourceDir, "fetch", "origin", "--quiet")
	fetch.Stderr = os.Stderr
	if err := fetch.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\ngit fetch failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Fetch complete")

	countOut, _ := exec.Command("git", "-C", sourceDir, "rev-list", "HEAD..FETCH_HEAD", "--count").Output()
	if strings.TrimSpace(string(countOut)) == "0" {
		fmt.Println("Already up to date.")
		return
	}

	fmt.Println("Pulling changes...")
	pull := exec.Command("git", "-C", sourceDir, "pull", "--ff-only")
	pull.Stdout = os.Stdout
	pull.Stderr = os.Stderr
	if err := pull.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\ngit pull failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Changes pulled")

	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\ncould not determine executable path: %v\n", err)
		os.Exit(1)
	}
	exePath, _ = filepath.EvalSymlinks(exePath)
	tmpPath := exePath + ".new"
	verOut, _ := exec.Command("git", "-C", sourceDir, "describe", "--tags", "--always", "--dirty").Output()
	gitVer := strings.TrimSpace(string(verOut))
	if gitVer == "" {
		gitVer = Version
	}
	commitOut, _ := exec.Command("git", "-C", sourceDir, "rev-parse", "--short", "HEAD").Output()
	gitCommit := strings.TrimSpace(string(commitOut))
	if gitCommit == "" {
		gitCommit = "unknown"
	}
	buildDate := time.Now().UTC().Format(time.RFC3339)
	ldflags := fmt.Sprintf("-s -w -X main.Version=%s -X main.Commit=%s -X main.BuildDate=%s -X main.sourceDir=%s",
		gitVer, gitCommit, buildDate, sourceDir)

	fmt.Println("Building new binary...")
	buildDone := make(chan error, 1)
	go func() {
		cmd := exec.Command("go", "build",
			"-ldflags", ldflags,
			"-trimpath",
			"-o", tmpPath,
			"./",
		)
		cmd.Dir = sourceDir
		buildDone <- cmd.Run()
	}()

	var buildErr error
	buildErr = <-buildDone

	if buildErr != nil {
		_ = os.Remove(tmpPath)
		fmt.Fprintf(os.Stderr, "\nbuild failed: %v\n", buildErr)
		os.Exit(1)
	}
	fmt.Println("Build complete")

	if err := os.Rename(tmpPath, exePath); err != nil {
		fmt.Fprintf(os.Stderr, "\ncould not replace binary (stop the bot first if it is running): %v\n", err)
		fmt.Fprintf(os.Stderr, "New binary saved at: %s\nRename manually: mv %s %s\n", tmpPath, tmpPath, exePath)
		os.Exit(1)
	}
	fmt.Println("Mack-Bot updated successfully.")
}

func runListSessions(ctx context.Context, dialect, dbAddr string) {
	dbLog := waLog.Stdout("Database", "ERROR", true)
	container, err := sqlstore.New(ctx, dialect, dbAddr, dbLog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
		os.Exit(1)
	}

	devices, err := container.GetAllDevices(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list sessions: %v\n", err)
		os.Exit(1)
	}

	if len(devices) == 0 {
		fmt.Println("No sessions found.")
		return
	}

	fmt.Printf("%-4s  %-20s  %s\n", "No.", "Phone", "JID")
	fmt.Println(strings.Repeat("-", 60))
	for i, dev := range devices {
		phone := "(unknown)"
		jid := "(unpaired)"
		if dev.ID != nil {
			phone = strings.SplitN(dev.ID.User, ".", 2)[0]
			jid = dev.ID.String()
		}
		fmt.Printf("%-4d  %-20s  %s\n", i+1, phone, jid)
	}
}

func runDeleteSession(ctx context.Context, dialect, dbAddr, phone string, reset bool) {
	dbLog := waLog.Stdout("Database", "ERROR", true)
	container, err := sqlstore.New(ctx, dialect, dbAddr, dbLog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
		os.Exit(1)
	}

	devices, err := container.GetAllDevices(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to query sessions: %v\n", err)
		os.Exit(1)
	}

	for _, dev := range devices {
		if dev.ID == nil {
			continue
		}
		if strings.SplitN(dev.ID.User, ".", 2)[0] == phone {
			if err := container.DeleteDevice(ctx, dev); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete session: %v\n", err)
				os.Exit(1)
			}
			if reset {
				fmt.Printf("Session for %s has been reset.\nRun with --phone-number %s to re-pair.\n", phone, phone)
			} else {
				fmt.Printf("Session for %s has been permanently deleted.\n", phone)
			}
			return
		}
	}

	fmt.Fprintf(os.Stderr, "No session found for phone number: %s\n", phone)
	os.Exit(1)
}
