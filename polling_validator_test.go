package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
)

type PollingValidatorTest struct {
	auth_email_file  *os.File
	polling_interval time.Duration
}

func NewPollingValidatorTest(t *testing.T) *PollingValidatorTest {
	vt := &PollingValidatorTest{}
	var err error
	vt.auth_email_file, err = ioutil.TempFile("", "test_auth_emails_")
	if err != nil {
		t.Fatal("failed to create temp file: " + err.Error())
	}
	return vt
}

func (vt *PollingValidatorTest) TearDown() {
	os.Remove(vt.auth_email_file.Name())
}

func (vt *PollingValidatorTest) NewValidator(domains []string,
	pollingInterval time.Duration) func(string) bool {
	return PollingValidator(domains, vt.auth_email_file.Name(),
		pollingInterval)
}

// This will close vt.auth_email_file.
func (vt *PollingValidatorTest) WriteEmails(t *testing.T, emails []string) {
	defer vt.auth_email_file.Close()
	vt.auth_email_file.WriteString(strings.Join(emails, "\n"))
	if err := vt.auth_email_file.Close(); err != nil {
		t.Fatal("failed to close temp file " +
			vt.auth_email_file.Name() + ": " + err.Error())
	}
}

func TestValidatorEmpty(t *testing.T) {
	vt := NewPollingValidatorTest(t)
	defer vt.TearDown()

	vt.WriteEmails(t, []string(nil))
	domains := []string(nil)
	validator := vt.NewValidator(domains, 1*time.Millisecond)

	if validator("foo.bar@example.com") {
		t.Error("nothing should validate when the email and " +
			"domain lists are empty")
	}
}

func TestValidatorSingleEmail(t *testing.T) {
	vt := NewPollingValidatorTest(t)
	defer vt.TearDown()

	vt.WriteEmails(t, []string{"foo.bar@example.com"})
	domains := []string(nil)
	validator := vt.NewValidator(domains, 1*time.Millisecond)

	if !validator("foo.bar@example.com") {
		t.Error("email should validate")
	}
	if validator("baz.quux@example.com") {
		t.Error("email from same domain but not in list " +
			"should not validate when domain list is empty")
	}
}

func TestValidatorSingleDomain(t *testing.T) {
	vt := NewPollingValidatorTest(t)
	defer vt.TearDown()

	vt.WriteEmails(t, []string(nil))
	domains := []string{"example.com"}
	validator := vt.NewValidator(domains, 1*time.Millisecond)

	if !validator("foo.bar@example.com") {
		t.Error("email should validate")
	}
	if !validator("baz.quux@example.com") {
		t.Error("email from same domain should validate")
	}
}

func TestValidatorMultipleEmailsMultipleDomains(t *testing.T) {
	vt := NewPollingValidatorTest(t)
	defer vt.TearDown()

	vt.WriteEmails(t, []string{
		"xyzzy@example.com",
		"plugh@example.com",
	})
	domains := []string{"example0.com", "example1.com"}
	validator := vt.NewValidator(domains, 1*time.Millisecond)

	if !validator("foo.bar@example0.com") {
		t.Error("email from first domain should validate")
	}
	if !validator("baz.quux@example1.com") {
		t.Error("email from second domain should validate")
	}
	if !validator("xyzzy@example.com") {
		t.Error("first email in list should validate")
	}
	if !validator("plugh@example.com") {
		t.Error("second email in list should validate")
	}
	if validator("xyzzy.plugh@example.com") {
		t.Error("email not in list that matches no domains " +
			"should not validate")
	}
}

func TestValidatorComparisonsAreCaseInsensitive(t *testing.T) {
	vt := NewPollingValidatorTest(t)
	defer vt.TearDown()

	vt.WriteEmails(t, []string{"Foo.Bar@Example.Com"})
	domains := []string{"Frobozz.Com"}
	validator := vt.NewValidator(domains, 1*time.Millisecond)

	if !validator("foo.bar@example.com") {
		t.Error("loaded email addresses are not lower-cased")
	}
	if !validator("Foo.Bar@Example.Com") {
		t.Error("validated email addresses are not lower-cased")
	}
	if !validator("foo.bar@frobozz.com") {
		t.Error("loaded domains are not lower-cased")
	}
	if !validator("foo.bar@Frobozz.Com") {
		t.Error("validated domains are not lower-cased")
	}
}

func TestValidatorIgnoreSpacesInAuthEmails(t *testing.T) {
	vt := NewPollingValidatorTest(t)
	defer vt.TearDown()

	vt.WriteEmails(t, []string{"   foo.bar@example.com   "})
	domains := []string(nil)
	validator := vt.NewValidator(domains, 1*time.Millisecond)

	if !validator("foo.bar@example.com") {
		t.Error("email should validate")
	}
}
