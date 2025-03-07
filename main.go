package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
)

var flagAppname = flag.String("appname", "maskedemail-cli", "the appname to identify the creator")
var flagToken = flag.String("token", "", "the token to authenticate with")
var flagAccountID = flag.String("accountid", "", "fastmail account id")
var action actionType = actionTypeUnknown

type actionType string

const (
	actionTypeUnknown = ""
	actionTypeCreate  = "create"
	actionTypeSession = "session"
	actionTypeDisable = "disable"
	actionTypeEnable  = "enable"
	actionTypeList    = "list"
)

func init() {
	flag.Parse()
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println("")
		fmt.Println("Commands:")
		fmt.Println("  maskedemail-cli create <domain>")
		fmt.Println("  maskedemail-cli enable <maskedemail>")
		fmt.Println("  maskedemail-cli disable <maskedemail>")
		fmt.Println("  maskedemail-cli session")
		fmt.Println("  maskedemail-cli list")
	}

	if len(flag.Args()) < 1 {
		log.Println("no argument given. currently supported: create, session, disable, enable")
		flag.Usage()
		os.Exit(1)
	}

	if *flagToken == "" {
		log.Println("-token flag is not set")
		flag.Usage()
		os.Exit(1)
	}

	switch strings.ToLower(flag.Arg(0)) {
	case
		"create":
		action = actionTypeCreate

	case "session":
		action = actionTypeSession

	case "disable":
		action = actionTypeDisable

	case "enable":
		action = actionTypeEnable

	case "list":
		action = actionTypeList
	}
}

func main() {
	client := NewClient(*flagToken, *flagAppname, "35c941ae")

	switch action {
	case actionTypeSession:
		session, err := client.Session()
		if err != nil {
			log.Fatalf("fetching session: %v", err)
		}
		var accIDs []string
		for accID := range session.Accounts {
			if *flagAccountID != "" && *flagAccountID != accID {
				continue
			}
			accIDs = append(accIDs, accID)
		}

		primaryAccountID := session.PrimaryAccounts[maskedEmailCapabilityURI]
		sort.Slice(
			accIDs,
			func(i, j int) bool {
				if primaryAccountID == accIDs[i] {
					return true
				}
				return accIDs[i] < accIDs[j]
			},
		)
		for _, accID := range accIDs {
			isPrimary := primaryAccountID == accID
			isEnabled := session.AccountHasCapability(accID, maskedEmailCapabilityURI)

			fmt.Printf(
				"%s [%s] (primary: %t, enabled: %t)\n",
				session.Accounts[accID].Name,
				accID,
				isPrimary,
				isEnabled,
			)
		}

	case actionTypeCreate:
		if flag.Arg(1) == "" {
			log.Fatalln("Usage: create <domain>")
		}

		session, err := client.Session()
		if err != nil {
			log.Fatalf("initializing session: %v", err)
		}

		createRes, err := client.CreateMaskedEmail(session, *flagAccountID, flag.Arg(1), true)
		if err != nil {
			log.Fatalf("err while creating maskedemail: %v", err)
		}

		fmt.Println(createRes.Email)

	case actionTypeDisable:
		if flag.Arg(1) == "" {
			log.Fatalln("Usage: disable <maskedemail>")
		}

		session, err := client.Session()
		if err != nil {
			log.Fatalf("initializing session: %v", err)
		}

		allAliases, err := client.GetAllMaskedEmails(session, *flagAccountID)
		if err != nil {
			log.Fatalf("err while getting maskedemails: %v", err)
		}

		// find the alias to disable
		var alias *MaskedEmail
		for _, a := range allAliases {
			if a.Email == flag.Arg(1) {
				alias = a
				break
			}
		}

		if alias == nil {
			log.Fatalf("maskedemail %s not found", flag.Arg(1))
		}

		_, err = client.UpdateMaskedEmailState(session, *flagAccountID, alias.ID, MaskedEmailStateDisabled)
		if err != nil {
			log.Fatalf("err while updating maskedemail: %v", err)
		}

		fmt.Printf("disabled email: %s\n", flag.Arg(1))

	case actionTypeEnable:
		if flag.Arg(1) == "" {
			log.Fatalln("Usage: enable <email>")
		}

		session, err := client.Session()
		if err != nil {
			log.Fatalf("initializing session: %v", err)
		}

		allAliases, err := client.GetAllMaskedEmails(session, *flagAccountID)
		if err != nil {
			log.Fatalf("err while updating maskedemail: %v", err)
		}

		// find the alias to disable
		var alias *MaskedEmail
		for _, a := range allAliases {
			if a.Email == flag.Arg(1) {
				alias = a
				break
			}
		}

		if alias == nil {
			log.Fatalf("maskedemail %s not found", flag.Arg(1))
		}

		_, err = client.UpdateMaskedEmailState(session, *flagAccountID, alias.ID, MaskedEmailStateEnabled)
		if err != nil {
			log.Fatalf("err while creating maskedemail: %v", err)
		}

		fmt.Printf("enabled maskedemail: %s\n", flag.Arg(1))

	case actionTypeList:
		session, err := client.Session()
		if err != nil {
			log.Fatalf("initializing session: %v", err)
		}

		maskedEmails, err := client.GetAllMaskedEmails(session, *flagAccountID)
		if err != nil {
			log.Fatalf("err while creating maskedemail: %v", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
		fmt.Fprintln(w, "Masked Email\tFor Domain\tState\tLast Email At\t")
		for _, email := range maskedEmails {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", email.Email, email.ForDomain, email.State, email.LastMessageAt)
		}
		w.Flush()

	default:
		fmt.Println("action not found")
		flag.Usage()
		os.Exit(1)
	}
}
