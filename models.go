package facebook

import (
	"encoding/json"
	"strconv"
)

// UserProfile is Facebook user profile
type UserProfile struct {
	ID         string      `json:"id"`
	FirstName  string      `json:"first_name"`
	LastName   string      `json:"last_name"`
	Email      string      `json:"email"`
	Picture    string      `json:"picture"`
	Accounts   []Account   `json:"accounts"`
	AdAccounts []AdAccount `json:"adaccounts"`
}

// UnmarshalJSON override
func (u *UserProfile) UnmarshalJSON(b []byte) error {
	type U UserProfile
	var v struct {
		U
		Picture    struct{ Data struct{ URL string } } `json:"picture"`
		Accounts   struct{ Data []Account }            `json:"accounts"`
		AdAccounts struct{ Data []AdAccount }          `json:"adaccounts"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	v.U.Accounts = v.Accounts.Data
	v.U.AdAccounts = v.AdAccounts.Data
	v.U.Picture = v.Picture.Data.URL
	*u = UserProfile(v.U)
	return nil
}

// Account is a user/page account
type Account struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AccessToken string `json:"access_token"`
	Picture     string `json:"picture"`
}

// UnmarshalJSON override
func (a *Account) UnmarshalJSON(b []byte) error {
	type A Account
	var v struct {
		A
		Picture struct{ Data struct{ URL string } } `json:"picture"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	v.A.Picture = v.Picture.Data.URL
	*a = Account(v.A)
	return nil
}

// AdAccount is the user Ad Account
type AdAccount struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
}

// UnmarshalJSON override
func (a *AdAccount) UnmarshalJSON(b []byte) error {
	type A AdAccount
	var v struct {
		A
		Picture struct{ Data struct{ URL string } } `json:"picture"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*a = AdAccount(v.A)
	return nil
}

// Album is a Facebook Album
type Album struct {
	ID      string
	Name    string
	Message string
	Images  []Image
}

// Image is a Facebook Image
type Image struct {
	ID     string
	Images []struct {
		Width  int
		Height int
		Source string
	}
}

// UploadSession is the start of a Download
type UploadSession struct {
	AdAccount string `json:"-"`
	Path      string `json:"-"`
	Size      int64  `json:"-"`

	Success *bool  `json:"success"`
	Title   string `json:"title"`
	Descr   string `json:"description"`

	UploadSessionID string `json:"upload_session_id"`
	VideoID         string `json:"video_id"`
	StartOffset     int64  `json:"start_offset"`
	EndOffset       int64  `json:"end_offset"`
}

// Progress returns a value between 0 and 1
func (u *UploadSession) Progress() float64 {
	switch {
	case u.UploadSessionID == "":
		return 0
	case u.StartOffset != u.EndOffset:
		return float64(u.StartOffset) / float64(u.Size)
	default:
		return 1
	}
}

// UnmarshalJSON override
func (u *UploadSession) UnmarshalJSON(b []byte) error {
	type U UploadSession
	var v = struct {
		U
		StartOffset string `json:"start_offset"`
		EndOffset   string `json:"end_offset"`
	}{U: U(*u)}
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	*u = UploadSession(v.U)
	if v.StartOffset != "" {
		u.StartOffset, err = strconv.ParseInt(v.StartOffset, 10, 64)
		if err != nil {
			return err
		}
	} else {
		u.StartOffset = 0
	}
	if v.EndOffset != "" {
		u.EndOffset, err = strconv.ParseInt(v.EndOffset, 10, 64)
		if err != nil {
			return err
		}
	} else {
		u.EndOffset = 0
	}
	return nil
}
