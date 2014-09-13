package youtube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"apperror"
	"platform/oauth"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
)

// Credentials for Youtube API: https://console.developers.google.com/project/592034655117/apiui/credential
const (
	cID       = "YOUR_CLIENT_ID"
	cSecret   = "YOUR_CLIENT_SECRET"
	cRedirect = "YOUR_REDIRECT_LINK"
)

type video struct {
	Snippet snippet `json:"snippet"`
}

type snippet struct {
	Title string `json:"title"`
	//Description string `json:"description"`
	//Category    string `json:"category"`
}

// Return a user sign in url where the user may grant access to us
func GetSignIn() string {
	params := map[string]string{
		"client_id":     cID,
		"redirect_uri":  cRedirect,
		"scope":         "https://www.googleapis.com/auth/youtube",
		"response_type": "code",
		"access_type":   "offline",
	}
	return oauth.GetSignIn("https://accounts.google.com/o/oauth2/auth", params)
}

// Authenticate and receive a access token using the client code after they provide access to us
func Auth(code string) (*oauth.Client, error) {
	c := oauth.NewClient(cID, cSecret, cRedirect, 1)
	err := oauth.Auth(c, code, "https://accounts.google.com/o/oauth2/token")
	if err != nil {
		return c, err
	}

	return c, nil
}

// Authenticate and receive a access token using the client code after they provide access to us
func Refresh(refresh string) (*oauth.Client, error) {
	c := oauth.NewClient(cID, cSecret, cRedirect, 1)
	c.Token = oauth.Token{RefreshToken: refresh}
	err := oauth.Refresh(c, "https://accounts.google.com/o/oauth2/token")
	if err != nil {
		return c, err
	}

	return c, nil
}

func Upload(path string, c *oauth.Client) (*map[string]interface{}, error) {
	//Create the url params
	v := url.Values{}
	v.Set("part", "contentDetails")
	//v.Set("uploadType", "multipart")

	uri := "https://www.googleapis.com/upload/youtube/v3/videos?" + v.Encode()
	//uri := "http://rileedesign.com"

	params := map[string]string{
	//"oauth_token": c.Token.AccessToken,
	//"part":                  "snippet, contentDetails, statistics, status",
	//"snippet[category]":     "5",
	//"Status[PrivacyStatus]": "unlisted",
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, apperror.Err{err, "Unable to open file", 500}
	}
	defer file.Close()

	// Create a buffer containg a form with information about the track and the track itself
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return nil, apperror.Err{err, "Unable to attach file to form", 500}
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, apperror.Err{err, "Unable to attach file to form", 500}
	}

	snipHeader := textproto.MIMEHeader{}
	snipHeader.Set("Content-Disposition", `form-data; name=""; filename="file.json"`)
	snipHeader.Set("Content-Type", "application/json")
	snipHeader.Set("Content-Transfer-Encoding", "binary")

	//Custom snippet thingsy
	snip := video{Snippet: snippet{Title: "testing123"}}
	b, err := json.Marshal(snip)
	fmt.Println(b, err)

	paramSnippet, err := writer.CreatePart(snipHeader)
	if err != nil {
		return nil, apperror.Err{err, "Unable to special write field", 500}
	}
	paramSnippet.Write(b)

	// Iterate through each parameter and add it to a field in the form
	for key, val := range params {
		err := writer.WriteField(key, val)
		if err != nil {
			return nil, apperror.Err{err, "Unable to write field", 500}
		}
	}
	err = writer.Close()
	if err != nil {
		return nil, apperror.Err{err, "Unable to close writer", 500}
	}

	req, err := http.NewRequest("POST", uri, body)
	fmt.Println(err)
	if err != nil {
		return nil, apperror.Err{err, "Can't make request", 500}
	}
	req.Header.Set("Authorization", "Bearer "+c.Token.AccessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	fmt.Println("HEADER: ", req.Header)

	resp, err := c.Client.Do(req)
	fmt.Println("DoErr ", err)
	if err != nil {
		return nil, apperror.Err{err, "Error with request", resp.StatusCode}
	}

	data, err := ioutil.ReadAll(resp.Body)
	fmt.Println("DATA:", string(data), "HEADER:", resp, err)

	var content map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&content)

	return &content, nil
}

// Retrieve all of a user's tracks and information about them
func GetTracks(c *oauth.Client) error {
	u, err := url.Parse("https://api.soundcloud.com/me/tracks.json")
	if err != nil {
		return apperror.Err{err, "Could not parse url", 500}
	}

	q := u.Query()
	q.Set("oauth_token", c.Token.AccessToken)

	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return apperror.Err{err, "Can't make request", 500}
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return apperror.Err{err, "Error with request", resp.StatusCode}
	}

	var content []struct{ Title string }
	err = json.NewDecoder(resp.Body).Decode(&content)
	if err != nil {
		return apperror.Err{err, "Unable to decode response JSON", 500}
	}

	return nil
}
