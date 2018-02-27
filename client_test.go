package facebook

import (
	"os"
	"testing"
)

func makeClient(t testing.TB) *Client {
	token := os.Getenv("FB_AUTH_TOKEN")
	if token == "" {
		t.Fatalf("$FB_AUTH_TOKEN not specified")
	}
	return New(token, "v2.11")
}

func TestUserProfile(t *testing.T) {
	client := makeClient(t)
	if t.Failed() {
		return
	}
	v, err := client.UserProfile("me")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Id: %s Name: %s %s Pic: %v", v.ID, v.FirstName, v.LastName, v.Picture)
}

func TestAlbums(t *testing.T) {
	client := makeClient(t)
	if t.Failed() {
		return
	}
	v, err := client.Albums("me")
	if err != nil {
		t.Fatal(err)
	}
	for i := range v {
		v, err := client.Album(v[i].ID)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(v)
	}
}

func TestVideo(t *testing.T) {
	client := makeClient(t)
	if t.Failed() {
		return
	}
	me, err := client.UserProfile("me")
	if err != nil {
		t.Fatal(err)
	}
	u := UploadSession{
		AdAccount: me.AdAccounts[0].ID,
		Path:      `C:\Users\klaid\Downloads\small.mp4`,
		Title:     "Chunked Upload",
		Descr:     "A video uploaded in chunks",
	}
	for u.Success == nil {
		if err := client.UploadVideo(&u); err != nil {
			t.Log("Error:", err)
		}
		t.Logf("%+v\n", u)
	}
	if !*u.Success {
		t.Fatal("Upload Failed")
	}
}

func TestCreateAlbum(t *testing.T) {
	client := makeClient(t)
	if t.Failed() {
		return
	}
	a := Album{Name: "test", Message: "delete me!"}
	err := client.CreateAlbum("me", &a, PrivacyPrivate)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Delete manually album:", a.ID)
}

func BenchmarkUserProfile(b *testing.B) {
	client := makeClient(b)
	if b.Failed() {
		return
	}
	for n := 0; n < b.N; n++ {
		_, err := client.UserProfile("me")
		if err != nil {
			b.Fatal(err)
		}
	}
}
