package facebook

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
)

func fields(list ...string) map[string]interface{} {
	return map[string]interface{}{
		"fields": strings.Join(list, ","),
	}
}

// New retuns a new Client with the specified Auth Token
func New(token, version string) *Client {
	return &Client{
		client:  &fasthttp.Client{},
		token:   token,
		version: version,
	}
}

// Client is a Facebook API client
type Client struct {
	client  *fasthttp.Client
	version string
	token   string
}

// Execute is the general method to call the API with the selected method/path.
// Response is unmarshalled inside r (that must be a pointer)
func (c *Client) Execute(method, path string, params map[string]interface{}, r interface{}, u ...Upload) error {
	var (
		req   = fasthttp.AcquireRequest()
		uri   = fasthttp.AcquireURI()
		query = fasthttp.AcquireArgs()
		body  = bytebufferpool.Get()
		resp  = fasthttp.AcquireResponse()
	)
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseURI(uri)
		fasthttp.ReleaseArgs(query)
		bytebufferpool.Put(body)
		fasthttp.ReleaseResponse(resp)
	}()

	switch method {
	case "GET":
		for k, v := range params {
			query.Add(k, fmt.Sprint(v))
		}
	case "POST":
		if len(u) == 0 {
			form := make(url.Values, len(params))
			for k, v := range params {
				form.Set(k, fmt.Sprint(v))
			}
			fmt.Fprint(body, form.Encode())
			break
		}
		w := multipart.NewWriter(body)
		for _, v := range u {
			fw, err := w.CreateFormFile(v.Name, v.FileName)
			if err != nil {
				return err
			}
			if _, err := v.Data.Seek(v.From, io.SeekStart); err != nil {
				return err
			}
			if _, err := io.CopyN(fw, v.Data, v.To-v.From); err != nil {
				return err
			}
		}
		for k, v := range params {
			if err := w.WriteField(k, fmt.Sprint(v)); err != nil {
				return err
			}
		}
		if err := w.Close(); err != nil {
			return err
		}
		req.Header.Set("Content-Type", w.FormDataContentType())

	}

	query.Add("access_token", c.token)
	uri.Update("https://graph.facebook.com")
	uri.SetPath(fmt.Sprintf("/%s/%s", c.version, path))
	uri.SetQueryStringBytes(query.QueryString())
	req.SetRequestURIBytes(uri.FullURI())
	req.SetBody(body.B)
	req.Header.SetMethod(method)

	if err := c.client.Do(req, resp); err != nil {
		return err
	}

	if code := resp.StatusCode(); code != http.StatusOK && code != http.StatusCreated {
		var v struct{ Error APIError }
		if err := json.Unmarshal(resp.Body(), &v); err != nil {
			return fmt.Errorf("Status: %v", code)
		}
		return &v.Error
	}
	return json.Unmarshal(resp.Body(), r)
}

var profileFields = fields("first_name", "last_name", "email", "picture", "adaccounts", "accounts{name,id,access_token,picture}")

// UserProfile returns a UserProfile
func (c *Client) UserProfile(userID string) (*UserProfile, error) {
	var v UserProfile
	if err := c.Execute("GET", fmt.Sprintf("/%s", userID), profileFields, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Albums returns a Albums
func (c *Client) Albums(userID string) ([]Album, error) {
	var v struct{ Data []Album }
	if err := c.Execute("GET", fmt.Sprintf("/%s/albums", userID), nil, &v); err != nil {
		return nil, err
	}
	return v.Data, nil
}

// Album returns a Album
func (c *Client) Album(albumID string) (*Album, error) {
	var v struct {
		Album
		Photos struct {
			Data []Image
		}
	}
	if err := c.Execute("GET", fmt.Sprintf("/%s", albumID), fields("id", "name", "photos{images}"), &v); err != nil {
		return nil, err
	}
	v.Album.Images = v.Photos.Data
	return &v.Album, nil
}

// CreateAlbum creates a new Album
func (c *Client) CreateAlbum(userID string, a *Album, p Privacy) error {
	return c.Execute("POST", fmt.Sprintf("/%s/albums", userID), map[string]interface{}{
		"name":    a.Name,
		"message": a.Message,
		"privacy": fmt.Sprintf(`{"value":%q}`, p),
	}, a)
}

// UploadVideo creates a new upload session
func (c *Client) UploadVideo(u *UploadSession) error {
	f, err := os.Open(u.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	u.Size = info.Size()
	switch {
	case u.UploadSessionID == "":
		return c.Execute("POST", fmt.Sprintf("/%s/advideos", u.AdAccount), map[string]interface{}{
			"upload_phase": "start",
			"file_size":    u.Size,
		}, u)
	case u.StartOffset != u.EndOffset:
		return c.Execute("POST", fmt.Sprintf("/%s/advideos", u.AdAccount), map[string]interface{}{
			"upload_phase":      "transfer",
			"start_offset":      u.StartOffset,
			"upload_session_id": u.UploadSessionID,
		}, u, Upload{
			Data:     f,
			Name:     "video_file_chunk",
			FileName: path.Base(u.Path),
			From:     u.StartOffset,
			To:       u.EndOffset,
		})
	default:
		return c.Execute("POST", fmt.Sprintf("/%s/advideos", u.AdAccount), map[string]interface{}{
			"upload_phase":      "finish",
			"upload_session_id": u.UploadSessionID,
			"title":             u.Title,
			"description":       u.Descr,
		}, u)
	}
}

// IsTemp returns if an error is temporary
func (c *Client) IsTemp(err error) bool {
	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}
	switch apiErr.Code {
	case 613: // Calls to this api have exceeded the rate limit.
		return false
	}
	return true
}

// Privacy is the element privacy level
type Privacy string

const (
	// PrivacyPrivate is just visible to the user
	PrivacyPrivate Privacy = "SELF"
	// PrivacyFriends is visible to all user friends
	PrivacyFriends Privacy = "ALL_FRIENDS"
)

// APIError is a facebook API error
type APIError struct {
	Message        string `json:"message"`
	Type           string `json:"type"`
	Code           int    `json:"code"`
	ErrorSubcode   int    `json:"error_subcode"`
	IsTransient    bool   `json:"is_transient"`
	ErrorUserTitle string `json:"error_user_title"`
	ErrorUserMsg   string `json:"error_user_msg"`
	FBTraceID      string `json:"fbtrace_id"`
}

func (a *APIError) Error() string {
	return fmt.Sprintf("%s: %s (%v) %s: %s", a.Type, a.Message, a.Code, a.ErrorUserTitle, a.ErrorUserMsg)
}

// Upload is a chunked upload request
type Upload struct {
	Name     string
	FileName string
	Data     io.ReadSeeker
	From, To int64
}
