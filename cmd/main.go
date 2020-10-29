package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/eko/monday/internal/runtime"
	"github.com/eko/monday/pkg/build"
	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/forward"
	"github.com/eko/monday/pkg/hostfile"
	"github.com/eko/monday/pkg/proxy"
	"github.com/eko/monday/pkg/run"
	"github.com/eko/monday/pkg/setup"
	"github.com/eko/monday/pkg/ui"
	"github.com/eko/monday/pkg/watch"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/jroimartin/gocui"
)

const (
	name = "Monday"
)

var (
	Version string

	proxyfier proxy.Proxy
	forwarder forward.Forwarder
	setuper   setup.Setuper
	builder   build.Builder
	runner    run.Runner
	watcher   watch.Watcher

	uiEnabled = len(os.Getenv("MONDAY_ENABLE_UI")) > 0
)

func main() {
	runtime.InitRuntimeEnvironment()

	rootCmd := &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {
			if !uiEnabled {
				uiEnabled, _ = strconv.ParseBool(cmd.Flag("ui").Value.String())
			}

			conf, err := config.Load()
			if err != nil {
				fmt.Printf("❌  %v\n", err)
				return
			}

			runProject(conf, selectProject(conf))

			handleExitSignal()
		},
	}

	// UI-enable flag (for both root and run commands)
	runCmd.Flags().Bool("ui", false, "Enable the terminal UI")
	rootCmd.Flags().Bool("ui", false, "Enable the terminal UI")

	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("❌  An error has occured during 'edit' command: %v\n", err)
		os.Exit(1)
	}
}

func selectProject(conf *config.Config) string {
	projects := conf.GetProjectNames()

	prompt := promptui.Select{
		Label: "Which project do you want to work on?",
		Items: projects,
		Size:  20,
		Searcher: func(input string, index int) bool {
			return strings.Contains(
				strings.Replace(strings.ToLower(projects[index]), " ", "", -1),
				strings.Replace(strings.ToLower(input), " ", "", -1),
			)
		},
	}

	_, choice, err := prompt.Run()
	if err != nil {
		if err.Error() == "^C" {
			fmt.Println("\n👋  Bye")
			os.Exit(0)
		} else {
			panic(fmt.Sprintf("selection error:\n%v", err))
		}
	}

	fmt.Print("\n")

	return choice
}

func runProject(conf *config.Config, choice string) {
	layout := ui.NewLayout(uiEnabled)
	layout.Init()

	// Retrieve selected project configuration by its name
	project, err := conf.GetProjectByName(choice)
	if err != nil {
		panic(err)
	}

	// Initializes hosts file manager
	hostfile, err := hostfile.NewClient()
	if err != nil {
		panic(err)
	}

	proxyfier = proxy.NewProxy(layout.GetProxyView(), hostfile)
	setuper = setup.NewSetuper(layout.GetLogsView(), project, conf.Setup)
	builder = build.NewBuilder(layout.GetLogsView(), project, conf.Build)
	runner = run.NewRunner(layout.GetLogsView(), proxyfier, project, conf.Run)
	forwarder = forward.NewForwarder(layout.GetForwardsView(), proxyfier, project)

	watcher = watch.NewWatcher(setuper, builder, runner, forwarder, conf.Watch, project)
	go watcher.Watch()

	if uiEnabled {
		defer layout.GetGui().Close()

		if err := layout.GetGui().SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
			panic(err)
		}

		layout.GetStatusView().Writef(" ⇢  %s | Commands: ←/→: select view | ↑/↓: scroll up/down | a: toggle autoscroll | f: toggle fullscreen", choice)

		if err := layout.GetGui().MainLoop(); err != nil && err != gocui.ErrQuit {
			fmt.Println(err)
			stopAll()
		}
	}
}

// Handle for an exit signal in order to quit application on a proper way (shutting down connections and servers).
func handleExitSignal() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, os.Kill)

	<-stop

	stopAll()
}

func stopAll() {
	fmt.Println("\n👋  Bye, closing your local applications and remote connections now")

	watcher.Stop()
	forwarder.Stop()
	proxyfier.Stop()
	runner.Stop()

	os.Exit(0)
}

func quit(g *gocui.Gui, v *gocui.View) error {
	g.Close()
	stopAll()
	return gocui.ErrQuit
}
