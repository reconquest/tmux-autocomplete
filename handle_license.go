package main

import (
	"fmt"
	"os"

	"github.com/reconquest/karma-go"
)

func handleLicenseInfo() {
	context := karma.Describe("path", getLicensePath())
	if !isLicenseExists() {
		fmt.Println(
			context.Format(nil, "no license file found"),
		)
		os.Exit(1)
	}

	license, err := getLicense()
	if err != nil {
		fmt.Println(
			context.Format(err, "unable to obtain license"),
		)
		os.Exit(2)
	}

	if license == nil {
		fmt.Println(
			context.Format(nil, "invalid license"),
		)
		os.Exit(3)
	}

	fmt.Println(string(license.Data))
	os.Exit(0)
}
