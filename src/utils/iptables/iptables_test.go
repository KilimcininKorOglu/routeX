//go:build testing

package iptables

import (
	"reflect"
	"testing"
)

// TestChainPatch verifies that rules are appended to existing ones
func TestChainPatch(t *testing.T) {
	fake := NewFakeIPTables(ProtocolIPv4)

	// Initial state: chain already has rules
	fake.SetInitialRules("filter", "FORWARD", [][]string{
		{"-i", "eth0", "-j", "ACCEPT"},
		{"-i", "eth1", "-j", "DROP"},
	})

	ipt := NewIPTables(fake)

	// Register chain as overlay
	err := ipt.RegisterChainPatch("filter", "FORWARD")
	if err != nil {
		t.Fatalf("RegisterChainPatch failed: %v", err)
	}

	// Add a new rule
	err = ipt.Append("filter", "FORWARD", "-i", "eth2", "-j", "ACCEPT")
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Commit
	err = ipt.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify result: old rules should remain, new one appended at the end
	rules := fake.GetRules("filter", "FORWARD")
	expected := [][]string{
		{"-i", "eth0", "-j", "ACCEPT"},
		{"-i", "eth1", "-j", "DROP"},
		{"-i", "eth2", "-j", "ACCEPT"},
	}

	if !reflect.DeepEqual(rules, expected) {
		t.Errorf("Patch failed.\nExpected: %v\nGot: %v", expected, rules)
	}
}

// TestChainPatchDelete verifies rule deletion in overlay mode
func TestChainPatchDelete(t *testing.T) {
	fake := NewFakeIPTables(ProtocolIPv4)

	fake.SetInitialRules("filter", "INPUT", [][]string{
		{"-p", "tcp", "--dport", "22", "-j", "ACCEPT"},
		{"-p", "tcp", "--dport", "80", "-j", "ACCEPT"},
		{"-p", "tcp", "--dport", "443", "-j", "ACCEPT"},
	})

	ipt := NewIPTables(fake)

	err := ipt.RegisterChainPatch("filter", "INPUT")
	if err != nil {
		t.Fatalf("RegisterChainPatch failed: %v", err)
	}

	// Delete the rule for port 80
	err = ipt.Delete("filter", "INPUT", "-p", "tcp", "--dport", "80", "-j", "ACCEPT")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	err = ipt.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	rules := fake.GetRules("filter", "INPUT")
	expected := [][]string{
		{"-p", "tcp", "--dport", "22", "-j", "ACCEPT"},
		{"-p", "tcp", "--dport", "443", "-j", "ACCEPT"},
	}

	if !reflect.DeepEqual(rules, expected) {
		t.Errorf("Patch delete failed.\nExpected: %v\nGot: %v", expected, rules)
	}
}

// TestChainPatchInsert verifies rule insertion at the beginning
func TestChainPatchInsert(t *testing.T) {
	fake := NewFakeIPTables(ProtocolIPv4)

	fake.SetInitialRules("filter", "OUTPUT", [][]string{
		{"-d", "10.0.0.0/8", "-j", "ACCEPT"},
	})

	ipt := NewIPTables(fake)

	err := ipt.RegisterChainPatch("filter", "OUTPUT")
	if err != nil {
		t.Fatalf("RegisterChainPatch failed: %v", err)
	}

	// Insert a rule at the beginning (position 1)
	err = ipt.Insert("filter", "OUTPUT", 1, "-d", "192.168.0.0/16", "-j", "ACCEPT")
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	err = ipt.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	rules := fake.GetRules("filter", "OUTPUT")
	expected := [][]string{
		{"-d", "192.168.0.0/16", "-j", "ACCEPT"},
		{"-d", "10.0.0.0/8", "-j", "ACCEPT"},
	}

	if !reflect.DeepEqual(rules, expected) {
		t.Errorf("Patch insert failed.\nExpected: %v\nGot: %v", expected, rules)
	}
}

// TestChainPatchNoDuplicates verifies that duplicates are not added
func TestChainPatchNoDuplicates(t *testing.T) {
	fake := NewFakeIPTables(ProtocolIPv4)

	fake.SetInitialRules("filter", "FORWARD", [][]string{
		{"-i", "eth0", "-j", "ACCEPT"},
	})

	ipt := NewIPTables(fake)

	err := ipt.RegisterChainPatch("filter", "FORWARD")
	if err != nil {
		t.Fatalf("RegisterChainPatch failed: %v", err)
	}

	// Try to add an already existing rule
	err = ipt.Append("filter", "FORWARD", "-i", "eth0", "-j", "ACCEPT")
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	err = ipt.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	rules := fake.GetRules("filter", "FORWARD")
	expected := [][]string{
		{"-i", "eth0", "-j", "ACCEPT"},
	}

	if !reflect.DeepEqual(rules, expected) {
		t.Errorf("Patch should not duplicate rules.\nExpected: %v\nGot: %v", expected, rules)
	}
}

// TestChainOverride verifies full chain replacement
func TestChainOverride(t *testing.T) {
	fake := NewFakeIPTables(ProtocolIPv4)

	// Initial state: chain with rules
	fake.SetInitialRules("nat", "PREROUTING", [][]string{
		{"-p", "tcp", "--dport", "80", "-j", "REDIRECT", "--to-port", "8080"},
		{"-p", "tcp", "--dport", "443", "-j", "REDIRECT", "--to-port", "8443"},
	})

	ipt := NewIPTables(fake)

	// Register as override - full replacement
	err := ipt.RegisterChainOverride("nat", "PREROUTING")
	if err != nil {
		t.Fatalf("RegisterChainOverride failed: %v", err)
	}

	// Add new rules (they will replace the old ones)
	err = ipt.Append("nat", "PREROUTING", "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-port", "5353")
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	err = ipt.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify: old rules should be gone, only the new one remains
	rules := fake.GetRules("nat", "PREROUTING")
	expected := [][]string{
		{"-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-port", "5353"},
	}

	if !reflect.DeepEqual(rules, expected) {
		t.Errorf("Override failed.\nExpected: %v\nGot: %v", expected, rules)
	}
}

// TestChainOverrideMultipleRules verifies replacement with multiple rules
func TestChainOverrideMultipleRules(t *testing.T) {
	fake := NewFakeIPTables(ProtocolIPv4)

	fake.SetInitialRules("mangle", "PREROUTING", [][]string{
		{"-j", "OLD_CHAIN"},
	})

	ipt := NewIPTables(fake)

	err := ipt.RegisterChainOverride("mangle", "PREROUTING")
	if err != nil {
		t.Fatalf("RegisterChainOverride failed: %v", err)
	}

	// Add multiple rules
	err = ipt.Append("mangle", "PREROUTING", "-j", "MARK", "--set-mark", "1")
	if err != nil {
		t.Fatalf("Append 1 failed: %v", err)
	}

	err = ipt.Append("mangle", "PREROUTING", "-j", "MARK", "--set-mark", "2")
	if err != nil {
		t.Fatalf("Append 2 failed: %v", err)
	}

	err = ipt.Append("mangle", "PREROUTING", "-j", "CONNMARK", "--save-mark")
	if err != nil {
		t.Fatalf("Append 3 failed: %v", err)
	}

	err = ipt.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	rules := fake.GetRules("mangle", "PREROUTING")
	expected := [][]string{
		{"-j", "MARK", "--set-mark", "1"},
		{"-j", "MARK", "--set-mark", "2"},
		{"-j", "CONNMARK", "--save-mark"},
	}

	if !reflect.DeepEqual(rules, expected) {
		t.Errorf("Override with multiple rules failed.\nExpected: %v\nGot: %v", expected, rules)
	}
}

// TestChainOverrideNoChangeIfSame verifies that if rules have not changed, nothing happens
func TestChainOverrideNoChangeIfSame(t *testing.T) {
	fake := NewFakeIPTables(ProtocolIPv4)

	fake.SetInitialRules("filter", "TEST", [][]string{
		{"-j", "ACCEPT"},
	})

	ipt := NewIPTables(fake)

	err := ipt.RegisterChainOverride("filter", "TEST")
	if err != nil {
		t.Fatalf("RegisterChainOverride failed: %v", err)
	}

	// Add the same rule
	err = ipt.Append("filter", "TEST", "-j", "ACCEPT")
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	err = ipt.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	rules := fake.GetRules("filter", "TEST")
	expected := [][]string{
		{"-j", "ACCEPT"},
	}

	if !reflect.DeepEqual(rules, expected) {
		t.Errorf("Override same rules failed.\nExpected: %v\nGot: %v", expected, rules)
	}
}

// TestChainDelete verifies chain deletion
func TestChainDelete(t *testing.T) {
	fake := NewFakeIPTables(ProtocolIPv4)

	// Create a chain with rules
	fake.SetInitialRules("filter", "MY_CHAIN", [][]string{
		{"-j", "ACCEPT"},
		{"-j", "DROP"},
	})

	ipt := NewIPTables(fake)

	// Register for deletion
	err := ipt.RegisterChainDelete("filter", "MY_CHAIN")
	if err != nil {
		t.Fatalf("RegisterChainDelete failed: %v", err)
	}

	err = ipt.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify that the chain is deleted
	if fake.ChainExists("filter", "MY_CHAIN") {
		t.Error("Chain should be deleted but still exists")
	}
}

// TestChainDeleteEmpty verifies deletion of an empty chain
func TestChainDeleteEmpty(t *testing.T) {
	fake := NewFakeIPTables(ProtocolIPv4)

	// Create an empty chain
	fake.SetInitialRules("filter", "EMPTY_CHAIN", [][]string{})

	ipt := NewIPTables(fake)

	err := ipt.RegisterChainDelete("filter", "EMPTY_CHAIN")
	if err != nil {
		t.Fatalf("RegisterChainDelete failed: %v", err)
	}

	err = ipt.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	if fake.ChainExists("filter", "EMPTY_CHAIN") {
		t.Error("Empty chain should be deleted but still exists")
	}
}

// TestChainDeleteNonExistent verifies that deleting a non-existent chain does not cause errors
func TestChainDeleteNonExistent(t *testing.T) {
	fake := NewFakeIPTables(ProtocolIPv4)

	// Do NOT create chain - it does not exist

	ipt := NewIPTables(fake)

	err := ipt.RegisterChainDelete("filter", "NON_EXISTENT")
	if err != nil {
		t.Fatalf("RegisterChainDelete failed: %v", err)
	}

	// Commit should not panic
	err = ipt.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Chain still does not exist
	if fake.ChainExists("filter", "NON_EXISTENT") {
		t.Error("Non-existent chain should not be created")
	}
}

// TestChainDeleteIgnoresAppend verifies that Append/Insert/Delete are ignored for a chain being deleted
func TestChainDeleteIgnoresAppend(t *testing.T) {
	fake := NewFakeIPTables(ProtocolIPv4)

	fake.SetInitialRules("filter", "TO_DELETE", [][]string{
		{"-j", "ACCEPT"},
	})

	ipt := NewIPTables(fake)

	err := ipt.RegisterChainDelete("filter", "TO_DELETE")
	if err != nil {
		t.Fatalf("RegisterChainDelete failed: %v", err)
	}

	// These operations should be ignored
	_ = ipt.Append("filter", "TO_DELETE", "-j", "DROP")
	_ = ipt.Insert("filter", "TO_DELETE", 1, "-j", "LOG")
	_ = ipt.Delete("filter", "TO_DELETE", "-j", "ACCEPT")

	err = ipt.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	if fake.ChainExists("filter", "TO_DELETE") {
		t.Error("Chain should be deleted")
	}
}

// TestMixedChainTypes verifies different chain types working within a single table
func TestMixedChainTypes(t *testing.T) {
	fake := NewFakeIPTables(ProtocolIPv4)

	// Initial state
	fake.SetInitialRules("filter", "INPUT", [][]string{
		{"-j", "ACCEPT"},
	})
	fake.SetInitialRules("filter", "FORWARD", [][]string{
		{"-j", "OLD_RULE"},
	})
	fake.SetInitialRules("filter", "TO_DELETE", [][]string{
		{"-j", "SOMETHING"},
	})

	ipt := NewIPTables(fake)

	// INPUT - overlay (append to existing)
	_ = ipt.RegisterChainPatch("filter", "INPUT")
	_ = ipt.Append("filter", "INPUT", "-j", "DROP")

	// FORWARD - override (full replacement)
	_ = ipt.RegisterChainOverride("filter", "FORWARD")
	_ = ipt.Append("filter", "FORWARD", "-j", "NEW_RULE")

	// TO_DELETE - delete
	_ = ipt.RegisterChainDelete("filter", "TO_DELETE")

	err := ipt.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify INPUT (overlay)
	inputRules := fake.GetRules("filter", "INPUT")
	expectedInput := [][]string{
		{"-j", "ACCEPT"},
		{"-j", "DROP"},
	}
	if !reflect.DeepEqual(inputRules, expectedInput) {
		t.Errorf("INPUT overlay failed.\nExpected: %v\nGot: %v", expectedInput, inputRules)
	}

	// Verify FORWARD (override)
	forwardRules := fake.GetRules("filter", "FORWARD")
	expectedForward := [][]string{
		{"-j", "NEW_RULE"},
	}
	if !reflect.DeepEqual(forwardRules, expectedForward) {
		t.Errorf("FORWARD override failed.\nExpected: %v\nGot: %v", expectedForward, forwardRules)
	}

	// Verify TO_DELETE (should be deleted)
	if fake.ChainExists("filter", "TO_DELETE") {
		t.Error("TO_DELETE chain should not exist")
	}
}

// TestMultipleCommits verifies multiple sequential commits
func TestMultipleCommits(t *testing.T) {
	fake := NewFakeIPTables(ProtocolIPv4)

	ipt := NewIPTables(fake)

	// First commit: create a chain
	_ = ipt.RegisterChainOverride("filter", "MY_CHAIN")
	_ = ipt.Append("filter", "MY_CHAIN", "-j", "ACCEPT")

	err := ipt.Commit()
	if err != nil {
		t.Fatalf("First commit failed: %v", err)
	}

	rules := fake.GetRules("filter", "MY_CHAIN")
	if len(rules) != 1 || rules[0][1] != "ACCEPT" {
		t.Errorf("After first commit: %v", rules)
	}

	// Second commit: add another rule
	_ = ipt.Append("filter", "MY_CHAIN", "-j", "DROP")

	err = ipt.Commit()
	if err != nil {
		t.Fatalf("Second commit failed: %v", err)
	}

	rules = fake.GetRules("filter", "MY_CHAIN")
	expected := [][]string{
		{"-j", "ACCEPT"},
		{"-j", "DROP"},
	}
	if !reflect.DeepEqual(rules, expected) {
		t.Errorf("After second commit.\nExpected: %v\nGot: %v", expected, rules)
	}
}

// TestIPv6 verifies IPv6 functionality
func TestIPv6(t *testing.T) {
	fake := NewFakeIPTables(ProtocolIPv6)

	ipt := NewIPTables(fake)

	if ipt.Proto() != ProtocolIPv6 {
		t.Error("Protocol should be IPv6")
	}

	_ = ipt.RegisterChainOverride("filter", "INPUT")
	_ = ipt.Append("filter", "INPUT", "-s", "::1", "-j", "ACCEPT")

	err := ipt.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	rules := fake.GetRules("filter", "INPUT")
	expected := [][]string{
		{"-s", "::1", "-j", "ACCEPT"},
	}

	if !reflect.DeepEqual(rules, expected) {
		t.Errorf("IPv6 rules failed.\nExpected: %v\nGot: %v", expected, rules)
	}
}

// TestErrorOnUninitializedChain verifies error when working with an unregistered chain
func TestErrorOnUninitializedChain(t *testing.T) {
	fake := NewFakeIPTables(ProtocolIPv4)
	ipt := NewIPTables(fake)

	err := ipt.Append("filter", "NONEXISTENT", "-j", "ACCEPT")
	if err != ErrChainNotInitialized {
		t.Errorf("Expected ErrChainNotInitialized, got: %v", err)
	}

	err = ipt.Insert("filter", "NONEXISTENT", 1, "-j", "ACCEPT")
	if err != ErrChainNotInitialized {
		t.Errorf("Expected ErrChainNotInitialized, got: %v", err)
	}

	err = ipt.Delete("filter", "NONEXISTENT", "-j", "ACCEPT")
	if err != ErrChainNotInitialized {
		t.Errorf("Expected ErrChainNotInitialized, got: %v", err)
	}
}

// TestPatchRemovesDuplicates verifies duplicate removal in overlay mode
func TestPatchRemovesDuplicates(t *testing.T) {
	fake := NewFakeIPTables(ProtocolIPv4)

	// Initial state with duplicates
	fake.SetInitialRules("filter", "FORWARD", [][]string{
		{"-j", "ACCEPT"},
		{"-j", "ACCEPT"}, // duplicate
		{"-j", "DROP"},
	})

	ipt := NewIPTables(fake)

	err := ipt.RegisterChainPatch("filter", "FORWARD")
	if err != nil {
		t.Fatalf("RegisterChainPatch failed: %v", err)
	}

	// Appending the same rule should remove duplicates but not add a new one
	err = ipt.Append("filter", "FORWARD", "-j", "ACCEPT")
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	err = ipt.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	rules := fake.GetRules("filter", "FORWARD")
	// Only one -j ACCEPT and one -j DROP should remain
	expected := [][]string{
		{"-j", "ACCEPT"},
		{"-j", "DROP"},
	}

	if !reflect.DeepEqual(rules, expected) {
		t.Errorf("Patch duplicate removal failed.\nExpected: %v\nGot: %v", expected, rules)
	}
}
