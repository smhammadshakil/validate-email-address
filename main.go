package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"regexp"
	"strings"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("Email validation tool\n")
	fmt.Printf("Input an email address:\n")
	for scanner.Scan() {
		email := scanner.Text()
		err := validateEmail(email)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Println("Email validate successfully!")
		}
		fmt.Print("\nInput an email address: ")
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error: could not read from input: %v\n\n", err)
	}
}

// Validate an email address
func validateEmail(email string) error {
	// Validate email format and extract domain
	domain, err := extractDomainFromEmail(email)
	if err != nil {
		return fmt.Errorf("invalid email: %v", err)
	}

	// Check domain DNS records (MX, SPF, DMARC)
	err = checkDomain(domain)
	if err != nil {
		return fmt.Errorf("domain validation failed: %v", err)
	}

	// Perform SMTP validation to validate the email address
	err = validateSMTP(email, domain)
	if err != nil {
		return fmt.Errorf("SMTP validation failed: %v", err)
	}

	return nil
}

// Extracts the domain from an email address and validates its format
func extractDomainFromEmail(email string) (string, error) {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})$`)
	matches := re.FindStringSubmatch(email)
	if len(matches) < 2 {
		return "", errors.New("invalid email format")
	}
	return matches[1], nil
}

// Check the domain's MX, SPF, and DMARC records
func checkDomain(domain string) error {
	var hasMX, hasSPF, hasDMARC bool
	var spfRecord, dmarcRecord string

	// Check MX records
	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		return fmt.Errorf("error looking up MX records: %v", err)
	}
	if len(mxRecords) > 0 {
		hasMX = true
	}

	// Check SPF records
	txtRecords, err := net.LookupTXT(domain)
	if err != nil {
		log.Printf("Error looking up TXT records for domain %s: %v\n", domain, err)
	}
	for _, record := range txtRecords {
		if strings.HasPrefix(record, "v=spf1") {
			hasSPF = true
			spfRecord = record
			break
		}
	}

	// Check DMARC records
	dmarcRecords, err := net.LookupTXT("_dmarc." + domain)
	if err != nil {
		log.Printf("Error looking up DMARC records for domain %s: %v\n", domain, err)
	}
	for _, record := range dmarcRecords {
		if strings.HasPrefix(record, "v=DMARC1") {
			hasDMARC = true
			dmarcRecord = record
			break
		}
	}

	// Print domain validation results
	fmt.Printf("\n=> Email Domain: %v\n", domain)
	fmt.Printf("=> hasMX: %v\n", hasMX)
	fmt.Printf("=> hasSPF: %v\n", hasSPF)
	fmt.Printf("=> spfRecord: %v\n", spfRecord)
	fmt.Printf("=> hasDMARC: %v\n", hasDMARC)
	fmt.Printf("=> dmarcRecord: %v\n", dmarcRecord)

	if !hasMX {
		return errors.New("no MX records found for the domain")
	}

	return nil
}

// Perform SMTP validation to validate if the email exists
func validateSMTP(email, domain string) error {
	// Get MX records for the domain
	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		return fmt.Errorf("error looking up MX records: %v", err)
	}

	// Use the first MX record for SMTP validation
	mx := mxRecords[0].Host

	// Connect to the mail server
	conn, err := net.Dial("tcp", mx+":25")
	if err != nil {
		return fmt.Errorf("error connecting to mail server: %v", err)
	}
	defer conn.Close()

	// Perform basic SMTP commands to validate the email
	client, err := smtp.NewClient(conn, mx)
	if err != nil {
		return fmt.Errorf("error creating SMTP client: %v", err)
	}
	defer client.Quit()

	// Say hello to the mail server
	err = client.Hello("localhost")
	if err != nil {
		return fmt.Errorf("error sending HELO command: %v", err)
	}

	// Set the sender (use a dummy sender address)
	err = client.Mail("test@example.com")
	if err != nil {
		return fmt.Errorf("error sending MAIL FROM command: %v", err)
	}

	// Set the recipient (the email address to validate)
	err = client.Rcpt(email)
	if err != nil {
		return fmt.Errorf("error sending RCPT TO command: %v", err)
	}

	// If all SMTP commands succeed, the email exists
	return nil
}
