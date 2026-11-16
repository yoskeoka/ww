package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func TestCheckoutExcludeLayerReplacement(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	repo := t.TempDir()
	dir := filepath.Join(xdg, "ww")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	global := "checkout_exclude = ['*.pem']\n[[projects]]\nroot = " + strconv.Quote(repo) + "\ncheckout_exclude = ['.env']\n"
	if err := os.WriteFile(filepath.Join(dir, GlobalFileName), []byte(global), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadWithOptions(repo, LoadOptions{ProjectRoot: repo})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.CheckoutExclude, []string{".env"}) {
		t.Fatalf("project: %v", cfg.CheckoutExclude)
	}
	if err := os.WriteFile(filepath.Join(repo, FileName), []byte("checkout_exclude = []\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadWithOptions(repo, LoadOptions{ProjectRoot: repo})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.CheckoutExclude) != 0 {
		t.Fatalf("local: %v", cfg.CheckoutExclude)
	}
}
