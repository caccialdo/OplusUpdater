package updater

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/deatil/go-cryptobin/cryptobin/crypto"
)

type Attribute struct {
	Zone       string
	Mode       int
	OtaVer     string
	AndroidVer string
	ColorOSVer string
	ProxyStr   string
	CarrierID  string
}

func (attr *Attribute) postProcessing() {
	// maybe only support at realme.
	// i dont known oppo and oplus 's version format.
	if len(strings.Split(attr.OtaVer, "_")) < 3 || len(strings.Split(attr.OtaVer, ".")) < 3 {
		attr.OtaVer += ".00_0000_000000000000"
	}
	if attr.Zone == "" {
		attr.Zone = "CN"
	}
	if attr.AndroidVer == "" {
		attr.AndroidVer = "nil"
	}
	if attr.ColorOSVer == "" {
		attr.ColorOSVer = "nil"
	}
	if attr.Mode == 0 {
		attr.Mode = 0
	}
}

type UpdateResponseCipher struct {
	Parent     string `json:"parent"`
	Components []struct {
		ComponentId      string `json:"componentId"`
		ComponentName    string `json:"componentName"`
		ComponentVersion string `json:"componentVersion"`

		ComponentPackets struct {
			Size    string `json:"size"`
			VabInfo struct {
				Data struct {
					OtaStreamingProperty string   `json:"otaStreamingProperty"`
					VabPackageHash       string   `json:"vab_package_hash"`
					ExtraParams          string   `json:"extra_params"`
					Header               []string `json:"header"`
				} `json:"data"`
			} `json:"vabInfo"`
			ManualUrl string `json:"manualUrl"`
			Id        string `json:"id"`
			Type      string `json:"type"`
			Url       string `json:"url"`
			Md5       string `json:"md5"`
		} `json:"componentPackets"`
	} `json:"components"`

	SecurityPatch   string `json:"securityPatch"`
	RealVersionName string `json:"realVersionName"`
	OtaVersion      string `json:"otaVersion"`
	IsNvDescription bool   `json:"isNvDescription"`

	Description struct {
		Opex       struct{} `json:"opex"`
		Share      string   `json:"share"`
		PanelUrl   string   `json:"panelUrl"`
		Url        string   `json:"url"`
		FirstTitle string   `json:"firstTitle"`
	} `json:"description"`

	VersionTypeId string `json:"versionTypeId"`
	VersionName   string `json:"versionName"`
	Rid           string `json:"rid"`
	ReminderValue struct {
		Download struct {
			Notice  []int  `json:"notice"`
			Pop     []int  `json:"pop"`
			Version string `json:"version"`
		} `json:"download"`
		Upgrade struct {
			Notice  []int  `json:"notice"`
			Pop     []int  `json:"pop"`
			Version string `json:"version"`
		} `json:"upgrade"`
	} `json:"reminderValue"`
	IsRecruit             bool     `json:"isRecruit"`
	RealAndroidVersion    string   `json:"realAndroidVersion"`
	OpexInfo              []string `json:"opexInfo"`
	IsSecret              bool     `json:"isSecret"`
	RealOsVersion         string   `json:"realOsVersion"`
	OsVersion             string   `json:"osVersion"`
	PublishedTime         int64    `json:"publishedTime"`
	ComponentAssembleType bool     `json:"componentAssembleType"`
	IsV5GkaVersion        int      `json:"isV5GkaVersion"`
	GooglePatchInfo       string   `json:"googlePatchInfo"`
	Id                    string   `json:"id"`
	ColorOSVersion        string   `json:"colorOSVersion"`
	IsConfidential        int      `json:"isConfidential"`
	BetaTasteInteract     bool     `json:"betaTasteInteract"`
	ParamFlag             int      `json:"paramFlag"`
	ReminderType          int      `json:"reminderType"`
	NoticeType            int      `json:"noticeType"`
	Decentralize          struct {
		StrategyVersion string `json:"strategyVersion"`
		Round           int    `json:"round"`
		Offset          int    `json:"offset"`
	} `json:"decentralize"`
	VersionCode         int    `json:"versionCode"`
	SilenceUpdate       int    `json:"silenceUpdate"`
	SecurityPatchVendor string `json:"securityPatchVendor"`
	GkaReq              int    `json:"gkaReq"`
	RealOtaVersion      string `json:"realOtaVersion"`
	AndroidVersion      string `json:"androidVersion"`
	NightUpdateLimit    string `json:"nightUpdateLimit"`
	VersionTypeH5       string `json:"versionTypeH5"`
	Aid                 string `json:"aid"`
	NvId16              string `json:"nvId16"`
	Status              string `json:"status"`
}

func QueryUpdater(attr *Attribute) (*UpdateResponseCipher, error) {
	rawBytes, err := QueryUpdaterRawBytes(attr)
	if err != nil {
		return nil, err
	}
	var cipher UpdateResponseCipher
	if err := json.Unmarshal(rawBytes, &cipher); err != nil {
		return nil, err
	}
	return &cipher, nil
}

func QueryUpdaterRawBytes(attr *Attribute) ([]byte, error) {
	attr.postProcessing()
	deviceId := GetDefaultDeviceId()

	c := GetConfig(attr.Zone)
	key, err := RandomKey()
	if err != nil {
		return nil, err
	}
	iv, err := RandomIv()
	if err != nil {
		return nil, err
	}
	protectedKey, err := GenerateProtectedKey(key, []byte(c.PublicKey))
	if err != nil {
		return nil, err
	}

	headers := UpdateRequestHeaders{
		AndroidVersion: attr.AndroidVer, // or Android13
		ColorOSVersion: attr.ColorOSVer, // or ColorOS13.1.0
		OtaVersion:     attr.OtaVer,
		ProtectedKey: map[string]CryptoConfig{
			"SCENE_1": {
				ProtectedKey:       protectedKey,
				Version:            GenerateProtectedVersion(),
				NegotiationVersion: c.PublicKeyVersion,
			},
		},
	}
	headers.SetHashedDeviceId(deviceId)
	cipher := NewUpdateRequestCipher(attr.Mode, deviceId)

	if attr.CarrierID != "" {
		c.CarrierID = attr.CarrierID
	}

	reqHeaders, err := headers.CreateRequestHeader(c)
	if err != nil {
		return nil, err
	}
	reqBody, err := cipher.CreateRequestBody(key, iv)
	if err != nil {
		return nil, err
	}

	req := &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "https", Host: c.Host, Path: "/update/v3"},
		Header: reqHeaders,
		Body:   io.NopCloser(bytes.NewBuffer(reqBody)),
	}

	transport, err := ParseTransportFromProxyStr(attr.ProxyStr)
	if err != nil {
		transport = &http.Transport{}
		log.Printf("Error in ParseTransportFromProxyStr: %v, not set.", err)
	}

	client := &http.Client{
		Transport: transport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {}(req.Body)

	var result ResponseResult
	if json.NewDecoder(resp.Body).Decode(&result) != nil {
		return nil, err
	}

	return decryptUpdateResponse(&result, key)
}

func decryptUpdateResponse(r *ResponseResult, key []byte) ([]byte, error) {
	if r.ResponseCode != 200 {
		return nil, fmt.Errorf("response code: %d, message: %s", r.ResponseCode, r.ErrMsg)
	}

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(r.Body.(string)), &m); err != nil {
		return nil, err
	}

	iv, err := base64.StdEncoding.DecodeString(m["iv"].(string))
	if err != nil {
		return nil, err
	}
	cipherBytes := crypto.FromBase64String(m["cipher"].(string)).
		Aes().CTR().NoPadding().
		WithKey(key).WithIv(iv).
		Decrypt().ToBytes()

	return cipherBytes, nil
}
