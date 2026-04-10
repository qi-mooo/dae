/*
*  SPDX-License-Identifier: AGPL-3.0-only
*  Copyright (c) 2022-2025, daeuniverse Organization <dae@v2raya.org>
 */

package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/daeuniverse/dae/config"
	"github.com/spf13/cobra"
)

const (
	AbortFile = "/var/run/dae.abort"
)

var (
	Version      = "unknown"
	ProgramName  = "dae"
	ProgramShort = "dae is a high-performance transparent proxy solution."
	ProgramLong  = `dae is a high-performance transparent proxy solution.`
	rootCmd      = &cobra.Command{
		Use:     "dae [flags] [command [argument ...]]",
		Short:   "dae is a high-performance transparent proxy solution.",
		Long:    `dae is a high-performance transparent proxy solution.`,
		Version: Version,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}
)

func init() {
	refreshRootCommandMetadata()
}

func refreshRootCommandMetadata() {
	config.Version = Version
	rootCmd.Use = fmt.Sprintf("%s [flags] [command [argument ...]]", ProgramName)
	rootCmd.Short = ProgramShort
	rootCmd.Long = ProgramLong
	rootCmd.Version = strings.Join([]string{
		Version,
		fmt.Sprintf("go runtime %v %v/%v", runtime.Version(), runtime.GOOS, runtime.GOARCH),
		"Copyright (c) 2022-2025 @daeuniverse",
		"License GNU AGPLv3 <https://github.com/daeuniverse/dae/blob/main/LICENSE>",
	}, "\n")
}

// Execute executes the root command.
func Execute() error {
	refreshRootCommandMetadata()
	return rootCmd.Execute()
}
