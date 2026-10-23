package frontend

import "testing"

func TestFileSystemAlwaysServesAnIndex(t *testing.T) {
	file, err := FileSystem().Open("/index.html")
	if err != nil {
		t.Fatalf("index.html missing from embedded frontend: %v", err)
	}
	_ = file.Close()
}
