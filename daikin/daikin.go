package daikin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type Daikin struct {
	Email          string
	Password       string
	tokenCache     *Token
	tokenExpiresAt time.Time
	httpClient     *http.Client
	urlBase        string
}

type Token struct {
	AccessToken          string `json:"accessToken"`
	AccessTokenExpiresIn int    `json:"accessTokenExpiresIn"`
	RefreshToken         string `json:"refreshToken"`
	TokenType            string `json:"tokenType"`
}

type Devices []Device

type Device struct {
	Id              string `json:"id"`
	LocationId      string `json:"locationId"`
	Name            string `json:"name"`
	Model           string `json:"model"`
	FirmwareVersion string `json:"firmwareVersion"`
	CreatedDate     int    `json:"createdDate"`
	HasOwner        bool   `json:"hasOwner"`
	HasWrite        bool   `json:"hasWrite"`
}

type EquipmentStatus uint8

const (
	EquipmentStatusCool EquipmentStatus = iota + 1
	EquipmentStatusOvercool
	EquipmentStatusHeat
	EquipmentStatusFan
	EquipmentStatusIdle
)

type Mode uint8

const (
	ModeOff Mode = iota
	ModeHeat
	ModeCool
	ModeAuto
	ModeEmHeat
)

type FanCirculateSpeed uint8

const (
	FanCirculateSpeedLow FanCirculateSpeed = iota
	FanCirculateSpeedMed
	FanCirculateSpeedHigh
)

type FanCirculate uint8

const (
	FanCirculateOff FanCirculate = iota
	FanCirculateOn
	FanCirculateSched
)

type DeviceInfo struct {
	CSPHome                float32           `json:"cspHome"`
	HSPHome                float32           `json:"hspHome"`
	FanCirculateSpeed      FanCirculateSpeed `json:"fanCirculateSpeed"`
	EquipmentStatus        EquipmentStatus   `json:"equipmentStatus"`
	HumOutdoor             int               `json:"humOutdoor"`
	TempIndoor             float32           `json:"tempIndoor"`
	TempDeltaMin           float32           `json:"tempDeltaMin"`
	EquipmentCommunication int               `json:"equipmentCommunication"`
	ModeEmHeatAvailable    bool              `json:"modeEmHeatAvailable"`
	GeofencingEnabled      bool              `json:"geofencingEnabled"`
	SchedEnabled           bool              `json:"schedEnabled"`
	HumIndoor              int               `json:"humIndoor"`
	ModeLimit              int               `json:"modeLimit"`
	Fan                    bool              `json:"fan"`
	FanCirculate           FanCirculate      `json:"fanCirculate"`
	TempOutdoor            float32           `json:"tempOutdoor"`
	Mode                   Mode              `json:"mode"`
}

func New(email string, password string) *Daikin {
	d := Daikin{
		Email:      email,
		Password:   password,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		urlBase:    "https://api.daikinskyport.com",
	}
	return &d
}

func (d *Daikin) getToken() (string, error) {

	if d.tokenCache != nil && time.Now().Before(d.tokenExpiresAt) {
		return d.tokenCache.AccessToken, nil
	}

	body := []byte(`{
		"email": "` + d.Email + `",
		"password": "` + d.Password + `"
	}`)

	r, err := http.NewRequest("POST", d.urlBase+"/users/auth/login", bytes.NewBuffer(body))
	if err != nil {
		return "", errors.New("http.NewRequest failed")
	}

	r.Header.Add("content-type", "application/json")

	res, err := d.httpClient.Do(r)
	if err != nil {
		return "", errors.New("http request failed")
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request returned a non-success response: %s", res.Status)
	}

	token := &Token{}
	derr := json.NewDecoder(res.Body).Decode(token)
	if derr != nil {
		return "", errors.New("json decode failed")
	}

	d.tokenCache = token
	d.tokenExpiresAt = time.Now().Add(time.Duration(token.AccessTokenExpiresIn) * time.Second)

	return token.AccessToken, nil
}

func (d *Daikin) GetDevices() (*Devices, error) {
	r, err := http.NewRequest("GET", d.urlBase+"/devices", nil)
	if err != nil {
		return nil, errors.New("http.NewRequest failed")
	}

	r.Header.Add("content-type", "application/json")

	token, err := d.getToken()
	if err != nil {
		return nil, errors.New("getToken did not return a token")
	}

	r.Header.Add("Authorization", "Bearer "+token)

	res, err := d.httpClient.Do(r)
	if err != nil {
		return nil, errors.New("http request failed")
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get devices request returned a non-success response: %s", res.Status)
	}

	var devices Devices
	derr := json.NewDecoder(res.Body).Decode(&devices)
	if derr != nil {
		return nil, errors.New("json decode failed")
	}

	return &devices, nil
}

func (d *Daikin) GetDeviceInfo(deviceId string) (*DeviceInfo, error) {
	r, err := http.NewRequest("GET", d.urlBase+"/deviceData/"+deviceId, nil)
	if err != nil {
		return nil, errors.New("http.NewRequest failed")
	}

	r.Header.Add("content-type", "application/json")

	token, err := d.getToken()
	if err != nil {
		return nil, errors.New("getToken did not return a token")
	}

	r.Header.Add("Authorization", "Bearer "+token)

	res, err := d.httpClient.Do(r)
	if err != nil {
		return nil, errors.New("http request failed")
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get device info request returned a non-success response: %s", res.Status)
	}

	var deviceInfo DeviceInfo
	derr := json.NewDecoder(res.Body).Decode(&deviceInfo)
	if derr != nil {
		return nil, errors.New("json decode failed")
	}

	return &deviceInfo, nil
}

func (d *Daikin) SetMode(deviceId string, mode Mode) error {
	json := `{ "mode": ` + strconv.Itoa(int(mode)) + `}`
	return d.UpdateDevice(deviceId, json)
}

func (d *Daikin) UpdateDevice(deviceId string, json string) error {
	body := []byte(json)

	r, err := http.NewRequest("PUT", d.urlBase+"/deviceData/"+deviceId, bytes.NewBuffer([]byte(body)))
	if err != nil {
		return errors.New("http.NewRequest failed")
	}

	r.Header.Add("content-type", "application/json")

	token, err := d.getToken()
	if err != nil {
		return errors.New("getToken did not return a token")
	}

	r.Header.Add("Authorization", "Bearer "+token)

	res, err := d.httpClient.Do(r)
	if err != nil {
		return errors.New("http request failed")
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("update request returned a non-success response: %s", res.Status)
	}

	return nil
}
