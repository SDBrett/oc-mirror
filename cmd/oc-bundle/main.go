package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"k8s.io/klog"
	klogv2 "k8s.io/klog/v2"
)

var (
	rootOpts struct {
		dir        string
		logLevel   string
		dryRun     bool
		skipTLS    bool
		configPath string
	}
)

func main() {
	// This attempts to configure klog (used by vendored Kubernetes code) not
	// to log anything.
	// Handle k8s.io/klog
	var fs flag.FlagSet
	klog.InitFlags(&fs)
	fs.Set("stderrthreshold", "4")
	klog.SetOutput(ioutil.Discard)
	// Handle k8s.io/klog/v2
	var fsv2 flag.FlagSet
	klogv2.InitFlags(&fsv2)
	fsv2.Set("stderrthreshold", "4")
	klogv2.SetOutput(ioutil.Discard)

	bundleMain()
}

func bundleMain() {
	rootCmd := newRootCmd()

	for _, subCmd := range []*cobra.Command{
		newCreateCmd(),
		newPublishCmd(),
	} {
		rootCmd.AddCommand(subCmd)
	}

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalf("Error executing oc bundle: %v", err)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:              filepath.Base(os.Args[0]),
		Short:            "Manages container images for internet-disconnected systems",
		Long:             "",
		PersistentPreRun: runRootCmd,
		SilenceErrors:    true,
		SilenceUsage:     true,
	}
	cmd.PersistentFlags().StringVarP(&rootOpts.dir, "dir", "d", ".", "assets directory")
	cmd.PersistentFlags().StringVar(&rootOpts.logLevel, "log-level", "info", "log level (e.g. \"debug | info | warn | error\")")
	cmd.PersistentFlags().BoolVar(&rootOpts.dryRun, "dry-run", false, "print actions without mirroring images "+
		"(experimental: only works for operator catalogs)")
	cmd.PersistentFlags().BoolVar(&rootOpts.skipTLS, "skip-tls", false, "skip client-side TLS validation")
	cmd.PersistentFlags().StringVarP(&rootOpts.configPath, "config", "c", "imageset-config.yaml", "Path to imageset configuration file")

	return cmd
}

func runRootCmd(cmd *cobra.Command, args []string) {
	logrus.SetOutput(ioutil.Discard)
	logrus.SetLevel(logrus.TraceLevel)

	level, err := logrus.ParseLevel(rootOpts.logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}

	logrus.AddHook(newFileHookWithNewlineTruncate(os.Stderr, level, &logrus.TextFormatter{
		// Setting ForceColors is necessary because logrus.TextFormatter determines
		// whether or not to enable colors by looking at the output of the logger.
		// In this case, the output is ioutil.Discard, which is not a terminal.
		// Overriding it here allows the same check to be done, but against the
		// hook's output instead of the logger's output.
		ForceColors:            terminal.IsTerminal(int(os.Stderr.Fd())),
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
		DisableQuote:           true,
	}))

	if err != nil {
		logrus.Fatal(errors.Wrap(err, "invalid log-level"))
	}
}
