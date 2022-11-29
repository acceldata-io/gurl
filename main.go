package main

import (
	"fmt"
	netURL "net/url"
	"os"
	"strings"

	"github.com/integrii/flaggy"
)

// Configurations
// NOTE: All these below configutaions are global scoped
// DO NOT MUTATE them any where in the program
var (
	Version                   = "0.0.0"
	BuildID                   = "0"
	url                       = ""
	reqType                   = "GET"
	isKerberized              = false
	keytabPath                = "/etc/security/hdfs-headless.keytab"
	kerberosPrinciple         = "hdfs@ACME.ORG"
	timestampLayout           = "02/01/2006"
	isBasicAuth               = ""
	basicAuthUser             = ""
	basicAuthPassword         = ""
	outputFile                = ""
	clientUserAgent           = "gurl/0.0.1"
	enforceTLSVerify          = false
	reqHTTPMethod             httpMethod
	availableTimestampLayouts = []string{"01/02/2006", "01/02/06", "02/01/2006", "02/01/06", "2006/01/02", "06/01/02", "2006/02/01", "06/02/01"}
	defaultShell              = "/usr/bin/sh"
	defaultKRBConfig          = "/etc/krb5.conf"
	secondaryKRBConfig        = "/etc/krb5/krb5.conf"
)

func init() {
	flaggy.SetName("gURL")
	flaggy.SetDescription("gURL - A replacement for statically compiled cURL binary")

	flaggy.DefaultParser.ShowHelpOnUnexpected = true
	flaggy.DefaultParser.AdditionalHelpAppend = `
Usage Format: "gurl -X HTTP_REQ_TYPE -ua user-agent -u "basic-auth-username:basic-auth-password" -k isKerberosEnabled -kt kerberos-keytab-path -kp kerberos-principle -ts timestamp-layout -ev enforce-TLS-verification -l url"
Example: "gurl -X GET -ua "gurl/0.0.1" -u "hdfs:" -k -kt /etc/security/hdfs-headless.keytab -kp hdfs@ACME.ORG -ts '01/02/2006' -ev -l https://node01.acme.org:9871/"`

	flaggy.DefaultParser.AdditionalHelpPrepend = "https://acceldata.io/"

	// set the version and parse all inputs into variables
	version := Version + "\n" + "Build ID: " + BuildID
	flaggy.SetVersion(version)

	//
	flaggy.String(&url, "l", "url", "URL to make request")

	flaggy.String(&reqType, "X", "type", "HTTP request type to use")

	flaggy.Bool(&isKerberized, "k", "kerberized", "Is Kerberos enabled for the URL")
	flaggy.String(&keytabPath, "kt", "keytab-path", "Kerberos Keytab Path")
	flaggy.String(&kerberosPrinciple, "kp", "kerberos-principle", "Kerberos principle to use with keytab")

	flaggy.String(&timestampLayout, "ts", "ts-format", "Timestamp format klist uses in 'Go Time Format'. Example: 'mm/dd/yyyy' => '01/02/2006'")

	flaggy.String(&isBasicAuth, "u", "basic-auth", "Is Basic Auth Enabled for the URL")

	flaggy.Bool(&enforceTLSVerify, "ev", "enforce-tls-verify", "Enforce TLS certification verification")

	flaggy.String(&clientUserAgent, "ua", "user-agent", "User Agent to be set for the client requests")
	flaggy.String(&outputFile, "o", "output-file", "Write the request response to a file")

	flaggy.Parse()

	// Trim Extra Space from all user inputs
	url = strings.TrimSpace(url)
	reqType = strings.TrimSpace(reqType)
	clientUserAgent = strings.TrimSpace(clientUserAgent)
	isBasicAuth = strings.TrimSpace(isBasicAuth)

	// Args validation & manipulation
	if url == "" {
		flaggy.ShowHelpAndExit("ERROR: 'url' parameter is required")
	} else {
		_, err := netURL.Parse(url)
		if err != nil {
			fmt.Println("ERROR: ", err.Error())
			flaggy.ShowHelpAndExit("ERROR: 'url' parameter has a invalid url")

		}
	}

	if isKerberized {
		//
		keytabPath = strings.TrimSpace(keytabPath)
		kerberosPrinciple = strings.TrimSpace(kerberosPrinciple)
		timestampLayout = strings.TrimSpace(timestampLayout)

		if keytabPath == "" {
			flaggy.ShowHelpAndExit("ERROR: 'keytab-path' parameter is required")
		}

		if _, err := os.Stat(keytabPath); err != nil {
			flaggy.ShowHelpAndExit("ERROR: cannot find or access the keytab file '" + keytabPath + "' because " + err.Error())
		}

		if kerberosPrinciple == "" {
			flaggy.ShowHelpAndExit("ERROR: 'kerberos-principle' parameter is required")
		}

		if timestampLayout != "" {
			if !isInSlice(timestampLayout, availableTimestampLayouts) {
				fmt.Printf("WARN: '%s' is not in the default 'ts-format' values\n", timestampLayout)
			} else {
				availableTimestampLayouts = removeFromSlice(timestampLayout, availableTimestampLayouts)
			}
		}

		// Check the dependecies
		if err := isKRBDepsAvail(); err != nil {
			flaggy.ShowHelpAndExit("ERROR: " + err.Error())
		}

	}

	// Basic Auth
	if isBasicAuth != "" {
		//
		basicAuthCreds := strings.Split(isBasicAuth, ":")
		basicAuthUser = strings.TrimSpace(basicAuthCreds[0])
		if len(basicAuthCreds) >= 2 {
			basicAuthPassword = strings.TrimSpace(strings.Join(basicAuthCreds[1:], ""))
		}
	}
}

func main() {
	// ------- NOTE ---------------
	// If kerberos is enabled
	// We expect 'kinit', 'klist', 'awk', 'grep' & 'head' binaries in the OS path
	// We expect '/etc/krb5.conf' file to be present
	// If the 'krb5.conf' path is different set it @ env 'KRB5_CONFIG'
	//
	// The temporary token will be generated @ '/tmp/krb5cc_<CURRENT-LINUX-UID>'
	// Incase of different location set it @ env 'KRB5CCNAME'
	// ------- NOTE ---------------

	// Check if kerberos is enabled
	if isKerberized {
		isKerberosCacheValid, err := isKerberosCacheValid(timestampLayout)
		if err != nil {
			fmt.Println("ERROR: Unable to validate Kerberos cache. Because: ", err.Error())
			os.Exit(1)
		}

		if !isKerberosCacheValid {
			if err := doKinit(keytabPath, kerberosPrinciple); err != nil {
				fmt.Println("ERROR: Unable to do Kinit. Because: ", err.Error())
				os.Exit(1)
			}
		}
	}

	if reqHTTPMethod, err := stringToMethod(reqType); err == nil {
		_, n, err := makeRequest(reqHTTPMethod, url)
		if err != nil {
			fmt.Println("STATUS: ", n)
			fmt.Println("ERROR: ", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}
}
