package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func doKinit(keytabPath, kerberosPrinciple string) error {
	// Try Kinit for the current user
	tryKinit := exec.Command(defaultShell, "-c", "kinit", "-kt", keytabPath, kerberosPrinciple)
	var outb, errb bytes.Buffer
	tryKinit.Stdout = &outb
	tryKinit.Stderr = &errb
	if err := tryKinit.Run(); err != nil {
		return fmt.Errorf(errb.String() + err.Error())
	}
	return nil
}

func isKerberosCacheValid(timestampLayout string) (bool, error) {
	//
	var outb, errb bytes.Buffer
	currentDate := time.Now()
	tryKlist := exec.Command(defaultShell, "-c", "klist | awk '{print $3}' | grep '^[0-9]' | head -1")
	tryKlist.Stdout = &outb
	tryKlist.Stderr = &errb
	if err := tryKlist.Run(); err != nil {
		return false, fmt.Errorf(errb.String() + err.Error())
	}
	expiryDateString := strings.TrimSpace(outb.String())
	if expiryDateString == "" {
		// NOTE:
		// Ideally we must've returned err
		// But if the klist is empty, the date will be empty
		// This might also mean that kinit was not done yet for the current user
		// So we can actually do a kinit for the first time to start with
		return false, nil
	}

	// TODO: This is not a reliable logic. Needs improvement.
	expiryDate, err := time.Parse(timestampLayout, expiryDateString)
	if err != nil {
		for _, ts := range availableTimestampLayouts {
			if expiryDate, err = time.Parse(ts, expiryDateString); err == nil {
				break
			}
		}
	}

	if expiryDate.IsZero() {
		fmt.Printf("ERROR: expiry date'%s' matches NONE of supported timestamp layout\n", expiryDate)
		return false, err
	}

	if expiryDate.Equal(currentDate) || expiryDate.Before(currentDate) {
		return false, nil
	}
	return true, nil
}
