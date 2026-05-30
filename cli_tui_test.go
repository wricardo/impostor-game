package main

import "testing"

func TestCommandModeDefaultsToServe(t *testing.T) {
	if got := commandMode(nil); got != "serve" {
		t.Fatalf("commandMode(nil) = %q, want serve", got)
	}
}

func TestCommandModeReturnsSubcommand(t *testing.T) {
	if got := commandMode([]string{"tui", "--local"}); got != "tui" {
		t.Fatalf("commandMode(tui) = %q, want tui", got)
	}
}

func TestAdjacentCategoryWraps(t *testing.T) {
	cats := []string{"Random", "Food", "Animals"}
	if got := adjacentCategory(cats, "Animals", 1); got != "Random" {
		t.Fatalf("next category = %q, want Random", got)
	}
	if got := adjacentCategory(cats, "Random", -1); got != "Animals" {
		t.Fatalf("previous category = %q, want Animals", got)
	}
}
