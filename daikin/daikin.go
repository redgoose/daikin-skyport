package daikin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
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

type SetTempParams struct {
	CoolSetpoint float32
	HeatSetpoint float32
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
	data := map[string]interface{}{"mode": mode}

	json, err := json.Marshal(data)
	if err != nil {
		return errors.New("json marshal failed")
	}

	return d.updateDevice(deviceId, json)
}

func (d *Daikin) SetTemp(deviceId string, params SetTempParams) error {

	if params.CoolSetpoint == params.HeatSetpoint {
		return errors.New("invalid setpoints provided")
	}

	if params.CoolSetpoint != 0 && params.HeatSetpoint != 0 &&
		params.CoolSetpoint < params.HeatSetpoint {
		return errors.New("cool setpoint can not be lower than heat setpoint")
	}

	deviceInfo, err := d.GetDeviceInfo(deviceId)
	if err != nil {
		return errors.New("get device info failed")
	}

	if params.CoolSetpoint == 0 {
		// hsp provided, default csp
		params.CoolSetpoint = deviceInfo.CspHome

		if (params.CoolSetpoint - params.HeatSetpoint) < deviceInfo.TempDeltaMin {
			// min delta not met, increase csp
			params.CoolSetpoint = params.HeatSetpoint + deviceInfo.TempDeltaMin
		}
	}

	if params.HeatSetpoint == 0 {
		// csp provided, default hsp
		params.HeatSetpoint = deviceInfo.HspHome

		if (params.CoolSetpoint - params.HeatSetpoint) < deviceInfo.TempDeltaMin {
			// min delta not met, lower hsp
			params.HeatSetpoint = params.CoolSetpoint - deviceInfo.TempDeltaMin
		}
	}

	if params.CoolSetpoint < deviceInfo.TempSPMin || params.CoolSetpoint > deviceInfo.TempSPMax ||
		params.HeatSetpoint < deviceInfo.TempSPMin || params.HeatSetpoint > deviceInfo.TempSPMax {
		return errors.New("setpoint(s) outside of allowable range")
	}

	data := map[string]interface{}{
		"cspHome":       params.CoolSetpoint,
		"hspHome":       params.HeatSetpoint,
		"schedOverride": 1,
	}

	json, err := json.Marshal(data)
	if err != nil {
		return errors.New("json marshal failed")
	}

	log.Println(string(json[:]))

	return d.updateDevice(deviceId, json)
}

func (d *Daikin) UpdateDeviceRaw(deviceId string, json string) error {
	return d.updateDevice(deviceId, []byte(json))
}

func (d *Daikin) updateDevice(deviceId string, body []byte) error {

	r, err := http.NewRequest("PUT", d.urlBase+"/deviceData/"+deviceId, bytes.NewBuffer(body))
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
