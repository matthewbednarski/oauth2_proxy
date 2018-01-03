package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

type UserDomainMap struct {
	m       unsafe.Pointer
	domains []string
}

func NewUserDomainMap(domains []string) *UserDomainMap {
	um := &UserDomainMap{domains: domains}
	m := make(map[string]bool)
	atomic.StorePointer(&um.m, unsafe.Pointer(&m))
	return um
}

func (um *UserDomainMap) IsValid(email string) (result bool) {
	m := *(*map[string]bool)(atomic.LoadPointer(&um.m))
	_, result = m[email]
	return
}

func parseAuthenticatedEmailsFile(usersFile string) (updated map[string]bool) {
	//TODO: look for changes first
	r, err := os.Open(usersFile)
	if err != nil {
		log.Fatalf("failed opening authenticated-emails-file=%q, %s", usersFile, err)
	}
	defer r.Close()

	csv_reader := csv.NewReader(r)
	csv_reader.Comma = ','
	csv_reader.Comment = '#'
	csv_reader.TrimLeadingSpace = true
	records, err := csv_reader.ReadAll()
	if err != nil {
		log.Printf("error reading authenticated-emails-file=%q, %s", usersFile, err)
		return
	}
	for _, r := range records {
		address := strings.ToLower(strings.TrimSpace(r[0]))
		updated[address] = true
	}
	return
}

func pollForUpdates(pollingInterval time.Duration, action func()) {
	ticker := time.NewTicker(pollingInterval)
	go func() {
		for range ticker.C {
			action()
		}
	}()
}
func (um *UserDomainMap) pollingValidatorImpl() func(string) bool {
	var allowAll bool
	for i, domain := range um.domains {
		if domain == "*" {
			allowAll = true
			continue
		}
		um.domains[i] = fmt.Sprintf("@%s", strings.ToLower(domain))
	}

	validator := func(email string) (valid bool) {
		if email == "" {
			return
		}
		email = strings.ToLower(email)
		for _, domain := range um.domains {
			valid = valid || strings.HasSuffix(email, domain)
		}
		if !valid {
			valid = um.IsValid(email)
		}
		if allowAll {
			valid = true
		}
		return valid
	}
	return validator
}
func PollingValidator(domains []string, usersFile string, pollingInterval time.Duration) func(string) bool {
	um := NewUserDomainMap(domains)
	if usersFile != "" {
		usersFile = filepath.Clean(usersFile)
		log.Printf("using authenticated emails file %s", usersFile)
		setupUsersMap := func() {
			log.Printf("polling %s for updates", usersFile)
			m := parseAuthenticatedEmailsFile(usersFile)
			atomic.StorePointer(&um.m, unsafe.Pointer(&m))
		}
		//DONE: add polling
		pollForUpdates(pollingInterval, setupUsersMap)
	}
	return um.pollingValidatorImpl()
}
