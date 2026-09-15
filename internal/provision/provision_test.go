package provision

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"
)

func TestPlatform(t *testing.T) {
	for raw, want := range map[string]string{"Linux\nx86_64\n": "linux-amd64", "Darwin\narm64\n": "darwin-arm64"} {
		got, e := Platform(raw)
		if got != want || e != nil {
			t.Fatal(got, e)
		}
	}
	if _, e := Platform("Welcome!\nLinux\nx86_64"); e == nil {
		t.Fatal("startup noise accepted")
	}
}
func TestExtractNoPathTraversal(t *testing.T) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "../../zmx", Typeflag: tar.TypeReg, Mode: 0755, Size: 3})
	tw.Write([]byte("bin"))
	tw.Close()
	gz.Close()
	data, e := ExtractZMX(b.Bytes())
	if e != nil || string(data) != "bin" {
		t.Fatal(e)
	} /* extracted to memory, never uses archive path */
}
