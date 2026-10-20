package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type account struct {
	Name        string `json:"name"`
	UUID        string `json:"uuid"`
	AccessToken string `json:"accessToken"`
	XUID        string `json:"xuid"`
	UserType    string `json:"userType"`
}

type savedAccount struct {
	Type         string    `json:"type"`
	Name         string    `json:"name"`
	UUID         string    `json:"uuid"`
	XUID         string    `json:"xuid"`
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	Expiry       time.Time `json:"expiry"`
}

type deviceAuth struct {
	DeviceCode string
	Interval   int
	Expiry     time.Time
}

const defaultMSAClientID = ""

func msaClientID() string {
	if v := os.Getenv("VOID_MSA_CLIENT_ID"); v != "" {
		return v
	}
	return defaultMSAClientID
}

func offlineUUID(name string) string {
	h := md5.Sum([]byte("OfflinePlayer:" + name))
	h[6] = (h[6] & 0x0f) | 0x30
	h[8] = (h[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}

func offlineAccount(name string) account {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Player"
	}
	return account{Name: name, UUID: offlineUUID(name), AccessToken: "0", XUID: "0", UserType: "legacy"}
}

func accountPath() string { return filepath.Join(sharedDir(), "account.json") }

func loadSavedAccount() *savedAccount {
	data, err := os.ReadFile(accountPath())
	if err != nil {
		return nil
	}
	var a savedAccount
	if json.Unmarshal(data, &a) != nil || a.Name == "" {
		return nil
	}
	return &a
}

func saveAccount(a *savedAccount) error {
	if err := os.MkdirAll(sharedDir(), 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(a, "", "  ")
	return writeAtomic(accountPath(), data)
}

func clearAccount() { os.Remove(accountPath()) }

func currentAccount() (account, error) {
	a := loadSavedAccount()
	if a == nil {
		return account{}, fmt.Errorf("Sign in with Microsoft, or pick a name to play offline, first.")
	}
	if a.Type == "offline" {
		return offlineAccount(a.Name), nil
	}
	if time.Now().Before(a.Expiry.Add(-60 * time.Second)) {
		return account{Name: a.Name, UUID: a.UUID, AccessToken: a.AccessToken, XUID: a.XUID, UserType: "msa"}, nil
	}
	refreshed, err := refreshMicrosoft(a.RefreshToken)
	if err != nil {
		return account{}, fmt.Errorf("Your Microsoft login expired — please sign in again. (%v)", err)
	}
	return account{Name: refreshed.Name, UUID: refreshed.UUID, AccessToken: refreshed.AccessToken, XUID: refreshed.XUID, UserType: "msa"}, nil
}

func postForm(endpoint string, form url.Values, out any) error {
	resp, err := httpClient.PostForm(endpoint, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	if resp.StatusCode >= 400 {
		var e struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		dec.Decode(&e)
		if e.Error != "" {
			return fmt.Errorf("%s", e.Error)
		}
		return fmt.Errorf("server said %s", resp.Status)
	}
	return dec.Decode(out)
}

func postJSON(endpoint string, body any, out any) error {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server said %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func startDeviceLogin() (userCode, verifyURL string, da *deviceAuth, err error) {
	cid := msaClientID()
	if cid == "" {
		return "", "", nil, fmt.Errorf("Microsoft login isn't set up yet. Set VOID_MSA_CLIENT_ID to an Azure app client id, or play offline.")
	}
	var r struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		Interval        int    `json:"interval"`
		ExpiresIn       int    `json:"expires_in"`
	}
	err = postForm("https://login.microsoftonline.com/consumers/oauth2/v2.0/devicecode", url.Values{
		"client_id": {cid},
		"scope":     {"XboxLive.signin offline_access"},
	}, &r)
	if err != nil {
		return "", "", nil, err
	}
	interval := r.Interval
	if interval <= 0 {
		interval = 5
	}
	return r.UserCode, r.VerificationURI, &deviceAuth{
		DeviceCode: r.DeviceCode,
		Interval:   interval,
		Expiry:     time.Now().Add(time.Duration(r.ExpiresIn) * time.Second),
	}, nil
}

type msToken struct {
	AccessToken  string
	RefreshToken string
}

func pollDeviceToken(da *deviceAuth) (tok *msToken, pending bool, err error) {
	var r struct {
		AccessToken      string `json:"access_token"`
		RefreshToken     string `json:"refresh_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	resp, err := httpClient.PostForm("https://login.microsoftonline.com/consumers/oauth2/v2.0/token", url.Values{
		"client_id":   {msaClientID()},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {da.DeviceCode},
	})
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&r)
	if r.AccessToken != "" {
		return &msToken{AccessToken: r.AccessToken, RefreshToken: r.RefreshToken}, false, nil
	}
	switch r.Error {
	case "authorization_pending", "slow_down":
		return nil, true, nil
	case "":
		return nil, true, nil
	default:
		return nil, false, fmt.Errorf("%s", r.Error)
	}
}

func refreshMicrosoft(refreshToken string) (*savedAccount, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("no saved login")
	}
	var r struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := postForm("https://login.microsoftonline.com/consumers/oauth2/v2.0/token", url.Values{
		"client_id":     {msaClientID()},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         {"XboxLive.signin offline_access"},
	}, &r); err != nil {
		return nil, err
	}
	return finishMicrosoft(&msToken{AccessToken: r.AccessToken, RefreshToken: r.RefreshToken})
}

func finishMicrosoft(tok *msToken) (*savedAccount, error) {
	var xbl struct {
		Token         string `json:"Token"`
		DisplayClaims struct {
			XUI []struct {
				UHS string `json:"uhs"`
			} `json:"xui"`
		} `json:"DisplayClaims"`
	}
	if err := postJSON("https://user.auth.xboxlive.com/user/authenticate", map[string]any{
		"Properties": map[string]any{
			"AuthMethod": "RPS",
			"SiteName":   "user.auth.xboxlive.com",
			"RpsTicket":  "d=" + tok.AccessToken,
		},
		"RelyingParty": "http://auth.xboxlive.com",
		"TokenType":    "JWT",
	}, &xbl); err != nil {
		return nil, fmt.Errorf("Xbox sign-in failed: %w", err)
	}
	if len(xbl.DisplayClaims.XUI) == 0 {
		return nil, fmt.Errorf("Xbox sign-in returned no profile")
	}
	uhs := xbl.DisplayClaims.XUI[0].UHS

	var xsts struct {
		Token         string `json:"Token"`
		DisplayClaims struct {
			XUI []struct {
				XID string `json:"xid"`
			} `json:"xui"`
		} `json:"DisplayClaims"`
	}
	if err := postJSON("https://xsts.auth.xboxlive.com/xsts/authorize", map[string]any{
		"Properties": map[string]any{
			"SandboxId":  "RETAIL",
			"UserTokens": []string{xbl.Token},
		},
		"RelyingParty": "rp://api.minecraftservices.com/",
		"TokenType":    "JWT",
	}, &xsts); err != nil {
		return nil, fmt.Errorf("Xbox authorization failed: %w", err)
	}
	xuid := ""
	if len(xsts.DisplayClaims.XUI) > 0 {
		xuid = xsts.DisplayClaims.XUI[0].XID
	}

	var mc struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := postJSON("https://api.minecraftservices.com/authentication/login_with_xbox", map[string]any{
		"identityToken": "XBL3.0 x=" + uhs + ";" + xsts.Token,
	}, &mc); err != nil {
		return nil, fmt.Errorf("Minecraft sign-in failed: %w", err)
	}

	prof, err := minecraftProfile(mc.AccessToken)
	if err != nil {
		return nil, err
	}
	expiresIn := mc.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 86400
	}
	return &savedAccount{
		Type:         "msa",
		Name:         prof.Name,
		UUID:         dashUUID(prof.ID),
		XUID:         xuid,
		AccessToken:  mc.AccessToken,
		RefreshToken: tok.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(expiresIn) * time.Second),
	}, nil
}

func minecraftProfile(token string) (*struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.minecraftservices.com/minecraft/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("This Microsoft account doesn't own Minecraft Java Edition.")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("couldn't read your Minecraft profile (%s)", resp.Status)
	}
	var prof struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prof); err != nil {
		return nil, err
	}
	return &prof, nil
}

func dashUUID(id string) string {
	if len(id) != 32 {
		return id
	}
	return id[0:8] + "-" + id[8:12] + "-" + id[12:16] + "-" + id[16:20] + "-" + id[20:32]
}
