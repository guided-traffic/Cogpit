package main

import (
	"slices"
	"testing"
)

// isTest follows the go tool's rule: the prefix alone, or followed by a rune that is not lower case.
func TestIsTest(t *testing.T) {
	for name, want := range map[string]bool{
		"Test": true, "TestShell": true, "Test_x": true, "Test1": true, "Testing": false,
		"Example": true, "ExampleRun": true, "Examples": false, "FuzzParse": true, "helper": false,
	} {
		prefix := "Test"
		switch name[0] {
		case 'E':
			prefix = "Example"
		case 'F':
			prefix = "Fuzz"
		}
		if got := isTest(name, prefix); got != want {
			t.Errorf("isTest(%q, %q) = %v, want %v", name, prefix, got, want)
		}
	}
}

// Every name lands in exactly one shard, the same one each time, and a name added later does not
// move the others.
func TestSplitIsStableAndComplete(t *testing.T) {
	names := []string{"TestA", "TestB", "TestC", "TestD", "TestE", "TestF", "TestG", "TestH"}
	shards := split(names, 3)
	var all []string
	for _, s := range shards {
		all = append(all, s...)
	}
	slices.Sort(all)
	if !slices.Equal(all, names) {
		t.Fatalf("the shards hold %v, want each of %v once", all, names)
	}
	if again := split(names, 3); !slices.EqualFunc(shards, again, slices.Equal) {
		t.Errorf("split changed between calls: %v, then %v", shards, again)
	}
	grown := split(append(slices.Clone(names), "TestNew"), 3)
	for i := range shards {
		kept := slices.DeleteFunc(slices.Clone(grown[i]), func(n string) bool { return n == "TestNew" })
		if !slices.Equal(kept, shards[i]) {
			t.Errorf("shard %d was %v and is %v after adding a test", i+1, shards[i], kept)
		}
	}
	if got := pattern([]string{"TestA", "TestB"}); got != "^(TestA|TestB)$" {
		t.Errorf("pattern = %q", got)
	}
}

// The package's own tests are found from its files, TestMain left out.
func TestTestNamesOfThisPackage(t *testing.T) {
	names, err := testNames(".", "")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"TestIsTest", "TestSplitIsStableAndComplete", "TestTestNamesOfThisPackage"}
	if !slices.Equal(names, want) {
		t.Errorf("testNames = %v, want %v", names, want)
	}
}
